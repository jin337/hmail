// Package config 负责加载并集中管理服务配置。
//
// 配置来源优先级（后者覆盖前者）：
//  1. 代码内置默认值；
//  2. config/config.env 文件（键名与原项目保持一致，兼容旧配置）；
//  3. 进程环境变量（键名由配置项自动映射为大写下划线形式，例如 mail.server.host -> MAIL_SERVER_HOST）。
//
// 通过 Load 读取配置后，使用 Current 取得不可变副本。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// 标准邮件文件夹。
const (
	FolderInbox   = "INBOX"
	FolderSent    = "Sent"
	FolderDrafts  = "Drafts"
	FolderDeleted = "Deleted"
	FolderJunk    = "Junk"
)

// DefaultFolders 是登录/建号时需要保证存在的标准文件夹。
var DefaultFolders = []string{
	FolderInbox,
	FolderSent,
	FolderDrafts,
	FolderDeleted,
	FolderJunk,
}

// FolderDrafts 返回草稿箱名称。
func (c Config) FolderDrafts() string { return FolderDrafts }

// FolderSent 返回已发送文件夹名称。
func (c Config) FolderSent() string { return FolderSent }

// MailServer 描述 WebMail/IMAP/SMTP 所在的主机与端口。
type MailServer struct {
	Host string
	Port int
}

// JWT 描述令牌签名与有效期配置。
type JWT struct {
	Secret    string
	LifeHours int
}

// Config 是整个服务运行期依赖的配置集合。
type Config struct {
	Mode          string // gin.mode: debug/release
	APIPrefix     string // 路由前缀，debug 下默认 /api，release 下默认 /
	Server        MailServer
	IMAP          MailServer
	SMTP          MailServer
	AdminPassword string
	JWT           JWT

	ContactsPath string // 联系人 KV 数据库目录
	StaticDir    string // 静态资源目录（头像、内联图片）
	MaxUploadMB  int64  // 单次请求最大上传体积
	CORSOrigins  []string
	ConfigPath   string
}

// DefaultConfig 返回与仓库自带 config.env 等价的内置默认配置。
func DefaultConfig() Config {
	return Config{
		Mode: "debug",
		Server: MailServer{
			Host: "127.0.0.1",
			Port: 8058,
		},
		IMAP: MailServer{
			Host: "127.0.0.1",
			Port: 993,
		},
		SMTP: MailServer{
			Host: "127.0.0.1",
			Port: 587,
		},
		AdminPassword: "a123456",
		JWT: JWT{
			Secret:    "email-server-hmail-20260525",
			LifeHours: 24,
		},
		ContactsPath: "cache/contact_db",
		StaticDir:    "static",
		MaxUploadMB:  20,
		CORSOrigins:  []string{"*"},
		APIPrefix:    "/api",
	}
}

// values 保存解析后的键值对，便于按键名查询。
type values map[string]string

var (
	current   Config
	currentMu sync.RWMutex
	loaded    bool
)

// Load 读取配置：先取默认值，再合并 config.env，最后合并环境变量。
// path 为空时依次尝试 EMAIL_SERVER_CONFIG、./config/config.env。
func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	vals := values{}

	filePath := path
	if filePath == "" {
		filePath = os.Getenv("EMAIL_SERVER_CONFIG")
	}
	if filePath == "" {
		filePath = filepath.Join(".", "config", "config.env")
	}
	// 文件不存在时允许仅使用默认值 + 环境变量，便于容器化部署。
	if data, err := os.ReadFile(filePath); err == nil {
		parsed, perr := parseEnvFile(string(data))
		if perr != nil {
			return cfg, fmt.Errorf("解析配置文件 %s 失败: %w", filePath, perr)
		}
		for k, v := range parsed {
			vals[k] = v
		}
		cfg.ConfigPath = filePath
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("读取配置文件 %s 失败: %w", filePath, err)
	}

	// 环境变量覆盖（键名转换规则见包注释）。
	for k := range allKeys {
		if v, ok := os.LookupEnv(envName(k)); ok && v != "" {
			vals[k] = v
		}
	}

	if err := apply(&cfg, vals); err != nil {
		return cfg, err
	}

	currentMu.Lock()
	current = cfg
	loaded = true
	currentMu.Unlock()

	return cfg, nil
}

// Current 返回最近一次成功加载的配置副本。
func Current() Config {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}

// Get 按原始键名读取配置值（兼容旧代码中的 config.GetConfig 调用习惯）。
func Get(key string) string {
	cfg := Current()
	switch key {
	case "mail.server.host":
		return cfg.Server.Host
	case "mail.server.port":
		return strconv.Itoa(cfg.Server.Port)
	case "smtp.port":
		return strconv.Itoa(cfg.SMTP.Port)
	case "imap.port":
		return strconv.Itoa(cfg.IMAP.Port)
	case "admin.password":
		return cfg.AdminPassword
	case "jwt.secret":
		return cfg.JWT.Secret
	case "jwt.life.web.span":
		return strconv.Itoa(cfg.JWT.LifeHours)
	case "gin.mode":
		return cfg.Mode
	case "api.prefix":
		return cfg.APIPrefix
	case "static.dir":
		return cfg.StaticDir
	case "contacts.db.path":
		return cfg.ContactsPath
	default:
		return ""
	}
}

