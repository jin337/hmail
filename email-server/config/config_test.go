package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	content := `
# comment
mail.server.host = "127.0.0.1"
mail.server.port=8058
admin.password = 'a123456'
empty =
`
	vals, err := parseEnvFile(content)
	if err != nil {
		t.Fatalf("parseEnvFile 返回错误: %v", err)
	}
	if vals["mail.server.host"] != "127.0.0.1" {
		t.Errorf("host 解析错误: %q", vals["mail.server.host"])
	}
	if vals["mail.server.port"] != "8058" {
		t.Errorf("port 解析错误: %q", vals["mail.server.port"])
	}
	if vals["admin.password"] != "a123456" {
		t.Errorf("单引号值解析错误: %q", vals["admin.password"])
	}
	if vals["empty"] != "" {
		t.Errorf("空值应为空串")
	}
}

func TestParseEnvFileInvalidLine(t *testing.T) {
	if _, err := parseEnvFile("no equal sign"); err == nil {
		t.Fatal("缺少等号时应返回错误")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	content := `
mail.server.host = "10.0.0.8"
mail.server.port = 9000
imap.port = 1993
smtp.port = 1587
admin.password = "secret"
jwt.secret = "jwt-secret"
jwt.life.web.span = 12
gin.mode = "debug"
static.dir = "public"
contacts.db.path = "data/contacts"
server.max.upload = 30
cors.origins = "http://localhost:5173, http://localhost:4055"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	if cfg.Server.Host != "10.0.0.8" || cfg.Server.Port != 9000 {
		t.Errorf("server 配置错误: %+v", cfg.Server)
	}
	if cfg.IMAP.Port != 1993 || cfg.SMTP.Port != 1587 {
		t.Errorf("端口配置错误: imap=%d smtp=%d", cfg.IMAP.Port, cfg.SMTP.Port)
	}
	if cfg.IMAP.Host != "10.0.0.8" || cfg.SMTP.Host != "10.0.0.8" {
		t.Errorf("IMAP/SMTP 主机应跟随 mail.server.host")
	}
	if cfg.APIPrefix != "/api" {
		t.Errorf("debug 模式下前缀应为 /api，实际 %q", cfg.APIPrefix)
	}
	if cfg.JWT.LifeHours != 12 || cfg.JWT.Secret != "jwt-secret" {
		t.Errorf("jwt 配置错误: %+v", cfg.JWT)
	}
	if cfg.StaticDir != "public" || cfg.ContactsPath != "data/contacts" {
		t.Errorf("存储配置错误: %q %q", cfg.StaticDir, cfg.ContactsPath)
	}
	if cfg.MaxUploadMB != 30 {
		t.Errorf("上传限制错误: %d", cfg.MaxUploadMB)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[1] != "http://localhost:4055" {
		t.Errorf("CORS 配置错误: %v", cfg.CORSOrigins)
	}

	if Current().Server.Port != 9000 {
		t.Errorf("Current 未返回最新配置")
	}
}

func TestReleaseModePrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	if err := os.WriteFile(path, []byte(`gin.mode = "release"`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPrefix != "/" {
		t.Errorf("release 模式前缀应为 /，实际 %q", cfg.APIPrefix)
	}
}

func TestExplicitPrefixAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	if err := os.WriteFile(path, []byte("gin.mode = \"debug\"\napi.prefix = \"api\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MAIL_SERVER_PORT", "7777")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPrefix != "/api" {
		t.Errorf("显式前缀未生效: %q", cfg.APIPrefix)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("环境变量未覆盖端口: %d", cfg.Server.Port)
	}
}

func TestInvalidValues(t *testing.T) {
	cases := []string{
		`gin.mode = "staging"`,
		"mail.server.port = abc",
		"imap.port = 70000",
		"jwt.life.web.span = 0",
		"server.max.upload = -1",
	}
	for _, content := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.env")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("非法配置应报错: %s", content)
		}
	}
}

func TestMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "not-exist.env"))
	if err != nil {
		t.Fatalf("文件不存在时应使用默认值: %v", err)
	}
	if cfg.Server.Port != 8058 || cfg.IMAP.Port != 993 || cfg.SMTP.Port != 587 {
		t.Errorf("默认端口错误: %+v %+v %+v", cfg.Server, cfg.IMAP, cfg.SMTP)
	}
}
