package service

import (
	"bytes"
	"email-server/config"
	"email-server/model"
	"email-server/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jhillyerd/enmime"
)

func TestLocalEmlParsing(t *testing.T) {
	emlPath := "../static/test.eml"
	email := "test@example.com"
	folder := "INBOX"
	uid := int64(999)

	_, err := ParseMailFromLocalEml(emlPath, email, folder, uid)
	if err != nil {
		t.Fatalf("测试失败: %v", err)
	}
}

// ParseMailFromLocalEml 使用本地 EML 文件测试邮件解析
func ParseMailFromLocalEml(emlFilePath string, email string, folder string, uid int64) (*model.MailDetail, error) {
	// 读取本地 EML 文件
	rawData, err := os.ReadFile(emlFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取 EML 文件失败: %w", err)
	}

	// 解析邮件
	env, err := enmime.ReadEnvelope(bytes.NewReader(rawData))
	if err != nil {
		return nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	fmt.Printf("✅ 邮件解析成功\n")

	// 获取邮件正文
	content := env.HTML
	if content == "" {
		content = env.Text
	}

	//	cid内容
	cidMap := make(map[string]string)

	// 构建附件列表
	var attachments []model.AttachmentInfo
	for idx, att := range env.Attachments {
		if att.Disposition == "attachment" {
			filetype := strings.Split(att.FileName, ".")[1]
			size := int64(len(att.Content))
			attachments = append(attachments, model.AttachmentInfo{
				PartID:      att.PartID,
				FileName:    att.FileName,
				ContentType: att.ContentType,
				FileType:    strings.ToLower(filetype),
				Size:        utils.FormatUnitSize(size),
			})
		} else {
			contentID := strings.Trim(att.Header.Get("Content-Id"), "<>")
			if contentID != "" && len(att.Content) > 0 {
				// 生成本地文件路径
				fileName := att.FileName
				ext := filepath.Ext(fileName)
				nameWithoutExt := strings.TrimSuffix(fileName, ext)
				fileName = fmt.Sprintf("%s_%d%s", nameWithoutExt, idx, ext)

				// 保存到静态资源目录
				staticDir := filepath.Join("static", "images", email, folder, fmt.Sprint(uid))
				if err := os.MkdirAll(staticDir, 0755); err != nil {
					fmt.Printf("创建静态目录失败: %v\n", err)
					continue
				}

				localPath := filepath.Join(staticDir, fileName)
				if err := os.WriteFile(localPath, att.Content, 0644); err != nil {
					fmt.Printf("保存内联图片失败: %v\n", err)
					continue
				}

				// 构建 HTTP 访问 URL
				serverHost := config.GetConfig("mail.server.host")
				serverPort := config.GetConfig("mail.server.port")
				imageURL := fmt.Sprintf("http://%s:%s/static/images/%s/%s/%d/%s",
					serverHost, serverPort, email, folder, uid, fileName)

				cidMap[contentID] = imageURL
			}
		}
	}

	// 处理内联图片
	if content != "" && len(env.Inlines) > 0 {

		for idx, inline := range env.Inlines {
			contentID := strings.Trim(inline.Header.Get("Content-Id"), "<>")
			if contentID != "" && len(inline.Content) > 0 {
				// 生成本地文件路径
				fileName := inline.FileName
				ext := filepath.Ext(fileName)
				nameWithoutExt := strings.TrimSuffix(fileName, ext)
				fileName = fmt.Sprintf("%s_%d%s", nameWithoutExt, idx, ext)

				// 保存到静态资源目录
				staticDir := filepath.Join("static", "images", email, folder, fmt.Sprint(uid))
				if err := os.MkdirAll(staticDir, 0755); err != nil {
					fmt.Printf("创建静态目录失败: %v\n", err)
					continue
				}

				localPath := filepath.Join(staticDir, fileName)
				if err := os.WriteFile(localPath, inline.Content, 0644); err != nil {
					fmt.Printf("保存内联图片失败: %v\n", err)
					continue
				}

				// 构建 HTTP 访问 URL
				serverHost := config.GetConfig("mail.server.host")
				serverPort := config.GetConfig("mail.server.port")
				imageURL := fmt.Sprintf("http://%s:%s/static/images/%s/%s/%d/%s",
					serverHost, serverPort, email, folder, uid, fileName)

				cidMap[contentID] = imageURL
			}
		}

	}

	// 批量替换 HTML 中的 cid: 引用
	for cid, imageURL := range cidMap {
		content = strings.ReplaceAll(content, "cid:"+cid, imageURL)
	}

	var totalSize int64
	for _, a := range env.Attachments {
		totalSize += int64(len(a.Content))
	}
	for _, i := range env.Inlines {
		totalSize += int64(len(i.Content))
	}

	detail := &model.MailDetail{
		Content:     content,
		Attachments: attachments,
		AttachSize:  utils.FormatUnitSize(totalSize),
	}

	return detail, nil
}
