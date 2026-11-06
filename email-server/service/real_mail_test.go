package service

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealEmlFixture 用仓库中的真实邮件样本做一次解析冒烟测试，确保复杂 MIME 不会 panic。
func TestRealEmlFixture(t *testing.T) {
	path := filepath.Join("..", "test", "test1.eml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("样本文件不存在，跳过: %v", err)
	}

	svc, server, _, _ := newTestService(t)
	uid := server.AddMessage("INBOX", raw)

	list, total, err := svc.MailList("user@example.com", "pwd", "INBOX", 1, 10, "", nil)
	if err != nil {
		t.Fatalf("真实邮件列表解析失败: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("列表结果错误: total=%d len=%d", total, len(list))
	}
	if list[0].Subject == "" {
		t.Error("真实邮件主题不应为空")
	}

	detail, err := svc.MailDetail("user@example.com", "pwd", "INBOX", int64(uid), "localhost:8058")
	if err != nil {
		t.Fatalf("真实邮件详情解析失败: %v", err)
	}
	if detail.Content == "" {
		t.Error("真实邮件正文不应为空")
	}
}
