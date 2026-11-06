package controller_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"email-server/internal/testutil"
	"email-server/utils"
)

func TestMailOpsAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Subject: "操作接口测试",
		HTML:    `<p>ops</p>`,
		Attachments: []testutil.Attachment{
			{FileName: "ops.txt", ContentType: "text/plain", Content: []byte("ops")},
		},
	})
	uid := int64(f.server.AddMessage("INBOX", raw))

	// 未读数初始为 1。
	rec := f.do(t, "POST", "/api/mail/unread-total", nil, token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("未读数接口失败: %s", rec.Body.String())
	}
	if got := body["data"].(map[string]any)["INBOX"].(float64); got != 1 {
		t.Errorf("INBOX 未读数错误: %v", got)
	}

	// 标记星标后星标列表可见，且未读星标为 1。
	rec = f.do(t, "POST", "/api/mail/status", map[string]any{
		"folder": "INBOX", "uids": []int64{uid}, "status": "Flagged", "type": 1,
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("标记星标失败: %s", rec.Body.String())
	}
	rec = f.do(t, "POST", "/api/mail/star-list", map[string]any{}, token)
	body = decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("星标列表失败: %s", rec.Body.String())
	}
	data := body["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Errorf("星标总数错误: %v", data["total"])
	}
	if list := data["list"].([]any); len(list) != 1 || list[0].(map[string]any)["subject"] != "操作接口测试" {
		t.Errorf("星标列表内容错误: %v", data["list"])
	}

	// 标记已读后未读数归零，星标未读也归零。
	rec = f.do(t, "POST", "/api/mail/status", map[string]any{
		"folder": "INBOX", "uids": []int64{uid}, "status": "Seen", "type": 1,
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("标记已读失败: %s", rec.Body.String())
	}
	rec = f.do(t, "POST", "/api/mail/unread-total", nil, token)
	body = decode(t, rec)
	total := body["data"].(map[string]any)
	if total["INBOX"].(float64) != 0 || total["Star"].(float64) != 0 {
		t.Errorf("未读数未归零: %v", total)
	}

	// 同文件夹移动应被拒绝。
	rec = f.do(t, "POST", "/api/mail/move", map[string]any{
		"from_folder": "INBOX", "to_folder": "INBOX", "uids": []int64{uid},
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 400 {
		t.Errorf("同文件夹移动应返回 400: %s", rec.Body.String())
	}

	// 移动到 Junk 后原文件夹为空。
	rec = f.do(t, "POST", "/api/mail/move", map[string]any{
		"from_folder": "INBOX", "to_folder": "Junk", "uids": []int64{uid},
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("移动邮件失败: %s", rec.Body.String())
	}
	if f.server.FolderLen("INBOX") != 0 || f.server.FolderLen("Junk") != 1 {
		t.Errorf("移动后数量错误: INBOX=%d Junk=%d", f.server.FolderLen("INBOX"), f.server.FolderLen("Junk"))
	}

	// 永久删除。
	rec = f.do(t, "POST", "/api/mail/delete", map[string]any{
		"folder": "Junk", "uids": []int64{uid},
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("删除邮件失败: %s", rec.Body.String())
	}
	if f.server.FolderLen("Junk") != 0 {
		t.Errorf("删除后 Junk 仍有邮件: %d", f.server.FolderLen("Junk"))
	}

	// 非法状态标志。
	rec = f.do(t, "POST", "/api/mail/status", map[string]any{
		"folder": "INBOX", "uids": []int64{uid}, "status": "Unknown", "type": 1,
	}, token)
	if body := decode(t, rec); body["code"].(float64) == 200 {
		t.Errorf("非法状态标志不应成功: %s", rec.Body.String())
	}
}

func TestUserAndContactAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	rec := f.do(t, "POST", "/api/user/create", map[string]any{
		"email": "qa@example.com", "password": "qa-pass",
		"person_first_name": "Q", "person_last_name": "A", "is_admin": 0,
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("创建用户失败: %s", rec.Body.String())
	}
	if _, err := f.dir.Get("qa@example.com"); err != nil {
		t.Errorf("创建用户后目录中不存在: %v", err)
	}

	rec = f.do(t, "POST", "/api/user/update", map[string]any{
		"email": "qa@example.com", "person_first_name": "Qa", "person_last_name": "Test", "is_admin": 1,
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("更新用户失败: %s", rec.Body.String())
	}
	acc, err := f.dir.Get("qa@example.com")
	if err != nil || acc.FirstName != "Qa" || acc.LastName != "Test" || acc.AdminLevel != 1 {
		t.Errorf("用户更新结果错误: %+v %v", acc, err)
	}

	rec = f.do(t, "POST", "/api/user/list", map[string]any{}, token)
	body := decode(t, rec)
	list := body["data"].(map[string]any)["list"].([]any)
	if len(list) != 2 {
		t.Errorf("用户列表数量错误: %d", len(list))
	}

	rec = f.do(t, "POST", "/api/user/delete", map[string]any{"email": "qa@example.com"}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("删除用户失败: %s", rec.Body.String())
	}
	if _, err := f.dir.Get("qa@example.com"); err == nil {
		t.Error("删除用户后目录中仍存在")
	}

	// 联系人新增、删除、清空。
	for _, c := range []map[string]any{
		{"prefix": "user_contact", "email": "a@example.com", "name": "A"},
		{"prefix": "user_contact", "email": "b@example.com", "name": "B"},
	} {
		rec = f.do(t, "POST", "/api/user/contact/save", c, token)
		if body := decode(t, rec); body["code"].(float64) != 200 {
			t.Fatalf("保存联系人失败: %s", rec.Body.String())
		}
	}
	rec = f.do(t, "POST", "/api/user/contact/list", map[string]any{"prefix": "user_contact"}, token)
	body = decode(t, rec)
	if body["data"].(map[string]any)["total"].(float64) != 2 {
		t.Errorf("联系人数错误: %s", rec.Body.String())
	}

	rec = f.do(t, "POST", "/api/user/contact/delete", map[string]any{
		"prefix": "user_contact", "email": "a@example.com",
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("删除联系人失败: %s", rec.Body.String())
	}
	rec = f.do(t, "POST", "/api/user/contact/list", map[string]any{"prefix": "user_contact"}, token)
	body = decode(t, rec)
	if body["data"].(map[string]any)["total"].(float64) != 1 {
		t.Errorf("删除后联系人数错误: %s", rec.Body.String())
	}

	rec = f.do(t, "POST", "/api/user/contact/clear", map[string]any{"prefix": "user_contact"}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("清空联系人失败: %s", rec.Body.String())
	}
	rec = f.do(t, "POST", "/api/user/contact/list", map[string]any{"prefix": "user_contact"}, token)
	body = decode(t, rec)
	if body["data"].(map[string]any)["total"].(float64) != 0 {
		t.Errorf("清空后联系人数错误: %s", rec.Body.String())
	}

	// 头像上传（1x1 PNG）。
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R',
		0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0x1f, 0x15, 0xc4, 0x89}
	buf, writer := testutil.MultipartWriter()
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(png); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	rec = f.do(t, "POST", "/api/user/uploadavatar", &multipartBody{buf: buf, contentType: writer.FormDataContentType()}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("上传头像失败: %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(f.cfg.StaticDir, "avatars", "user@example.com.webp")); err != nil {
		t.Errorf("头像文件未写入: %v", err)
	}
}

func TestUnScheduleAPI(t *testing.T) {
	f := newAPI(t)
	token := f.login(t)

	schedule := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	rec := f.do(t, "POST", "/api/mail/save-draft", newMultipart(t, map[string]string{
		"from":            "user@example.com",
		"to":              "a@example.com",
		"subject":         "定时草稿",
		"content":         "<p>schedule</p>",
		"x-schedule-send": schedule,
	}, nil), token)
	body := decode(t, rec)
	if body["code"].(float64) != 200 {
		t.Fatalf("保存定时草稿失败: %s", rec.Body.String())
	}
	if f.server.FolderLen("Drafts") != 1 {
		t.Fatalf("草稿数量错误: %d", f.server.FolderLen("Drafts"))
	}
	uid := int64(f.server.Messages("Drafts")[0].UID)
	if header := utils.ExtractHeader(f.server.Messages("Drafts")[0].Raw, "X-Schedule-Send"); header == "" {
		t.Fatal("定时草稿没有 X-Schedule-Send 头")
	}

	rec = f.do(t, "POST", "/api/mail/un-schedule", map[string]any{
		"folder": "Drafts", "uid": uid, "status": "Draft", "type": 2,
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Fatalf("取消定时失败: %s", rec.Body.String())
	}
	if f.server.FolderLen("Drafts") != 1 {
		t.Fatalf("取消定时后草稿数量错误: %d", f.server.FolderLen("Drafts"))
	}
	newRaw := f.server.Messages("Drafts")[0].Raw
	if header := utils.ExtractHeader(newRaw, "X-Schedule-Send"); header != "" {
		t.Errorf("取消定时后仍存在 X-Schedule-Send: %q", header)
	}

	// 清理草稿。
	rec = f.do(t, "POST", "/api/mail/delete", map[string]any{
		"folder": "Drafts", "uids": []int64{int64(f.server.Messages("Drafts")[0].UID)},
	}, token)
	if body := decode(t, rec); body["code"].(float64) != 200 {
		t.Errorf("清理草稿失败: %s", rec.Body.String())
	}
}
