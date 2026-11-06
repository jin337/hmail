package utils

import (
	"fmt"
	"strings"
	"time"
)

// 邮件场景常见的时间格式，按匹配优先级排列。
var dateLayouts = []string{
	time.RFC1123Z,               // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC1123,                // Mon, 02 Jan 2006 15:04:05 UTC
	time.RFC3339,                // 2006-01-02T15:04:05+08:00
	"2006-01-02 15:04:05",       // 本地无时区
	"Mon, 02 Jan 2006 15:04:05", // 缺失时区后缀的 RFC1123
	time.RFC822Z,                // 02 Jan 06 15:04 -0700
	time.RFC822,                 // 02 Jan 06 15:04 MST
}

// ParseMailDate 解析邮件头中的日期，统一返回本地时区时间。
func ParseMailDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("时间字符串为空")
	}

	var parseErr error
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t.Local(), nil
		}
		parseErr = err
	}
	return time.Time{}, fmt.Errorf("不支持的日期格式[%s], 最后解析错误: %w", dateStr, parseErr)
}

// FormatDate 是 ParseMailDate 的兼容别名，便于旧调用方迁移。
func FormatDate(dateStr string) (time.Time, error) {
	return ParseMailDate(dateStr)
}

// FormatDateTime 输出账号列表使用的时间格式。
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
