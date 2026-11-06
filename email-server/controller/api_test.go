package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"email-server/config"
	"email-server/contacts"
	"email-server/controller"
	"email-server/internal/testutil"
	"email-server/router"
	"email-server/service"

	"github.com/gin-gonic/gin"
	"github.com/jhillyerd/enmime"
)

type apiFixture struct {
	engine *gin.Engine
	server *testutil.Server
	sender *testutil.Sender
	dir    *testutil.Directory
	cfg    config.Config
}

func newAPI(t *testing.T) *apiFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// 在临时工作目录下运行，静态目录按生产环境的相对路径 "static" 配置。
	baseDir := t.TempDir()
	t.Chdir(baseDir)
	if err := os.MkdirAll("static", 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.StaticDir = "static"
	cfg.Mode = "debug"
	cfg.APIPrefix = "/api"

	server := testutil.NewServer("user@example.com", "pwd")
	sender := &testutil.Sender{}
	dir := testutil.NewDirectory()
	dir.Add("user@example.com", "张", "三", 0)

	svc := service.New(service.Deps{
		Config:    cfg,
		IMAP:      server,
		SMTP:      sender,
		Directory: dir,
		Contacts:  contacts.NewMemoryStore(),
	})
	svc.SetAsyncRunner(func(fn func()) { fn() })

	engine := gin.New()
	router.Setup(engine, controller.New(svc, cfg), cfg)

	return &apiFixture{engine: engine, server: server, sender: sender, dir: dir, cfg: cfg}
}

func (f *apiFixture) do(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	contentType := ""

	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case *multipartBody:
		reader = bytes.NewReader(b.buf.Bytes())
		contentType = b.contentType
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
		contentType = "application/json"
	}

	req := httptest.NewRequest(method, path, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	f.engine.ServeHTTP(rec, req)
	return rec
}

type multipartBody struct {
	buf         *bytes.Buffer
	contentType string
}

func newMultipart(t *testing.T, fields map[string]string, files map[string][]byte) *multipartBody {
	t.Helper()
	buf, writer := testutil.MultipartWriter()
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
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
	return &multipartBody{buf: buf, contentType: writer.FormDataContentType()}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON: %v (%s)", err, rec.Body.String())
	}
	return body
}

func (f *apiFixture) login(t *testing.T) string {
	t.Helper()
	rec := f.do(t, "POST", "/api/login", map[string]string{"email": "user@example.com", "password": "pwd"}, "")
	body := decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("登录失败: %s", rec.Body.String())
	}
	data := body["data"].(map[string]any)
	return data["token"].(string)
}

func TestLoginAndAuth(t *testing.T) {
	f := newAPI(t)

	rec := f.do(t, "POST", "/api/login", map[string]string{"email": "user@example.com", "password": "bad"}, "")
	if body := decode(t, rec); body["code"].(float64) == 200 {
		t.Fatal("错误密码不应登录成功")
	}

	rec = f.do(t, "POST", "/api/mail/list", map[string]any{"folder": "INBOX"}, "")
	if rec.Code != 401 {
		t.Errorf("未登录应返回 401，实际 %d", rec.Code)
	}

	rec = f.do(t, "POST", "/api/mail/list", map[string]any{"folder": "INBOX"}, "Bearer garbage")
	if rec.Code != 401 {
		t.Errorf("非法 Token 应返回 401，实际 %d", rec.Code)
	}
}

func TestMailListAndDetailAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Subject: "API 测试",
		HTML:    `<p>正文</p><img src="cid:pic@x">`,
		Attachments: []testutil.Attachment{
			{FileName: "note.txt", ContentType: "text/plain", Content: []byte("note")},
			{FileName: "pic.png", ContentType: "image/png", Content: []byte("png"), Inline: true, CID: "pic@x"},
		},
	})
	f.server.AddMessage("INBOX", raw)

	rec := f.do(t, "POST", "/api/mail/list", map[string]any{"folder": "INBOX", "page": 1, "size": 10}, token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("列表失败: %s", rec.Body.String())
	}
	data := body["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Errorf("total 错误: %v", data["total"])
	}
	list := data["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("列表长度错误: %d", len(list))
	}
	if list[0].(map[string]any)["subject"] != "API 测试" {
		t.Errorf("主题错误: %v", list[0])
	}

	// 非法文件夹
	rec = f.do(t, "POST", "/api/mail/list", map[string]any{"folder": "Star"}, token)
	if body := decode(t, rec); body["code"].(float64) != 400 || body["msg"] != "folder 参数无效" {
		t.Errorf("非法文件夹应返回 400: %s", rec.Body.String())
	}

	uid := f.server.Messages("INBOX")[0].UID
	rec = f.do(t, "POST", "/api/mail/detail", map[string]any{"folder": "INBOX", "uid": uid}, token)
	body = decode(t, rec)
	detail := body["data"].(map[string]any)
	content := detail["content"].(string)
	if !bytes.Contains([]byte(content), []byte("data-href=\"http://")) {
		t.Errorf("内联图片未改写: %s", content)
	}
	attachments := detail["attachments"].([]any)
	if len(attachments) != 1 {
		t.Errorf("附件数量错误: %d", len(attachments))
	}
}

func TestSendMailAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	mp := newMultipart(t, map[string]string{
		"from":    "user@example.com",
		"to":      "a@example.com,b@example.com",
		"cc":      "c@example.com",
		"subject": "发送测试",
		"content": "<p>hello</p>",
	}, map[string][]byte{"a.txt": []byte("attachment")})

	rec := f.do(t, "POST", "/api/mail/send", mp, token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 || body["msg"] != "发送成功" {
		t.Fatalf("发送失败: %s", rec.Body.String())
	}

	if f.sender.SendCnt != 1 {
		t.Errorf("SMTP 调用次数错误: %d", f.sender.SendCnt)
	}
	last, _ := f.sender.Last()
	if len(last.To) != 2 || len(last.Cc) != 1 {
		t.Errorf("收件人解析错误: %+v", last)
	}

	// 收件人为空
	rec = f.do(t, "POST", "/api/mail/send", newMultipart(t, map[string]string{
		"from": "user@example.com", "subject": "x", "content": "y",
	}, nil), token)
	if body := decode(t, rec); body["code"].(float64) != 400 || body["msg"] != "收件人不能为空" {
		t.Errorf("收件人为空应返回 400: %s", rec.Body.String())
	}
}

func TestSaveDraftAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	mp := newMultipart(t, map[string]string{
		"from":    "user@example.com",
		"to":      "a@example.com",
		"subject": "草稿",
		"content": "<p>draft</p>",
	}, nil)

	rec := f.do(t, "POST", "/api/mail/save-draft", mp, token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 || body["msg"] != "草稿保存成功" {
		t.Fatalf("保存草稿失败: %s", rec.Body.String())
	}
	if f.server.FolderLen("Drafts") != 1 {
		t.Errorf("草稿数错误: %d", f.server.FolderLen("Drafts"))
	}
}

func TestDownloadAttachmentAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Attachments: []testutil.Attachment{{FileName: "下载.txt", ContentType: "text/plain", Content: []byte("download-body")}},
	})
	uid := f.server.AddMessage("INBOX", raw)

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	partID := env.Attachments[0].PartID

	rec := f.do(t, "POST", "/api/mail/download", map[string]any{
		"folder": "INBOX", "uid": uid, "part_id": partID,
	}, token)
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type 错误: %q", ct)
	}
	if !bytes.Contains([]byte(rec.Header().Get("Content-Disposition")), []byte("filename")) {
		t.Errorf("缺少文件名头: %q", rec.Header().Get("Content-Disposition"))
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("download-body")) {
		t.Errorf("附件内容错误: %q", rec.Body.String())
	}

	// 下载失败仍返回可读 blob
	rec = f.do(t, "POST", "/api/mail/download", map[string]any{
		"folder": "INBOX", "uid": uid, "part_id": "999",
	}, token)
	if !bytes.Contains(rec.Body.Bytes(), []byte("下载失败")) {
		t.Errorf("失败提示缺失: %q", rec.Body.String())
	}
}

func TestViewFileSafety(t *testing.T) {
	f := newAPI(t)

	// 正常文件
	filePath := filepath.Join(f.cfg.StaticDir, "images", "a.png")
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("image-content"), 0644); err != nil {
		t.Fatal(err)
	}

	rec := f.do(t, "GET", "/api/viewfile?url=static/images/a.png", nil, "")
	if rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte("image-content")) {
		t.Errorf("正常文件应可访问: %d %q", rec.Code, rec.Body.String())
	}

	// 路径穿越
	secret := filepath.Join(f.cfg.StaticDir, "..", "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	rec = f.do(t, "GET", "/api/viewfile?url=../secret.txt", nil, "")
	body := decode(t, rec)
	if body["code"].(float64) != 400 || body["msg"] != "非法文件路径" {
		t.Errorf("路径穿越应被拒绝: %s", rec.Body.String())
	}

	// 参数缺失
	rec = f.do(t, "GET", "/api/viewfile", nil, "")
	if body := decode(t, rec); body["code"].(float64) != 500 {
		t.Errorf("缺少参数应返回 500: %s", rec.Body.String())
	}
}

func TestUserAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	rec := f.do(t, "POST", "/api/user/list", map[string]any{}, token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("用户列表失败: %s", rec.Body.String())
	}
	data := body["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Errorf("用户数错误: %v", data["total"])
	}

	rec = f.do(t, "POST", "/api/user/contact/save", map[string]any{
		"prefix": "user_contact", "email": "a@example.com", "name": "A",
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Errorf("保存联系人失败: %s", rec.Body.String())
	}

	rec = f.do(t, "POST", "/api/user/contact/list", map[string]any{"prefix": "user_contact"}, token)
	body = decode(t, rec)
	data = body["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Errorf("联系人数错误: %v", data["total"])
	}

	// 修改密码：旧密码错误
	rec = f.do(t, "POST", "/api/user/chgpwd", map[string]string{"old_password": "bad", "new_password": "new"}, token)
	if body := decode(t, rec); body["code"].(float64) != 500 {
		t.Errorf("旧密码错误应失败: %s", rec.Body.String())
	}
}
