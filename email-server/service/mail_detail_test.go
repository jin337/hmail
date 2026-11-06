package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"email-server/internal/testutil"
)

func TestMailDetailInlineImageAndAttachments(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Subject: "带图片的邮件",
		HTML:    `<p>hi</p><img src="cid:logo@example.com">`,
		Text:    "hi",
		Attachments: []testutil.Attachment{
			{FileName: "报表.xlsx", ContentType: "application/vnd.ms-excel", Content: []byte("excel-data")},
			{FileName: "logo.png", ContentType: "image/png", Content: []byte("png-data"), Inline: true, CID: "logo@example.com"},
		},
	})
	uid := server.AddMessage("INBOX", raw)

	detail, err := svc.MailDetail("user@example.com", "pwd", "INBOX", int64(uid), "mail.example.com")
	if err != nil {
		t.Fatalf("MailDetail 失败: %v", err)
	}

	// cid 图片被改写为带 data-href 的绝对地址，src 保持原样
	if !strings.Contains(detail.Content, `src="cid:logo@example.com"`) {
		t.Errorf("正文应保留 cid 引用: %s", detail.Content)
	}
	if !strings.Contains(detail.Content, `data-href="http://mail.example.com/api/viewfile?url=`) {
		t.Errorf("data-href 生成错误: %s", detail.Content)
	}
	if !strings.Contains(detail.Content, "image_3.png") {
		t.Errorf("data-href 未指向落盘图片: %s", detail.Content)
	}
	if strings.Contains(detail.Content, `url=C:\`) {
		t.Errorf("URL 不应包含 Windows 反斜杠: %s", detail.Content)
	}

	// 图片确实落盘
	dir := filepath.Join(svc.Config().StaticDir, "images", "user@example.com", "1")
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("内联图片目录不存在: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("应落盘 1 张内联图片，实际 %d", len(files))
	}

	if len(detail.Attachments) != 1 {
		t.Fatalf("应只有 1 个附件，实际 %d", len(detail.Attachments))
	}
	if detail.Attachments[0].FileName != "报表.xlsx" || detail.Attachments[0].FileType != "xlsx" {
		t.Errorf("附件解析错误: %+v", detail.Attachments[0])
	}
	if detail.AttachSize == "" {
		t.Error("附件总大小不应为空")
	}
}

func TestMailDetailWithoutExtension(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Attachments: []testutil.Attachment{{FileName: "README", ContentType: "text/plain", Content: []byte("hello")}},
	})
	uid := server.AddMessage("INBOX", raw)

	detail, err := svc.MailDetail("user@example.com", "pwd", "INBOX", int64(uid), "host")
	if err != nil {
		t.Fatalf("无扩展名附件不应 panic/报错: %v", err)
	}
	if detail.Attachments[0].FileType != "" {
		t.Errorf("无扩展名应为空，实际 %q", detail.Attachments[0].FileType)
	}
}

func TestMailDetailNotFound(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	if _, err := svc.MailDetail("user@example.com", "pwd", "INBOX", 999, "host"); err == nil {
		t.Fatal("不存在的 UID 应返回错误")
	}
}

func TestExtFromContentType(t *testing.T) {
	cases := map[string]string{
		"image/png":     ".png",
		"image/gif":     ".gif",
		"image/jpeg":    ".jpg",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"application/x": ".png",
	}
	for ct, want := range cases {
		if got := extFromContentType(ct); got != want {
			t.Errorf("extFromContentType(%q)=%q, want %q", ct, got, want)
		}
	}
}

func TestSafePathSegment(t *testing.T) {
	got := safePathSegment("../a/b\\c")
	if strings.ContainsAny(got, "/\\") || strings.Contains(got, "..") {
		t.Errorf("路径分隔符未被清理: %q", got)
	}
}
