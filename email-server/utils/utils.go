package utils

import (
	"email-server/model"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidateRequiredParams 校验必填参数
func ValidateRequiredParams(fields []string, obj interface{}) error {
	// 使用反射获取对象的字段值
	v := reflect.ValueOf(obj)

	// 如果是指针，获取其指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 只支持结构体
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("参数结构错误: 只支持Object类型")
	}

	for _, field := range fields {
		// 查找对应字段（首字母大写）
		fieldVal := v.FieldByName(field)

		if !fieldVal.IsValid() {
			return fmt.Errorf("字段 %s 不存在", field)
		}

		// 检查字段值是否为空
		if fieldVal.Kind() == reflect.String {
			if strings.TrimSpace(fieldVal.String()) == "" {
				return fmt.Errorf("%s 参数不能为空", field)
			}
		}
	}
	return nil
}

// FormatMailName 解析邮箱
func FormatMailName(mailStr string) (*string, []*model.MailInfo, error) {
	if mailStr == "" {
		return nil, nil, nil
	}

	// 标准库直接解析（自动处理中文、编码、格式）
	addresses, err := mail.ParseAddressList(mailStr)
	if err != nil {
		return nil, nil, err
	}

	mailList := make([]string, 0, len(addresses))
	infoList := make([]*model.MailInfo, 0, len(addresses))

	for _, addr := range addresses {
		mailList = append(mailList, addr.Address)

		name := addr.Name
		if name == "" {
			addname := strings.Split(addr.Address, "@")
			name = addname[0]
		}

		infoList = append(infoList, &model.MailInfo{
			Name:  name,
			Email: addr.Address,
		})
	}
	str := strings.Join(mailList, ", ")

	return &str, infoList, nil
}

// FormatUnitSize 字节 转为 友好单位 B/KB/MB/GB
func FormatUnitSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(size)/1024/1024)
	} else {
		return fmt.Sprintf("%.1fGB", float64(size)/1024/1024/1024)
	}
}

// Formatize 反向：可读字符串转字节总数
func Formatize(size string) int64 {
	// 正则匹配数字+单位
	re := regexp.MustCompile(`^([0-9.]+)(B|KB|MB|GB)$`)
	match := re.FindStringSubmatch(strings.TrimSpace(size))
	if len(match) != 3 {
		return 0
	}

	numStr, unit := match[1], match[2]
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	if val < 0 {
		return 0
	}

	var bytes int64
	switch unit {
	case "B":
		bytes = int64(val)
	case "KB":
		bytes = int64(val * 1024)
	case "MB":
		bytes = int64(val * 1024 * 1024)
	case "GB":
		bytes = int64(val * 1024 * 1024 * 1024)
	default:
		return 0
	}
	return bytes
}

// FormatDate 解析时间，格式化为 2006-01-02 15:04:05
func FormatDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("时间为空")
	}

	if t, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
		return t.Local(), nil
	}

	if t, err := time.Parse(time.RFC1123, dateStr); err == nil {
		return t.Local(), nil
	}

	return time.Time{}, fmt.Errorf("不支持的日期格式: %s", dateStr)
}

// HasAvatar 判断用户是否上传了头像
func HasAvatar(accountList []string) error {
	// 定义目录与默认头像路径
	avatarDir := "static/avatars"
	defaultAvatarPath := filepath.Join(avatarDir, "default.webp")

	// 检查头像文件夹是否存在，不存在则创建
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return fmt.Errorf("创建头像目录失败: %w", err)
	}

	// 校验默认头像文件是否存在，不存在直接报错
	_, err := os.Stat(defaultAvatarPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("默认头像 default.webp 不存在，请检查文件")
		}
		return fmt.Errorf("读取默认头像状态失败: %w", err)
	}
	// 遍历所有账号处理头像
	for _, account := range accountList {
		// 当前账号头像完整路径
		userAvatarPath := filepath.Join(avatarDir, fmt.Sprintf("%s.webp", account))

		// 判断用户头像是否存在
		stat, err := os.Stat(userAvatarPath)
		if err != nil {
			// 文件不存在，复制默认头像
			if os.IsNotExist(err) {
				if err := copyFile(defaultAvatarPath, userAvatarPath); err != nil {
					return fmt.Errorf("账号[%s]复制默认头像失败: %w", account, err)
				}
				continue
			}
			// 其他错误（权限、读取失败等）直接返回
			return fmt.Errorf("检查账号[%s]头像状态失败: %w", account, err)
		}

		// 存在但为文件夹，异常报错
		if stat.IsDir() {
			return fmt.Errorf("账号[%s]头像路径是文件夹，非法", account)
		}
	}

	return nil
}

// copyFile 文件复制工具函数
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件，权限0644
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 拷贝文件内容
	_, err = io.Copy(dstFile, srcFile)
	return err
}
