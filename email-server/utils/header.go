package utils

import (
	"bytes"

	"github.com/emersion/go-message/mail"
)

// ExtractHeader 读取邮件头中的第一个指定字段，解析失败或不存在时返回空串。
func ExtractHeader(raw []byte, key string) string {
	reader, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	defer reader.Close()
	return reader.Header.Get(key)
}
