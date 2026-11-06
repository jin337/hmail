package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"email-server/mailbox"
	"email-server/model"
	"email-server/utils"

	"github.com/jhillyerd/enmime"
)

var cidImagePattern = regexp.MustCompile(`(?i)<img([^>]*?)src="cid:([^"]+)"([^>]*?)>`)

// MailDetail 返回邮件详情：HTML 正文、附件列表，并把内联图片落盘后改写为可访问 URL。
func (s *Service) MailDetail(email, pwd, folder string, uid int64, host string) (*model.MailDetail, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	msg, err := client.FetchUID(uint32(uid))
	if err != nil {
		if errors.Is(err, mailbox.ErrNotFound) {
			return nil, fmt.Errorf("未找到邮件 UID: %d", uid)
		}
		return nil, fmt.Errorf("获取邮件失败: %w", err)
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(msg.Raw))
	if err != nil {
		return nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	content := env.HTML
	if content == "" {
		content = env.Text
	}

	cidMap, attachments := s.collectDetailParts(email, uid, env, host)

	// 给 cid 图片追加 data-href，保留原始 src（前端据此再决定是否展示）。
	content = cidImagePattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := cidImagePattern.FindStringSubmatch(match)
		if len(sub) != 4 {
			return match
		}
		target, ok := cidMap[sub[2]]
		if !ok {
			return match
		}
		return fmt.Sprintf(`<img%s src="cid:%s" data-href="%s"%s>`, sub[1], sub[2], target, sub[3])
	})

	var totalSize int64
	for _, a := range env.Attachments {
		totalSize += int64(len(a.Content))
	}
	for _, i := range env.Inlines {
		totalSize += int64(len(i.Content))
	}

	return &model.MailDetail{
		Content:     content,
		Attachments: attachments,
		AttachSize:  utils.FormatUnitSize(totalSize),
	}, nil
}

// collectDetailParts 解析附件与内联图片。内联图片写入静态目录并记录 cid -> URL 映射。
func (s *Service) collectDetailParts(email string, uid int64, env *enmime.Envelope, host string) (map[string]string, []model.AttachmentInfo) {
	cidMap := make(map[string]string)
	attachments := make([]model.AttachmentInfo, 0, len(env.Attachments))

	dir := filepath.Join(s.cfg.StaticDir, "images", safePathSegment(email), fmt.Sprint(uid))

	for _, att := range env.Attachments {
		if att.Disposition == "attachment" {
			attachments = append(attachments, model.AttachmentInfo{
				PartID:      att.PartID,
				FileName:    att.FileName,
				ContentType: att.ContentType,
				FileType:    utils.FileExtension(att.FileName),
				Size:        utils.FormatUnitSize(int64(len(att.Content))),
			})
			continue
		}

		// 非 attachment 且带 Content-ID 的部分通常是正文内嵌图片。
		contentID := strings.Trim(att.Header.Get("Content-ID"), "<>")
		if contentID == "" || len(att.Content) == 0 {
			continue
		}
		saveInline(dir, att.PartID, filepath.Ext(att.FileName), att.Content, func(localPath, fileName string) {
			cidMap[contentID] = s.staticURL(host, localPath, fileName)
		})
	}

	for _, inline := range env.Inlines {
		contentID := strings.Trim(inline.Header.Get("Content-ID"), "<>")
		if contentID == "" || len(inline.Content) == 0 {
			continue
		}
		ext := filepath.Ext(inline.FileName)
		if ext == "" {
			ext = extFromContentType(inline.ContentType)
		}
		saveInline(dir, inline.PartID, ext, inline.Content, func(localPath, fileName string) {
			cidMap[contentID] = s.staticURL(host, localPath, fileName)
		})
	}

	return cidMap, attachments
}

// saveInline 把内联图片写入静态目录，成功后回调相对路径与文件名。
func saveInline(dir, partID, ext string, content []byte, onSaved func(localPath, fileName string)) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建静态目录失败: %v\n", err)
		return
	}

	fileName := fmt.Sprintf("image_%s%s", partID, ext)
	localPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(localPath, content, 0644); err != nil {
		fmt.Printf("保存内联图片失败: %v\n", err)
		return
	}
	onSaved(localPath, fileName)
}

// staticURL 生成前端可访问的静态资源 URL。
func (s *Service) staticURL(host, localPath, fileName string) string {
	// 统一使用 / 分隔，并剥离 static.dir 前缀，避免 Windows 反斜杠进入 URL。
	rel := filepath.ToSlash(localPath)
	base := strings.TrimPrefix(filepath.ToSlash(s.cfg.StaticDir), "./")
	base = strings.TrimSuffix(base, "/")
	rel = strings.TrimPrefix(rel, base+"/")
	return fmt.Sprintf("http://%s%s/viewfile?url=%s/%s", host, s.cfg.APIPrefix, base, rel)
}

// extFromContentType 在内联图片没有文件名时，从 MIME 类型推断扩展名。
func extFromContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ".png"
	}
}

// safePathSegment 去掉路径分隔符，避免邮箱地址被用于目录穿越。
func safePathSegment(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "_")
	return s
}
