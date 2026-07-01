package utils

import (
	"email-server/model"
	"fmt"
	"net/mail"
	"reflect"
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

// FormatSize 字节 转为 友好单位 B/KB/MB/GB
func FormatSize(size uint32) string {
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
