package service

import (
	"fmt"
	"strings"
)

// domainOf 从邮箱地址提取域名。
func domainOf(email string) (string, error) {
	idx := strings.LastIndex(email, "@")
	if idx < 0 || idx == len(email)-1 {
		return "", fmt.Errorf("邮箱格式非法，缺少@符号")
	}
	return email[idx+1:], nil
}
