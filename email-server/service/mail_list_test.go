package service

import (
	"testing"
	"time"

	"email-server/config"
	"email-server/contacts"
	"email-server/internal/testutil"
)

// newTestService 构造一个完全基于内存依赖的 Service。
func newTestService(t *testing.T) (*Service, *testutil.Server, *testutil.Sender, *testutil.Directory) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.StaticDir = t.TempDir()
	cfg.ContactsPath = t.TempDir()

	server := testutil.NewServer("user@example.com", "pwd")
	sender := &testutil.Sender{}
	dir := testutil.NewDirectory()
	dir.Add("user@example.com", "张", "三", 0)

	svc := New(Deps{
		Config:    cfg,
		IMAP:      server,
		SMTP:      sender,
		Directory: dir,
		Contacts:  contacts.NewMemoryStore(),
	})
	svc.SetAsyncRunner(func(fn func()) { fn() })
	svc.SetSleeper(func(time.Duration) {})

	return svc, server, sender, dir
}

func seedInbox(t *testing.T, server *testutil.Server, count int) {
	t.Helper()
	base := time.Now().Add(-time.Duration(count) * time.Hour)
	for i := 0; i < count; i++ {
		raw := testutil.BuildMessage(testutil.MessageOptions{
			Subject: "邮件 " + string(rune('A'+i)),
			Date:    base.Add(time.Duration(i) * time.Hour),
			Text:    "正文内容",
		})
		server.AddMessage("INBOX", raw)
	}
}

func TestMailListPagination(t *testing.T) {
	svc, server, _, _ := newTestService(t)
	seedInbox(t, server, 12)

	list, total, err := svc.MailList("user@example.com", "pwd", "INBOX", 1, 5, "", nil)
	if err != nil {
		t.Fatalf("MailList 失败: %v", err)
	}
	if total != 12 {
		t.Errorf("total 应为 12，实际 %d", total)
	}
	if len(list) != 5 {
		t.Fatalf("第一页应有 5 封，实际 %d", len(list))
	}

	all := server.Messages("INBOX")
	// page=1 是最后一页（最新的 5 封），且按序列号升序返回
	if list[0].Uid != int64(all[7].UID) || list[4].Uid != int64(all[11].UID) {
		t.Errorf("分页 UID 范围错误: got %d..%d, want %d..%d",
			list[0].Uid, list[4].Uid, all[7].UID, all[11].UID)
	}

	list, total, err = svc.MailList("user@example.com", "pwd", "INBOX", 3, 5, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 12 || len(list) != 2 {
		t.Errorf("第三页应有 2 封，实际 %d（total=%d）", len(list), total)
	}

	// 超出范围返回空列表但保留总数
	list, total, err = svc.MailList("user@example.com", "pwd", "INBOX", 4, 5, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 || total != 12 {
		t.Errorf("越界分页应返回空列表，实际 %d 封", len(list))
	}
}

func TestMailListUnreadKeywordAndSort(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{Subject: "alpha report"}))
	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{Subject: "beta report"}), `\Seen`)
	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{Subject: "gamma"}))

	list, total, err := svc.MailList("user@example.com", "pwd", "INBOX", 1, 10, "", []string{"unread"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Errorf("未读筛选应为 2 封，实际 %d", total)
	}
	for _, item := range list {
		for _, f := range item.Flags {
			if f == "Seen" {
				t.Error("未读筛选结果不应包含 Seen 标记")
			}
		}
	}

	list, total, err = svc.MailList("user@example.com", "pwd", "INBOX", 1, 10, "report", nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("关键字搜索结果应为 2 封，实际 %d", total)
	}

	list, _, err = svc.MailList("user@example.com", "pwd", "INBOX", 1, 10, "", []string{"date_desc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("应返回 3 封，实际 %d", len(list))
	}
	if list[0].SendTime.Before(list[1].SendTime) {
		t.Error("date_desc 排序结果不正确")
	}
}

func TestMailListMissingFolder(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	if _, _, err := svc.MailList("user@example.com", "pwd", "NoSuch", 1, 10, "", nil); err == nil {
		t.Fatal("不存在的文件夹应返回错误")
	}
}

func TestStarMailList(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{Subject: "starred inbox"}), `\Flagged`)
	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{Subject: "normal"}))

	draftUID := server.AddMessage("Sent", testutil.BuildMessage(testutil.MessageOptions{Subject: "starred sent"}), `\Flagged`)
	server.AddMessage("Drafts", testutil.BuildMessage(testutil.MessageOptions{Subject: "draft starred"}), `\Flagged`, `\Seen`)

	list, total, err := svc.StarMailList("user@example.com", "pwd", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("星标邮件应为 3 封，实际 total=%d len=%d", total, len(list))
	}

	// 非 INBOX 的星标邮件被强制标记为已读（沿用旧行为）
	for _, item := range list {
		if item.Folder == "Sent" && item.Uid == int64(draftUID) {
			seen := false
			for _, f := range item.Flags {
				if f == "Seen" {
					seen = true
				}
			}
			if !seen {
				t.Error("非 INBOX 的星标邮件应被标记为 Seen")
			}
		}
	}

	// 未读筛选：Drafts 中那封是 Seen，因此只剩 INBOX 一封
	list, total, err = svc.StarMailList("user@example.com", "pwd", "", []string{"unread"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Errorf("未读星标应为 2 封，实际 %d", total)
	}
}

func TestUnreadMailTotal(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{}))
	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{}), `\Seen`)
	server.AddMessage("INBOX", testutil.BuildMessage(testutil.MessageOptions{}), `\Flagged`)
	server.AddMessage("Drafts", testutil.BuildMessage(testutil.MessageOptions{}))

	total, err := svc.UnreadMailTotal("user@example.com", "pwd")
	if err != nil {
		t.Fatal(err)
	}
	if total["INBOX"] != 2 {
		t.Errorf("INBOX 未读应为 2，实际 %d", total["INBOX"])
	}
	if total["Drafts"] != 1 {
		t.Errorf("Drafts 未读应为 1，实际 %d", total["Drafts"])
	}
	if total["Star"] != 1 {
		t.Errorf("Star 未读应为 1，实际 %d", total["Star"])
	}
	if _, ok := total["Junk"]; !ok {
		t.Error("应包含全部标准文件夹的统计")
	}
}

func TestMailListBadCredentials(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	if _, _, err := svc.MailList("user@example.com", "wrong", "INBOX", 1, 10, "", nil); err == nil {
		t.Fatal("错误密码应返回错误")
	}
}
