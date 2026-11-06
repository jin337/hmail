package mailbox

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPSender 通过 hMailServer 的 SMTP 端口逐封投递邮件。
type SMTPSender struct {
	Host string
	Port string
}

// NewSMTPSender 构造 SMTP 投递器。
func NewSMTPSender(host string, port int) *SMTPSender {
	return &SMTPSender{Host: host, Port: fmt.Sprint(port)}
}

// Send 建立一次 SMTP 会话投递一封邮件，行为与原实现保持一致：
// 先认证，再依次设置发件人、收件人、抄送，最后写入原始报文。
func (s *SMTPSender) Send(from, password string, to, cc []string, raw []byte) error {
	address := net.JoinHostPort(s.Host, s.Port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %w", err)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}

	auth := smtp.PlainAuth("", from, password, s.Host)
	if err := client.Auth(auth); err != nil {
		_ = client.Quit()
		_ = conn.Close()
		return fmt.Errorf("SMTP认证失败: %w", err)
	}

	if err := client.Mail(from); err != nil {
		_ = client.Quit()
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	recipients := append(append([]string{}, to...), cc...)
	for _, addr := range recipients {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if err := client.Rcpt(addr); err != nil {
			_ = client.Quit()
			return fmt.Errorf("设置收件人 %s 失败: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		_ = client.Quit()
		return fmt.Errorf("获取邮件数据失败: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		_ = w.Close()
		_ = client.Quit()
		return fmt.Errorf("写入邮件数据失败: %w", err)
	}
	if err := w.Close(); err != nil {
		_ = client.Quit()
		return fmt.Errorf("关闭邮件数据失败: %w", err)
	}

	return client.Quit()
}
