package testutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
	"time"
)

// Attachment 描述测试邮件中的附件。
type Attachment struct {
	FileName    string
	ContentType string
	Content     []byte
	Inline      bool
	CID         string
}

// MessageOptions 描述一封测试邮件。
type MessageOptions struct {
	From        string
	To          string
	Cc          string
	Subject     string
	Text        string
	HTML        string
	Date        time.Time
	MessageID   string
	Schedule    string
	InReplyTo   string
	References  string
	Attachments []Attachment
	Flags       []string
}

// BuildMessage 生成一封可被 enmime 解析的 MIME 邮件。
func BuildMessage(o MessageOptions) []byte {
	buf := &bytes.Buffer{}

	if o.From == "" {
		o.From = "sender@example.com"
	}
	if o.To == "" {
		o.To = "receiver@example.com"
	}
	if o.MessageID == "" {
		o.MessageID = fmt.Sprintf("<%d@example.com>", time.Now().UnixNano())
	}
	if o.Date.IsZero() {
		o.Date = time.Now()
	}
	if o.Text == "" && o.HTML == "" {
		o.Text = "hello world"
	}

	fmt.Fprintf(buf, "From: %s\r\n", o.From)
	fmt.Fprintf(buf, "To: %s\r\n", o.To)
	if o.Cc != "" {
		fmt.Fprintf(buf, "Cc: %s\r\n", o.Cc)
	}
	fmt.Fprintf(buf, "Subject: %s\r\n", o.Subject)
	fmt.Fprintf(buf, "Date: %s\r\n", o.Date.Format(time.RFC1123Z))
	fmt.Fprintf(buf, "Message-ID: %s\r\n", o.MessageID)
	if o.InReplyTo != "" {
		fmt.Fprintf(buf, "In-Reply-To: %s\r\n", o.InReplyTo)
	}
	if o.References != "" {
		fmt.Fprintf(buf, "References: %s\r\n", o.References)
	}
	if o.Schedule != "" {
		fmt.Fprintf(buf, "X-Schedule-Send: %s\r\n", o.Schedule)
	}

	boundary := "test-boundary-0001"
	fmt.Fprintf(buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary)

	altBoundary := "test-alt-0001"
	fmt.Fprintf(buf, "--%s\r\n", boundary)
	fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

	if o.Text != "" {
		fmt.Fprintf(buf, "--%s\r\n", altBoundary)
		fmt.Fprintf(buf, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
		fmt.Fprintf(buf, "Content-Transfer-Encoding: base64\r\n\r\n")
		fmt.Fprintf(buf, "%s\r\n", base64.StdEncoding.EncodeToString([]byte(o.Text)))
	}
	if o.HTML != "" {
		fmt.Fprintf(buf, "--%s\r\n", altBoundary)
		fmt.Fprintf(buf, "Content-Type: text/html; charset=\"utf-8\"\r\n")
		fmt.Fprintf(buf, "Content-Transfer-Encoding: base64\r\n\r\n")
		fmt.Fprintf(buf, "%s\r\n", base64.StdEncoding.EncodeToString([]byte(o.HTML)))
	}
	fmt.Fprintf(buf, "--%s--\r\n", altBoundary)

	partID := 2
	for _, att := range o.Attachments {
		partID++
		header := textproto.MIMEHeader{}
		contentType := att.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		header.Set("Content-Type", fmt.Sprintf("%s; name=\"%s\"", contentType, att.FileName))
		header.Set("Content-Transfer-Encoding", "base64")
		if att.Inline {
			header.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", att.FileName))
			header.Set("Content-ID", fmt.Sprintf("<%s>", att.CID))
		} else {
			header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.FileName))
		}

		fmt.Fprintf(buf, "--%s\r\n", boundary)
		for k, vs := range header {
			for _, v := range vs {
				fmt.Fprintf(buf, "%s: %s\r\n", k, v)
			}
		}
		fmt.Fprintf(buf, "\r\n%s\r\n", base64.StdEncoding.EncodeToString(att.Content))
	}
	fmt.Fprintf(buf, "--%s--\r\n", boundary)

	return buf.Bytes()
}

// MultipartWriter 便于构造 multipart 表单请求。
func MultipartWriter() (*bytes.Buffer, *multipart.Writer) {
	buf := &bytes.Buffer{}
	return buf, multipart.NewWriter(buf)
}

// JoinAddresses 用逗号连接地址，模拟前端提交格式。
func JoinAddresses(addrs []string) string {
	return strings.Join(addrs, ",")
}