// IsLoaded 用于测试判断配置是否已经初始化。
func IsLoaded() bool {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return loaded
}

var allKeys = map[string]struct{}{
	"mail.server.host":  {},
	"mail.server.port":  {},
	"smtp.port":         {},
	"imap.port":         {},
	"admin.password":    {},
	"jwt.secret":        {},
	"jwt.life.web.span": {},
	"gin.mode":          {},
	"api.prefix":        {},
	"static.dir":        {},
	"contacts.db.path":  {},
	"server.max.upload": {},
	"cors.origins":      {},
}

// envName 把点分键名映射为大写下划线形式：mail.server.host -> MAIL_SERVER_HOST。
func envName(key string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(key))
}

// parseEnvFile 解析 key = "value" 形式的配置文件，支持 # 注释与空行。
func parseEnvFile(content string) (values, error) {
	vals := values{}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			return nil, fmt.Errorf("第 %d 行缺少 '=': %s", i+1, line)
		}
		key := strings.TrimSpace(line[:eq])
		raw := strings.TrimSpace(line[eq+1:])
		if key == "" {
			return nil, fmt.Errorf("第 %d 行键名为空", i+1)
		}
		// 去掉成对引号；未闭合时按原样保留，避免误伤密码中的引号。
		if len(raw) >= 2 {
			if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
				raw = raw[1 : len(raw)-1]
			}
		}
		vals[key] = raw
	}
	return vals, nil
}

func apply(cfg *Config, vals values) error {
	if v := vals["gin.mode"]; v != "" {
		if v != "debug" && v != "release" && v != "test" {
			return fmt.Errorf("gin.mode 仅支持 debug/release/test，当前为 %q", v)
		}
		cfg.Mode = v
	}

	// 路由前缀默认跟随运行模式，允许显式覆盖。
	cfg.APIPrefix = "/"
	if cfg.Mode == "debug" {
		cfg.APIPrefix = "/api"
	}
	if v := vals["api.prefix"]; v != "" {
		if !strings.HasPrefix(v, "/") {
			v = "/" + v
		}
		cfg.APIPrefix = strings.TrimSuffix(v, "/")
		if cfg.APIPrefix == "" {
			cfg.APIPrefix = "/"
		}
	}

	cfg.Server.Host = firstNonEmpty(vals["mail.server.host"], cfg.Server.Host)
	cfg.IMAP.Host = cfg.Server.Host
	cfg.SMTP.Host = cfg.Server.Host

	var err error
	if cfg.Server.Port, err = intValue(vals, "mail.server.port", cfg.Server.Port); err != nil {
		return err
	}
	if cfg.IMAP.Port, err = intValue(vals, "imap.port", cfg.IMAP.Port); err != nil {
		return err
	}
	if cfg.SMTP.Port, err = intValue(vals, "smtp.port", cfg.SMTP.Port); err != nil {
		return err
	}
	if cfg.JWT.LifeHours, err = intValue(vals, "jwt.life.web.span", cfg.JWT.LifeHours); err != nil {
		return err
	}
	if cfg.JWT.LifeHours <= 0 {
		return fmt.Errorf("jwt.life.web.span 必须为正整数")
	}

	cfg.AdminPassword = firstNonEmpty(vals["admin.password"], cfg.AdminPassword)
	cfg.JWT.Secret = firstNonEmpty(vals["jwt.secret"], cfg.JWT.Secret)
	cfg.StaticDir = firstNonEmpty(vals["static.dir"], cfg.StaticDir)
	cfg.ContactsPath = firstNonEmpty(vals["contacts.db.path"], cfg.ContactsPath)

	if cfg.MaxUploadMB, err = int64Value(vals, "server.max.upload", cfg.MaxUploadMB); err != nil {
		return err
	}
	if cfg.MaxUploadMB <= 0 {
		return fmt.Errorf("server.max.upload 必须为正整数")
	}

	if v := vals["cors.origins"]; v != "" {
		parts := strings.Split(v, ",")
		origins := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				origins = append(origins, p)
			}
		}
		cfg.CORSOrigins = origins
	}

	return nil
}

func firstNonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func intValue(vals values, key string, def int) (int, error) {
	v := strings.TrimSpace(vals[key])
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("%s 必须是整数，当前为 %q", key, v)
	}
	if n <= 0 || n > 65535 {
		return def, fmt.Errorf("%s 必须在 1-65535 之间，当前为 %d", key, n)
	}
	return n, nil
}

func int64Value(vals values, key string, def int64) (int64, error) {
	v := strings.TrimSpace(vals[key])
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def, fmt.Errorf("%s 必须是整数，当前为 %q", key, v)
	}
	return n, nil
}
