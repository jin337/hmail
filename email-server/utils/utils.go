package utils

import (
	"email-server/config"
	"fmt"
	"net"
	"net/smtp"
	"reflect"
	"regexp"
	"strings"

	"github.com/emersion/go-imap/client"
)

// DialIMAPClient 连接IMAP服务器
func DialIMAPClient(email, password string) (*client.Client, error) {
	imapClient, err := client.Dial(fmt.Sprintf("%s:%d", config.HmailHost, config.ImapPort))
	if err != nil {
		return nil, fmt.Errorf("连接IMAP失败: %v", err)
	}

	if err := imapClient.Login(email, password); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("用户名或密码错误")
	}

	return imapClient, nil
}

// DialSMTPClient 连接SMTP服务器
func DialSMTPClient(email, password string) (*smtp.Client, error) {
	// 建立TCP连接
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.HmailHost, config.SmtpPort))
	if err != nil {
		return nil, fmt.Errorf("连接SMTP服务器失败: %w", err)
	}

	// 创建SMTP客户端
	smtpClient, err := smtp.NewClient(conn, config.HmailHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建SMTP客户端失败: %w", err)
	}

	// SMTP认证
	auth := smtp.PlainAuth("", email, password, config.HmailHost)
	if err := smtpClient.Auth(auth); err != nil {
		smtpClient.Quit()
		conn.Close()
		return nil, fmt.Errorf("SMTP认证失败: %w", err)
	}

	return smtpClient, nil
}

// containsFlag 检查标志是否在标志列表中
func ContainsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

// CleanText 清理文本中的特殊字符
func CleanText(text string) string {
	// 去除首尾空白
	text = strings.TrimSpace(text)

	// 将多个连续空白字符（包括换行符、制表符等）替换为单个空格
	reg := regexp.MustCompile(`\s+`)
	text = reg.ReplaceAllString(text, "")

	// 去除 * 号
	text = strings.ReplaceAll(text, "*", "")

	return text
}

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
		fieldName := strings.ToUpper(field[:1]) + field[1:]
		fieldVal := v.FieldByName(fieldName)

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
