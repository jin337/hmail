package controller_test

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"email-server/config"
	"email-server/contacts"
	"email-server/controller"
	"email-server/directory"
	"email-server/mailbox"
	"email-server/router"
	"email-server/service"

	"github.com/gin-gonic/gin"
)

const (
	realAccount  = "admin@jin337.top"
	realPassword = "a123456"
)

type realClient struct {
	t      *testing.T
	base   string
	client *http.Client
	token  string
}

func realJSON(t *testing.T, c *realClient, path, token string, payload any) (int, map[string]any, []byte) {
	t.Helper()
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest("POST", c.base+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("请求 %s 失败: %v", path, err)
	}
	return resp.StatusCode, decodeReal(t, resp, path), nil
}

func decodeReal(t *testing.T, resp *http.Response, path string) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读取 %s 响应失败: %v", path, err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("%s 响应不是 JSON: %v (%s)", path, err, string(raw))
	}
	return body
}

func realCode(body map[string]any) int {
	if v, ok := body["code"].(float64); ok {
		return int(v)
	}
	return 0
}

func realItems(data map[string]any) []any {
	if v, ok := data["list"].([]any); ok {
		return v
	}
	return nil
}

func realMultipart(t *testing.T, fields map[string]string, files map[string][]byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
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
	return buf.Bytes(), writer.FormDataContentType()
}

func (c *realClient) postMultipart(t *testing.T, path string, raw []byte, contentType string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest("POST", c.base+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("请求 %s 失败: %v", path, err)
	}
	return resp.StatusCode, decodeReal(t, resp, path)
}

// downloadRaw 直接读取附件下载的二进制响应，不按 JSON 解析。
func (c *realClient) downloadRaw(t *testing.T, folder string, uid int64, partID string) (int, []byte, http.Header) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"folder": folder, "uid": uid, "part_id": partID})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", c.base+"/api/mail/download", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("下载附件请求失败: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读取下载响应失败: %v", err)
	}
	return resp.StatusCode, data, resp.Header
}

func (c *realClient) list(t *testing.T, folder string, page, size int64) map[string]any {
	t.Helper()
	_, body, _ := realJSON(t, c, "/api/mail/list", c.token, map[string]any{
		"folder": folder, "page": page, "size": size,
	})
	if realCode(body) != 200 {
		t.Fatalf("列表 %s 失败: %v", folder, body)
	}
	return body["data"].(map[string]any)
}

func (c *realClient) findMessage(t *testing.T, folder, subject string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		data := c.list(t, folder, 1, 50)
		list := realItems(data)
		for _, it := range list {
			item := it.(map[string]any)
			if item["subject"] == subject {
				return item
			}
		}
		time.Sleep(700 * time.Millisecond)
	}
	return nil
}

func (c *realClient) cleanupMatching(t *testing.T, match func(string) bool) {
	t.Helper()
	for _, folder := range []string{"INBOX", "Drafts", "Junk", "Sent", "Deleted"} {
		func() {
			defer func() { _ = recover() }()
			data := c.list(t, folder, 1, 50)
			list := realItems(data)
			var uids []int64
			for _, it := range list {
				item, _ := it.(map[string]any)
				subject, _ := item["subject"].(string)
				if match(subject) {
					if uid, ok := item["uid"].(float64); ok {
						uids = append(uids, int64(uid))
					}
				}
			}
			if len(uids) == 0 {
				return
			}
			_, body, _ := realJSON(t, c, "/api/mail/delete", c.token, map[string]any{"folder": folder, "uids": uids})
			if realCode(body) != 200 {
				t.Logf("清理 %s 失败: %v", folder, body)
			}
		}()
	}
}

func (c *realClient) cleanupSubjects(t *testing.T, subjects []string) {
	t.Helper()
	c.cleanupMatching(t, func(subject string) bool {
		for _, s := range subjects {
			if subject == s {
				return true
			}
		}
		return false
	})
}

