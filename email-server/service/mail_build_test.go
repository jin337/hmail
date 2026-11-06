package service

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"email-server/internal/testutil"

	"github.com/jhillyerd/enmime"
)

// fileHeaders 通过真实的 multipart 解析得到 *multipart.FileHeader。
func fileHeaders(t *testing.T, files map[string][]byte) []*multipart.FileHeader {
	t.Helper()

	buf, writer := testutil.MultipartWriter()
	for name, content := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/mail/send", buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["files"]
}

func TestBuildRawEmail(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	comp := Compose{
		From:       "user@example.com",
		To:         []string{"a@example.com", "b@example.com"},
		Cc:         []string{"c@example.com"},
		Subject:    "项目周报",
		Body:       `<p>本周进展</p><img src="data:image/png;base64,iVBORw0KGgo=">`,
		InReplyTo:  "<in@example.com>",
		References: "<root@example.com>",
		Schedule:   "2030-01-02 03:04:05",
		Files:      fileHeaders(t, map[string][]byte{"说明.txt": []byte("attachment-body")}),
	}

	raw, err := svc.BuildRawEmail(comp)
	if err != nil {
		t.Fatalf("BuildRawEmail 失败: %v", err)
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("生成的邮件无法解析: %v", err)
	}

	if env.GetHeader("Subject") != "项目周报" {
		t.Errorf("主题错误: %q", env.GetHeader("Subject"))
	}
	// 发件人显示名来自账号目录
	if from := env.GetHeader("From"); !strings.Contains(from, "张三") || !strings.Contains(from, "user@example.com") {
		t.Errorf("From 头错误: %q", from)
	}
	if to := env.GetHeader("To"); !strings.Contains(to, "a@example.com") || !strings.Contains(to, "b@example.com") {
		t.Errorf("To 头错误: %q", to)
	}
	if cc := env.GetHeader("Cc"); !strings.Contains(cc, "c@example.com") {
		t.Errorf("Cc 头错误: %q", cc)
	}
	if env.GetHeader("In-Reply-To") != "<in@example.com>" {
		t.Errorf("In-Reply-To 头错误: %q", env.GetHeader("In-Reply-To"))
	}
	if env.GetHeader("References") != "<root@example.com>" {
		t.Errorf("References 头错误: %q", env.GetHeader("References"))
	}

	schedule := env.GetHeader("X-Schedule-Send")
	if schedule == "" {
		t.Fatal("缺少 X-Schedule-Send 头")
	}

	if len(env.Inlines) != 1 {
		t.Errorf("data-uri 图片应转为 1 个内联附件，实际 %d", len(env.Inlines))
	}
	if len(env.Attachments) != 1 || env.Attachments[0].FileName != "说明.txt" {
		t.Errorf("上传附件解析错误: %+v", env.Attachments)
	}
	if string(env.Attachments[0].Content) != "attachment-body" {
		t.Errorf("附件内容错误: %q", env.Attachments[0].Content)
	}

	messageID := env.GetHeader("Message-Id")
	if !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, "@example.com>") {
		t.Errorf("Message-ID 格式错误: %q", messageID)
	}

	if env.Text == "" {
		t.Error("应生成 text/plain 备选正文")
	}
}

func TestBuildRawEmailWithoutScheduleAndCc(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	raw, err := svc.BuildRawEmail(Compose{
		From:    "user@example.com",
		To:      []string{"a@example.com"},
		Subject: "hi",
		Body:    "<p>hello</p>",
	})
	if err != nil {
		t.Fatal(err)
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if env.GetHeader("X-Schedule-Send") != "" {
		t.Error("未设置定时不应写入 X-Schedule-Send")
	}
	if env.GetHeader("Cc") != "" {
		t.Error("无抄送不应写入 Cc")
	}
	if !strings.Contains(env.HTML, "hello") {
		t.Errorf("HTML 正文丢失: %q", env.HTML)
	}
}

func TestParseLocalSchedule(t *testing.T) {
	tm, err := parseLocalSchedule("2030-01-02 03:04:05")
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2030 || tm.Hour() != 3 {
		t.Errorf("本地时间解析错误: %v", tm)
	}

	tm, err = parseLocalSchedule("2030-01-02T03:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2030 {
		t.Errorf("RFC3339 解析错误: %v", tm)
	}

	if _, err := parseLocalSchedule("bad"); err == nil {
		t.Error("非法时间应报错")
	}
}

func TestEncodeBase64Lines(t *testing.T) {
	long := strings.Repeat("a", 200)
	encoded := encodeBase64Lines([]byte(long))
	lines := strings.Split(strings.TrimSuffix(encoded, "\r\n"), "\r\n")
	for _, line := range lines {
		if len(line) > 76 {
			t.Errorf("每行不应超过 76 字符: %d", len(line))
		}
	}
}
