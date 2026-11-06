package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"email-server/contacts"
	"email-server/internal/testutil"

	"github.com/jhillyerd/enmime"
)

func TestSendEmailImmediately(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	msg, err := svc.SendEmail("user@example.com", "pwd", Compose{
		From:    "user@example.com",
		To:      []string{"friend@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "hello",
		Body:    "<p>hi</p>",
	})
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if msg != "发送成功" {
		t.Errorf("提示文案错误: %q", msg)
	}

	last, ok := sender.Last()
	if !ok {
		t.Fatal("SMTP 未收到邮件")
	}
	if last.From != "user@example.com" || last.Password != "pwd" {
		t.Errorf("发信账号错误: %+v", last)
	}
	if len(last.To) != 1 || last.To[0] != "friend@example.com" {
		t.Errorf("收件人错误: %v", last.To)
	}
	if len(last.Cc) != 1 || last.Cc[0] != "cc@example.com" {
		t.Errorf("抄送错误: %v", last.Cc)
	}

	// 已发送文件夹应有 1 封，并带 Seen 标记
	if server.FolderLen("Sent") != 1 {
		t.Errorf("已发送邮件数错误: %d", server.FolderLen("Sent"))
	}
	sent := server.Messages("Sent")
	if !strings.Contains(strings.Join(sent[0].Flags, ","), `\Seen`) {
		t.Errorf("已发送邮件应带 Seen 标记: %v", sent[0].Flags)
	}

	// 最近联系人已记录
	list, _, err := svc.ContactList(contacts.PrefixSent, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Email != "friend@example.com" {
		t.Errorf("最近联系人记录错误: %+v", list)
	}
}

func TestSendEmailFromDraft(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	draftUID := server.AddMessage("Drafts", testutil.BuildMessage(testutil.MessageOptions{
		Subject: "草稿",
	}), `\Draft`)

	msg, err := svc.SendEmail("user@example.com", "pwd", Compose{
		From:      "user@example.com",
		To:        []string{"a@example.com"},
		Subject:   "草稿",
		Body:      "body",
		SourceUID: int64(draftUID),
		Folder:    "Drafts",
	})
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if msg != "发送成功" {
		t.Errorf("提示文案错误: %q", msg)
	}
	if sender.SendCnt != 1 {
		t.Errorf("应发送 1 次，实际 %d", sender.SendCnt)
	}

	// 草稿被移动到已发送（UID 不变）
	if server.FolderLen("Drafts") != 0 {
		t.Errorf("草稿应被移除，剩余 %d", server.FolderLen("Drafts"))
	}
	sent := server.Messages("Sent")
	if len(sent) != 1 || sent[0].UID != draftUID {
		t.Errorf("已发送邮件 UID 错误: %+v", sent)
	}
}

func TestSendEmailEmptyRecipient(t *testing.T) {
	svc, _, sender, _ := newTestService(t)
	if _, err := svc.SendEmail("user@example.com", "pwd", Compose{Subject: "x", Body: "y"}); err == nil {
		t.Fatal("收件人为空应报错")
	}
	if sender.SendCnt != 0 {
		t.Error("收件人为空不应调用 SMTP")
	}
}

func TestSendEmailScheduleFuture(t *testing.T) {
	svc, server, sender, _ := newTestService(t)

	var pending []func()
	svc.SetAsyncRunner(func(fn func()) { pending = append(pending, fn) })
	svc.SetSleeper(func(time.Duration) {})

	future := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	msg, err := svc.SendEmail("user@example.com", "pwd", Compose{
		From:     "user@example.com",
		To:       []string{"a@example.com"},
		Subject:  "future",
		Body:     "<p>later</p>",
		Schedule: future,
	})
	if err != nil {
		t.Fatalf("设置定时失败: %v", err)
	}
	if msg != "定时发送已设置" {
		t.Errorf("提示文案错误: %q", msg)
	}
	if sender.SendCnt != 0 {
		t.Error("定时邮件不应立即发送")
	}
	if server.FolderLen("Drafts") != 1 {
		t.Fatalf("定时邮件应存入草稿，实际 %d", server.FolderLen("Drafts"))
	}
	if len(pending) == 0 {
		t.Fatal("应注册定时任务")
	}

	// 执行定时任务：发送成功并从草稿移动到已发送
	pending[0]()

	if sender.SendCnt != 1 {
		t.Errorf("定时任务应发送 1 次，实际 %d", sender.SendCnt)
	}
	if server.FolderLen("Drafts") != 0 || server.FolderLen("Sent") != 1 {
		t.Errorf("定时发送后归档错误: Drafts=%d Sent=%d", server.FolderLen("Drafts"), server.FolderLen("Sent"))
	}
}

func TestSendWithRetry(t *testing.T) {
	svc, _, sender, _ := newTestService(t)

	var sleeps []time.Duration
	svc.SetSleeper(func(d time.Duration) { sleeps = append(sleeps, d) })

	sender.Errors = []error{strings2Error("smtp down"), strings2Error("smtp down"), strings2Error("smtp down")}

	if err := svc.sendWithRetry("user@example.com", "pwd", []string{"a@example.com"}, nil, []byte("raw")); err == nil {
		t.Fatal("三次失败应最终报错")
	}
	if len(sleeps) != 2 {
		t.Errorf("应重试 2 次，实际等待 %d 次", len(sleeps))
	}
	if sleeps[0] != 5*time.Second || sleeps[1] != 15*time.Second {
		t.Errorf("重试间隔错误: %v", sleeps)
	}
}

func TestSendWithRetrySuccessAfterFailure(t *testing.T) {
	svc, _, sender, _ := newTestService(t)
	svc.SetSleeper(func(time.Duration) {})

	sender.Errors = []error{strings2Error("temporary"), nil}

	if err := svc.sendWithRetry("user@example.com", "pwd", []string{"a@example.com"}, nil, []byte("raw")); err != nil {
		t.Fatalf("第二次应成功: %v", err)
	}
	if sender.SendCnt != 2 {
		t.Errorf("应调用 SMTP 2 次，实际 %d", sender.SendCnt)
	}
}

func TestSaveDraft(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	uid, msg, err := svc.SaveDraft("user@example.com", "pwd", Compose{
		From:    "user@example.com",
		To:      []string{"a@example.com"},
		Subject: "draft subject",
		Body:    "<p>draft</p>",
	})
	if err != nil {
		t.Fatalf("保存草稿失败: %v", err)
	}
	if uid != 0 || msg != "草稿保存成功" {
		t.Errorf("新建草稿返回值错误: uid=%d msg=%q", uid, msg)
	}
	if server.FolderLen("Drafts") != 1 {
		t.Fatalf("草稿数错误: %d", server.FolderLen("Drafts"))
	}

	// 更新草稿：删除旧草稿并追加新草稿
	existingUID := server.Messages("Drafts")[0].UID
	uid, msg, err = svc.SaveDraft("user@example.com", "pwd", Compose{
		From:      "user@example.com",
		To:        []string{"a@example.com"},
		Subject:   "updated",
		Body:      "<p>updated</p>",
		SourceUID: int64(existingUID),
		Folder:    "Drafts",
	})
	if err != nil {
		t.Fatalf("更新草稿失败: %v", err)
	}
	if uid != int64(existingUID) || msg != "草稿更新成功" {
		t.Errorf("更新草稿返回值错误: uid=%d msg=%q", uid, msg)
	}
	if server.FolderLen("Drafts") != 1 {
		t.Errorf("更新后草稿数错误: %d", server.FolderLen("Drafts"))
	}
	if !bytes.Contains(server.Messages("Drafts")[0].Raw, []byte("updated")) {
		t.Error("草稿内容未更新")
	}
}

func TestMailResources(t *testing.T) {
	svc, server, _, _ := newTestService(t)

	raw := testutil.BuildMessage(testutil.MessageOptions{
		Attachments: []testutil.Attachment{
			{FileName: "keep.txt", ContentType: "text/plain", Content: []byte("keep")},
			{FileName: "drop.txt", ContentType: "text/plain", Content: []byte("drop")},
			{FileName: "logo.png", ContentType: "image/png", Content: []byte("png"), Inline: true, CID: "logo@x"},
		},
	})
	uid := server.AddMessage("Drafts", raw)

	env, err := enmimeRead(raw)
	if err != nil {
		t.Fatal(err)
	}
	keepPart := ""
	for _, a := range env.Attachments {
		if a.FileName == "keep.txt" {
			keepPart = a.PartID
		}
	}

	inlines, attaches, err := svc.MailResources("user@example.com", "pwd", "Drafts", int64(uid), keepPart)
	if err != nil {
		t.Fatalf("MailResources 失败: %v", err)
	}
	if len(attaches) != 1 || attaches[0].FileName != "keep.txt" {
		t.Errorf("附件筛选错误: %+v", attaches)
	}
	if len(inlines) == 0 {
		t.Error("应带回内联图片")
	}
}

func TestSaveSentContactsSkipsSelf(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	svc.saveSentContacts([]string{"user@example.com", "other@example.com", ""}, "user@example.com")

	list, _, err := svc.ContactList(contacts.PrefixSent, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Email != "other@example.com" {
		t.Errorf("联系人记录错误: %+v", list)
	}
	if list[0].Name != "other" {
		t.Errorf("联系人名称应回退为邮箱前缀: %q", list[0].Name)
	}
}

// strings2Error 把字符串转为 error，避免测试文件引入 errors 包。
type strings2Error string

func (e strings2Error) Error() string { return string(e) }

// enmimeRead 是 enmime 解析的薄封装。
func enmimeRead(raw []byte) (*enmime.Envelope, error) {
	return enmime.ReadEnvelope(bytes.NewReader(raw))
}