func (c *realClient) cleanupPrefix(t *testing.T, prefix string) {
	t.Helper()
	c.cleanupMatching(t, func(subject string) bool {
		return strings.HasPrefix(subject, prefix)
	})
}

func TestRealFullHTTPFlow(t *testing.T) {
	if os.Getenv("MAIL_REAL_TEST") != "1" {
		t.Skip("设置 MAIL_REAL_TEST=1 运行真实邮箱联调")
	}

	_, file, _, _ := runtime.Caller(0)
	cfgPath := filepath.Join(filepath.Dir(file), "..", "config", "config.env")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("加载真实配置失败: %v", err)
	}
	cfg.Mode = "debug"
	cfg.APIPrefix = "/api"
	cfg.StaticDir = t.TempDir()

	storePath := filepath.Join(t.TempDir(), "contacts", "contact_db")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := contacts.Open(storePath)
	if err != nil {
		t.Fatalf("打开联系人库失败: %v", err)
	}
	defer store.Close()

	svc := service.New(service.Deps{
		Config:    cfg,
		IMAP:      mailbox.NewIMAPFactory(cfg.IMAP.Host, cfg.IMAP.Port),
		SMTP:      mailbox.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port),
		Directory: directory.New(cfg.AdminPassword),
		Contacts:  store,
	})
	engine := gin.New()
	router.Setup(engine, controller.New(svc, cfg), cfg)
	ts := httptest.NewServer(engine)
	defer ts.Close()

	c := &realClient{t: t, base: ts.URL, client: &http.Client{Timeout: 40 * time.Second}}

	// 登录成功/失败与鉴权。
	_, body, _ := realJSON(t, c, "/api/login", "", map[string]string{
		"email": realAccount, "password": realPassword,
	})
	if realCode(body) != 200 {
		t.Fatalf("真实账号登录失败: %v", body)
	}
	c.token = body["data"].(map[string]any)["token"].(string)
	// 清理此前失败运行留下的 FULLTEST 邮件。
	c.cleanupPrefix(t, "FULLTEST-")

	_, bad, _ := realJSON(t, c, "/api/login", "", map[string]string{
		"email": realAccount, "password": "wrong-password",
	})
	if realCode(bad) == 200 {
		t.Errorf("错误密码登录不应成功: %v", bad)
	}

	_, noAuth, _ := realJSON(t, c, "/api/mail/list", "", map[string]any{"folder": "INBOX", "page": 1, "size": 5})
	if realCode(noAuth) != 401 {
		t.Errorf("无令牌访问应返回 401: %v", noAuth)
	}

	// 账号目录（只读）。
	_, users, _ := realJSON(t, c, "/api/user/list", c.token, map[string]any{})
	if realCode(users) != 200 {
		t.Fatalf("真实用户列表失败: %v", users)
	}
	userData := users["data"].(map[string]any)
	foundAdmin := false
	for _, u := range realItems(userData) {
		if u.(map[string]any)["email"] == realAccount {
			foundAdmin = true
		}
	}
	if !foundAdmin {
		t.Errorf("用户列表中缺少登录账号: %v", userData)
	}

	// 各文件夹列表、星标列表、未读数。
	for _, folder := range []string{"INBOX", "Sent", "Drafts", "Deleted"} {
		data := c.list(t, folder, 1, 10)
		if _, ok := data["total"].(float64); !ok {
			t.Errorf("%s 列表缺少 total: %v", folder, data)
		}
	}
	_, starBody, _ := realJSON(t, c, "/api/mail/star-list", c.token, map[string]any{"page": 1, "size": 10})
	if realCode(starBody) != 200 {
		t.Errorf("星标列表失败: %v", starBody)
	}
	_, unreadBody, _ := realJSON(t, c, "/api/mail/unread-total", c.token, map[string]any{})
	if realCode(unreadBody) != 200 {
		t.Errorf("未读数失败: %v", unreadBody)
	}

	// 现有邮件详情、附件下载、viewfile。
	inbox := c.list(t, "INBOX", 1, 20)
	var target map[string]any
	inboxItems := realItems(inbox)
	for _, it := range inboxItems {
		item := it.(map[string]any)
		if flag, _ := item["has_attach"].(bool); flag {
			target = item
			break
		}
	}
	if target == nil && len(inboxItems) > 0 {
		target = inboxItems[0].(map[string]any)
	}
	if target != nil {
		uid := int64(target["uid"].(float64))
		_, detail, _ := realJSON(t, c, "/api/mail/detail", c.token, map[string]any{"folder": "INBOX", "uid": uid})
		if realCode(detail) != 200 {
			t.Errorf("邮件详情失败 uid=%d: %v", uid, detail)
		} else {
			dd := detail["data"].(map[string]any)
			content := dd["content"].(string)
			if content == "" {
				t.Errorf("详情正文为空 uid=%d", uid)
			}
			atts, _ := dd["attachments"].([]any)
			if len(atts) > 0 {
				att := atts[0].(map[string]any)
				partID := att["part_id"].(string)
				status, data, headers := c.downloadRaw(t, "INBOX", uid, partID)
				if status != 200 || len(data) == 0 {
					t.Errorf("附件下载失败 part=%s: status=%d len=%d", partID, status, len(data))
				}
				if cd := headers.Get("Content-Disposition"); !strings.Contains(cd, "filename") {
					t.Errorf("附件下载缺少文件名头: %q", cd)
				}
			}
			re := regexp.MustCompile(`data-href="([^"]+)"`)
			if m := re.FindStringSubmatch(content); len(m) == 2 {
				targetURL := html.UnescapeString(m[1])
				if !strings.HasPrefix(targetURL, "http") {
					targetURL = ts.URL + targetURL
				}
				resp, err := c.client.Get(targetURL)
				if err != nil {
					t.Errorf("viewfile 请求失败: %v", err)
				} else {
					data, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if resp.StatusCode != 200 || len(data) == 0 {
						t.Errorf("viewfile 返回异常: status=%d len=%d url=%s", resp.StatusCode, len(data), targetURL)
					}
				}
			}
		}
	}

	// 发送一封到本账号的测试邮件。
	stamp := time.Now().Format("20060102-150405")
	subject := "FULLTEST-" + stamp
	defer c.cleanupSubjects(t, []string{subject, subject + "-SCHED"})
	raw, ct := realMultipart(t, map[string]string{
		"from":    realAccount,
		"to":      realAccount,
		"subject": subject,
		"content": "<p>full-http-flow " + stamp + "</p>",
	}, map[string][]byte{"fulltest.txt": []byte("full test attachment")})
	_, sendBody := c.postMultipart(t, "/api/mail/send", raw, ct)
	if realCode(sendBody) != 200 {
		t.Fatalf("真实发送失败: %v", sendBody)
	}

	msg := c.findMessage(t, "INBOX", subject)
	if msg == nil {
		t.Fatalf("发送后未在 INBOX 找到测试邮件 %s", subject)
	}
	uid := int64(msg["uid"].(float64))
	_, testDetail, _ := realJSON(t, c, "/api/mail/detail", c.token, map[string]any{"folder": "INBOX", "uid": uid})
	if realCode(testDetail) != 200 {
		t.Fatalf("测试邮件详情失败: %v", testDetail)
	}
	td := testDetail["data"].(map[string]any)
	if !strings.Contains(td["content"].(string), stamp) {
		t.Errorf("测试邮件正文未包含标记: %s", td["content"])
	}
	if atts, _ := td["attachments"].([]any); len(atts) == 0 {
		t.Errorf("测试邮件未解析出附件: %v", td)
	}

	// 标记、星标、已读。
	_, mark, _ := realJSON(t, c, "/api/mail/status", c.token, map[string]any{
		"folder": "INBOX", "uids": []int64{uid}, "status": "Flagged", "type": 1,
	})
	if realCode(mark) != 200 {
		t.Errorf("标记星标失败: %v", mark)
	}
	_, starAfter, _ := realJSON(t, c, "/api/mail/star-list", c.token, map[string]any{"page": 1, "size": 20})
	starList := realItems(starAfter["data"].(map[string]any))
	starFound := false
	for _, it := range starList {
		if it.(map[string]any)["subject"] == subject {
			starFound = true
		}
	}
	if !starFound {
		t.Errorf("星标列表中未找到测试邮件: %v", starAfter)
	}
	for _, fn := range []func() (int, map[string]any, []byte){
		func() (int, map[string]any, []byte) {
			return realJSON(t, c, "/api/mail/status", c.token, map[string]any{
				"folder": "INBOX", "uids": []int64{uid}, "status": "Seen", "type": 1,
			})
		},
		func() (int, map[string]any, []byte) {
			return realJSON(t, c, "/api/mail/status", c.token, map[string]any{
				"folder": "INBOX", "uids": []int64{uid}, "status": "Flagged", "type": 2,
			})
		},
	} {
		_, flagBody, _ := fn()
		if realCode(flagBody) != 200 {
			t.Errorf("更新标记失败: %v", flagBody)
		}
	}

	// 同文件夹移动应报错，随后移动到 Junk 并永久删除。
	_, sameMove, _ := realJSON(t, c, "/api/mail/move", c.token, map[string]any{
		"from_folder": "INBOX", "to_folder": "INBOX", "uids": []int64{uid},
	})
	if realCode(sameMove) != 400 {
		t.Errorf("同文件夹移动应 400: %v", sameMove)
	}
	_, moveBody, _ := realJSON(t, c, "/api/mail/move", c.token, map[string]any{
		"from_folder": "INBOX", "to_folder": "Junk", "uids": []int64{uid},
	})
	if realCode(moveBody) != 200 {
		t.Errorf("移动测试邮件到 Junk 失败: %v", moveBody)
	} else {
		moved := c.findMessage(t, "Junk", subject)
		if moved == nil {
			t.Errorf("移动后未在 Junk 找到测试邮件")
		} else {
			_, delBody, _ := realJSON(t, c, "/api/mail/delete", c.token, map[string]any{
				"folder": "Junk", "uids": []int64{int64(moved["uid"].(float64))},
			})
			if realCode(delBody) != 200 {
				t.Errorf("删除测试邮件失败: %v", delBody)
			}
		}
	}

	// 定时发送 + 取消定时。
	schedSubject := subject + "-SCHED"
	schedTime := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	sraw, sct := realMultipart(t, map[string]string{
		"from":            realAccount,
		"to":              realAccount,
		"subject":         schedSubject,
		"content":         "<p>scheduled " + stamp + "</p>",
		"x-schedule-send": schedTime,
	}, nil)
	_, schedSend := c.postMultipart(t, "/api/mail/send", sraw, sct)
	if realCode(schedSend) != 200 {
		t.Fatalf("设置定时发送失败: %v", schedSend)
	}
	if msgText, _ := schedSend["msg"].(string); !strings.Contains(msgText, "定时") {
		t.Errorf("定时发送提示异常: %v", schedSend)
	}
	schedMsg := c.findMessage(t, "Drafts", schedSubject)
	if schedMsg == nil {
		t.Fatalf("定时发送后未在 Drafts 找到 %s", schedSubject)
	}
	schedUID := int64(schedMsg["uid"].(float64))
	_, cancelBody, _ := realJSON(t, c, "/api/mail/un-schedule", c.token, map[string]any{
		"folder": "Drafts", "uid": schedUID, "status": "Draft", "type": 2,
	})
	if realCode(cancelBody) != 200 {
		t.Fatalf("取消定时失败: %v", cancelBody)
	}
	time.Sleep(1 * time.Second)
	cancelled := c.findMessage(t, "Drafts", schedSubject)
	if cancelled == nil {
		t.Fatalf("取消定时后草稿丢失")
	} else if schedule, _ := cancelled["schedule"].(string); strings.HasPrefix(schedule, "0001-") {
		// 已清除定时时间。
	} else {
		t.Errorf("取消定时后 schedule 未清空: %v", cancelled)
	}
	_, delDraft, _ := realJSON(t, c, "/api/mail/delete", c.token, map[string]any{
		"folder": "Drafts", "uids": []int64{int64(cancelled["uid"].(float64))},
	})
	if realCode(delDraft) != 200 {
		t.Errorf("删除定时草稿失败: %v", delDraft)
	}

	// 联系人读写。
	contactPrefix := "fulltest_contact"
	_, saveContact, _ := realJSON(t, c, "/api/user/contact/save", c.token, map[string]any{
		"prefix": contactPrefix, "email": "qa-contact@example.com", "name": "QA",
	})
	if realCode(saveContact) != 200 {
		t.Errorf("保存联系人失败: %v", saveContact)
	}
	_, listContact, _ := realJSON(t, c, "/api/user/contact/list", c.token, map[string]any{"prefix": contactPrefix})
	if realCode(listContact) != 200 {
		t.Errorf("联系人列表失败: %v", listContact)
	} else if total, _ := listContact["data"].(map[string]any)["total"].(float64); total != 1 {
		t.Errorf("联系人数错误: %v", listContact)
	}
	_, delContact, _ := realJSON(t, c, "/api/user/contact/delete", c.token, map[string]any{
		"prefix": contactPrefix, "email": "qa-contact@example.com",
	})
	if realCode(delContact) != 200 {
		t.Errorf("删除联系人失败: %v", delContact)
	}
	_, clearContact, _ := realJSON(t, c, "/api/user/contact/clear", c.token, map[string]any{"prefix": contactPrefix})
	if realCode(clearContact) != 200 {
		t.Errorf("清空联系人失败: %v", clearContact)
	}

	// 头像上传写入临时静态目录，不影响生产头像。
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R',
		0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}
	var avBuf bytes.Buffer
	avWriter := multipart.NewWriter(&avBuf)
	part, err := avWriter.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(png)
	_ = avWriter.Close()
	req, err := http.NewRequest("POST", ts.URL+"/api/user/uploadavatar", bytes.NewReader(avBuf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", avWriter.FormDataContentType())
	req.Header.Set("Authorization", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		t.Fatalf("上传头像请求失败: %v", err)
	}
	avatarBody := decodeReal(t, resp, "/api/user/uploadavatar")
	if realCode(avatarBody) != 200 {
		t.Errorf("真实头像上传失败: %v", avatarBody)
	}
	if _, err := os.Stat(filepath.Join(cfg.StaticDir, "avatars", realAccount+".webp")); err != nil {
		t.Errorf("头像文件未写入临时目录: %v", err)
	}

	// 错误分支：缺参、无效附件。
	_, badDetail, _ := realJSON(t, c, "/api/mail/detail", c.token, map[string]any{"folder": "INBOX"})
	if realCode(badDetail) == 200 {
		t.Errorf("缺少 uid 的详情请求不应成功: %v", badDetail)
	}
	badStatus, badData, _ := c.downloadRaw(t, "INBOX", uid, "999")
	if badStatus == 200 && !bytes.Contains(badData, []byte("下载失败")) {
		t.Errorf("无效 PartID 下载不应成功: status=%d body=%s", badStatus, string(badData))
	}

	// 最终确认测试邮件已清理。
	for _, folder := range []string{"INBOX", "Junk", "Drafts"} {
		data := c.list(t, folder, 1, 50)
		for _, it := range realItems(data) {
			item := it.(map[string]any)
			if s, _ := item["subject"].(string); s == subject || s == schedSubject {
				t.Errorf("测试邮件残留于 %s: %s", folder, s)
			}
		}
	}
}
