package service

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"email-server/internal/testutil"

	"github.com/jhillyerd/enmime"
)

func TestUpdateMailFlag(t *testing.T) {
	svc, server, _, _ := newTestService(t)
	uid := server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{}))

	if err := svc.UpdateMailFlag("user@example.com", "pwd", "INBOX", []int64{int64(uid)}, 1, "Flagged"); err != nil {
		t.Fatalf("添加标记失败: %v", err)
	}
	msgs := server.Messages("INBOX")
	if len(msgs) != 1 || !strings.Contains(strings.Join(msgs[0].Flags, ","), `\Flagged`) {
		t.Errorf("标记未添加: %v", msgs[0].Flags)
	}

	if err := svc.UpdateMailFlag("user@example.com", "pwd", "INBOX", []int64{int64(uid)}, 2, "Flagged"); err != nil {
		t.Fatalf("移除标记失败: %v", err)
	}
	if strings.Contains(strings.Join(server.Messages("INBOX")[0].Flags, ","), "Flagged") {
		t.Error("标记未被移除")
	}

	if err := svc.UpdateMailFlag("user@example.com", "pwd", "INBOX", []int64{int64(uid)}, 1, "Unknown"); err == nil {
		t.Error("非法状态应报错")
	}
	if err := svc.UpdateMailFlag("user@example.com", "pwd", "INBOX", []int64{int64(uid)}, 9, "Seen"); err == nil {
		t.Error("非法操作类型应报错")
	}
}

func TestMoveAndDeleteMail(t *testing.T) {
	svc, server, _, _ := newTestService(t)
	uid := server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{}))

	if err := svc.MoveMail("user@example.com", "pwd", "INBOX", "INBOX", []int64{int64(uid)}); err == nil {
		t.Error("相同文件夹应报错")
	}
	if err := svc.MoveMail("user@example.com", "pwd", "INBOX", "Star", []int64{int64(uid)}); err == nil {
		t.Error("虚拟文件夹 Star 不能作为移动目标")
	}

	if err := svc.MoveMail("user@example.com", "pwd", "INBOX", "Deleted", []int64{int64(uid)}); err != nil {
		t.Fatalf("移动失败: %v", err)
	}
	if server.FolderLen("INBOX") != 0 || server.FolderLen("Deleted") != 1 {
		t.Errorf("移动结果错误: INBOX=%d Deleted=%d", server.FolderLen("INBOX"), server.FolderLen("Deleted"))
	}

	// 删除时清理内联图片缓存
	dir := filepath.Join(svc.Config().StaticDir, "images", "user@example.com", "1")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "image_2.png"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteMail("user@example.com", "pwd", "Deleted", []int64{int64(uid)}); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if server.FolderLen("Deleted") != 0 {
		t.Error("邮件未被删除")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("内联图片缓存未被清理")
	}
}

func TestDownloadAttachment(t *testing.T) {
	svc, server, _, _ := newTestService(t)
	raw := testutil.BuildMessage(testutil.MessageOptions{
		Attachments: []testutil.Attachment{{FileName: "a.txt", ContentType: "text/plain", Content: []byte("content-a")}},
	})
	uid := server.AddMessage("INBOX", raw)

	env, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	partID := env.Attachments[0].PartID

	name, data, err := svc.DownloadAttachment("user@example.com", "pwd", "INBOX", int64(uid), partID)
	if err != nil {
		t.Fatalf("下载附件失败: %v", err)
	}
	if name != "a.txt" || string(data) != "content-a" {
		t.Errorf("附件内容错误: %q %q", name, data)
	}

	if _, _, err := svc.DownloadAttachment("user@example.com", "pwd", "INBOX", int64(uid), "999"); err == nil {
		t.Error("不存在的 PartID 应报错")
	}
}

func TestSaveAndUpdateDraft(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	raw := []byte("From: a@b.com\r\nSubject: draft\r\n\r\nbody")
	if err := svc.SaveMailToFolder("user@example.com", "pwd", "Drafts", raw); err != nil {
		t.Fatal(err)
	}
	if server.FolderLen("Drafts") != 1 {
		t.Fatal("草稿未保存")
	}
	uid := server.Messages("Drafts")[0].UID

	newRaw := []byte("From: a@b.com\r\nSubject: draft2\r\n\r\nbody2")
	if err := svc.UpdateDraft("user@example.com", "pwd", "Drafts", newRaw, int64(uid)); err != nil {
		t.Fatal(err)
	}
	if server.FolderLen("Drafts") != 1 {
		t.Errorf("替换草稿后应只剩 1 封，实际 %d", server.FolderLen("Drafts"))
	}
	if !bytes.Contains(server.Messages("Drafts")[0].Raw, []byte("draft2")) {
		t.Error("草稿内容未更新")
	}
}

func TestUnScheduleEmail(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	raw := []byte("From: a@b.com\r\nTo: c@d.com\r\nSubject: scheduled\r\nX-Schedule-Send: Wed, 01 Jan 2030 00:00:00 +0000\r\nX-Folded: a\r\n b\r\n\r\nbody")
	uid := server.AddMessage("Drafts", raw, `\Draft`, `\Seen`)

	if err := svc.UnScheduleEmail("user@example.com", "pwd", "Drafts", int64(uid), 2, "Draft"); err != nil {
		t.Fatalf("取消失败: %v", err)
	}

	if server.FolderLen("Drafts") != 1 {
		t.Fatalf("取消定时后草稿数错误: %d", server.FolderLen("Drafts"))
	}
	newRaw := server.Messages("Drafts")[0].Raw
	if bytes.Contains(newRaw, []byte("X-Schedule-Send")) {
		t.Error("X-Schedule-Send 未被移除")
	}
	if !bytes.Contains(newRaw, []byte("X-Folded: a\r\n b")) {
		t.Error("其它折叠头被误删")
	}
	if !bytes.Contains(newRaw, []byte("body")) {
		t.Error("正文被破坏")
	}
}

func TestRemoveHeader(t *testing.T) {
	raw := []byte("A: 1\r\nX-Test: v\r\n continuation\r\nB: 2\r\n\r\nbody X-Test: keep")
	out := RemoveHeader(raw, "x-test")
	if bytes.Contains(out, []byte("X-Test: v")) || bytes.Contains(out, []byte(" continuation")) {
		t.Errorf("头部未被完整移除: %q", out)
	}
	if !bytes.Contains(out, []byte("A: 1\r\nB: 2\r\n\r\nbody X-Test: keep")) {
		t.Errorf("其余内容被破坏: %q", out)
	}

	if out := RemoveHeader([]byte("no header separator"), "X"); string(out) != "no header separator" {
		t.Error("无分隔符时应原样返回")
	}
}
