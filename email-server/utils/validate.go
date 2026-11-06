package utils

import (
	"fmt"
	"strings"
)

// Field 表示一个"名称 + 值"的必填项，替代原有的反射校验。
type Field struct {
	Name  string
	Value string
}

// Required 构造一个必填字符串字段。
func Required(name, value string) Field {
	return Field{Name: name, Value: value}
}

// CheckRequired 按顺序校验必填字符串，错误消息与原有接口一致：
// "{字段名} 参数不能为空"。
func CheckRequired(fields ...Field) error {
	for _, f := range fields {
		if strings.TrimSpace(f.Value) == "" {
			return fmt.Errorf("%s 参数不能为空", f.Name)
		}
	}
	return nil
}

// FileExtension 返回小写扩展名（不含点）；没有扩展名时返回空串。
func FileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx < 0 || idx == len(filename)-1 {
		return ""
	}
	return strings.ToLower(filename[idx+1:])
}

// SplitAddressList 把逗号分隔的地址串拆分为去空白的地址列表。
// 空串返回 nil，保持与原接口一致（用于判断收件人是否为空）。
func SplitAddressList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
