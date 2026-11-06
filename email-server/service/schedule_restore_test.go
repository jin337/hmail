package service

import (
	"testing"
	"time"

	"email-server/internal/testutil"
)

// TestEnsureScheduledRestored 服务重启后草稿箱里的定时邮件应被重新注册并投递。
func TestEnsureScheduledRestored(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		From:     "user@example.com",
		To:       "阿三 <a@example.com>",
		Subject:  "restored schedule",
		Text:     "body",
		Schedule: time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05"),
	})
	uid := server.AddMessage("Drafts", raw, `\Draft`, `\Seen`)
	if uid == 0 {
		t.Fatal("草稿写入失败")
	}

	svc.EnsureScheduledRestored("user@example.com", "pwd")

	if sender.SendCnt != 1 {
		t.Fatalf("恢复后的定时任务应发送 1 次，实际 %d", sender.SendCnt)
	}
	last, ok := sender.Last()
	if !ok {
		t.Fatal("SMTP 未收到邮件")
	}
	if len(last.To) != 1 || last.To[0] != "a@example.com" {
		t.Errorf("收件人应解析为纯净地址，实际 %v", last.To)
	}
	if server.FolderLen("Drafts") != 0 || server.FolderLen("Sent") != 1 {
		t.Errorf("恢复后归档错误: Drafts=%d Sent=%d", server.FolderLen("Drafts"), server.FolderLen("Sent"))
	}
	if svc.ScheduledCount() != 0 {
		t.Errorf("任务完成后应释放注册记录，实际 %d", svc.ScheduledCount())
	}
}

// TestEnsureScheduledRestoredOncePerUser 同一用户多次请求只扫描一次，不会重复投递。
func TestEnsureScheduledRestoredOncePerUser(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	var pending []func()
	svc.SetAsyncRunner(func(fn func()) { pending = append(pending, fn) })
	svc.SetSleeper(func(time.Duration) {})

	raw := testutil.BuildMessage(testutil.MessageOptions{
		To:       "a@example.com",
		Subject:  "dedupe",
		Schedule: time.Now().Add(time.Hour).Format("2006-01-02 15:04:05"),
	})
	server.AddMessage("Drafts", raw, `\Draft`)

	svc.EnsureScheduledRestored("user@example.com", "pwd")
	svc.EnsureScheduledRestored("user@example.com", "pwd")

	if len(pending) != 1 {
		t.Fatalf("同一用户只应扫描一次，实际排队 %d 个任务", len(pending))
	}

	pending[0]() // 执行扫描，内部会注册到点任务
	if svc.ScheduledCount() != 1 {
		t.Fatalf("扫描后应注册 1 个任务，实际 %d", svc.ScheduledCount())
	}
	if len(pending) != 2 {
		t.Fatalf("应排队 1 个到点任务，实际 %d", len(pending))
	}

	pending[1]() // 执行到点任务
	if sender.SendCnt != 1 {
		t.Errorf("应发送 1 次，实际 %d", sender.SendCnt)
	}
	if svc.ScheduledCount() != 0 {
		t.Errorf("任务执行后应释放注册记录，实际 %d", svc.ScheduledCount())
	}
}

// TestRestoreSkipsDraftWithoutSchedule 普通草稿不应被当成定时任务。
func TestRestoreSkipsDraftWithoutSchedule(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	server.AddMessage("Drafts", testutil.BuildMessage(testutil.MessageOptions{
		To:      "a@example.com",
		Subject: "plain draft",
	}), `\Draft`)

	svc.EnsureScheduledRestored("user@example.com", "pwd")

	if sender.SendCnt != 0 {
		t.Errorf("普通草稿不应被发送，实际 %d 次", sender.SendCnt)
	}
	if server.FolderLen("Drafts") != 1 {
		t.Errorf("普通草稿应保留，实际 %d", server.FolderLen("Drafts"))
	}
}
