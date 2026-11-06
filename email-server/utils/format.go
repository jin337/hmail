// Package utils 提供与业务无关的纯函数工具：邮件地址解析、大小与时间格式化、
// 参数校验、文件名处理、头像文件维护。
//
// 这些函数不依赖网络与全局状态，便于单元测试。
package utils

import (
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"email-server/model"
)

// FormatMailName 解析邮件地址头，返回规范化地址串与结构化信息。
// 无法解析时返回错误，调用方可自行决定回退策略。
func FormatMailName(mailStr string) (*string, []*model.MailInfo, error) {
	if strings.TrimSpace(mailStr) == "" {
		return nil, nil, nil
	}

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
			name = strings.Split(addr.Address, "@")[0]
		}

		infoList = append(infoList, &model.MailInfo{
			Name:  name,
			Email: addr.Address,
		})
	}

	str := strings.Join(mailList, ", ")
	return &str, infoList, nil
}

// FormatUnitSize 把字节数转换为 B/KB/MB/GB 友好字符串。
func FormatUnitSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%dB", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.1fMB", float64(size)/1024/1024)
	default:
		return fmt.Sprintf("%.1fGB", float64(size)/1024/1024/1024)
	}
}

var sizePattern = regexp.MustCompile(`^([0-9.]+)(B|KB|MB|GB)$`)

// Formatize 把 B/KB/MB/GB 字符串还原为字节数；无法识别时返回 0。
func Formatize(size string) int64 {
	match := sizePattern.FindStringSubmatch(strings.TrimSpace(size))
	if len(match) != 3 {
		return 0
	}

	val, err := strconv.ParseFloat(match[1], 64)
	if err != nil || val < 0 {
		return 0
	}

	switch match[2] {
	case "B":
		return int64(val)
	case "KB":
		return int64(val * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

var (
	htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
	spacePattern   = regexp.MustCompile(`\s+`)
	starPattern    = regexp.MustCompile(`\*`)
)

// StripHTML 去除 HTML 标签、压缩空白。
func StripHTML(html string) string {
	txt := htmlTagPattern.ReplaceAllString(html, "")
	txt = strings.ReplaceAll(txt, "&nbsp;", " ")
	txt = spacePattern.ReplaceAllString(txt, " ")
	return strings.TrimSpace(txt)
}

// PreviewText 生成列表页的摘要文本：压缩空白、去掉星号，可选反转义尖括号。
func PreviewText(text string, unescape bool) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = spacePattern.ReplaceAllString(text, "")
	text = starPattern.ReplaceAllString(text, "")
	if unescape {
		text = strings.ReplaceAll(text, "&lt;", "<")
		text = strings.ReplaceAll(text, "&gt;", ">")
	}
	return text
}

// FormatFilter 把前端传入的筛选字符串转换为结构体。
func FormatFilter(filter []string) model.MailFilter {
	var f model.MailFilter
	for _, item := range filter {
		switch item {
		case "unread":
			f.Unread = true
		case "date_asc":
			f.DateAsc = true
		case "date_desc":
			f.DateDesc = true
		case "size_asc":
			f.SizeAsc = true
		case "size_desc":
			f.SizeDesc = true
		}
	}
	// 多个排序条件同时出现时按优先级只保留一个，避免排序结果不确定。
	switch {
	case f.DateAsc:
		f.DateDesc, f.SizeAsc, f.SizeDesc = false, false, false
	case f.DateDesc:
		f.SizeAsc, f.SizeDesc = false, false
	case f.SizeAsc:
		f.SizeDesc = false
	}
	return f
}

// SortMailList 根据筛选条件原地排序；未指定排序条件时保持服务端顺序。
func SortMailList(list []*model.MailItem, f model.MailFilter) {
	if len(list) <= 1 {
		return
	}

	switch {
	case f.DateAsc:
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].SendTime.Before(list[j].SendTime)
		})
	case f.DateDesc:
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].SendTime.After(list[j].SendTime)
		})
	case f.SizeAsc:
		sort.SliceStable(list, func(i, j int) bool {
			return Formatize(list[i].Size) < Formatize(list[j].Size)
		})
	case f.SizeDesc:
		sort.SliceStable(list, func(i, j int) bool {
			return Formatize(list[i].Size) > Formatize(list[j].Size)
		})
	}
}
