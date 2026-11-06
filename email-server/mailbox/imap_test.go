package mailbox

import "testing"

// TestHeaderCriteria 防止 SearchCriteria.Header 为 nil 时 Add 触发 panic（真实环境曾复现）。
func TestHeaderCriteria(t *testing.T) {
	crit := headerCriteria("Message-Id", "<abc@example.com>")

	if crit.Header == nil {
		t.Fatal("Header 不应为 nil")
	}
	got := crit.Header.Get("Message-Id")
	if got != "<abc@example.com>" {
		t.Fatalf("Header 值错误: %q", got)
	}
}

func TestToCriteria(t *testing.T) {
	q := Query{Text: "hello", WithFlags: []string{"Seen"}, WithoutFlags: []string{"Deleted"}}
	crit := toCriteria(q)

	if len(crit.Text) != 1 || crit.Text[0] != "hello" {
		t.Fatalf("Text 条件错误: %v", crit.Text)
	}
	if len(crit.WithFlags) != 1 || crit.WithFlags[0] != "Seen" {
		t.Fatalf("WithFlags 条件错误: %v", crit.WithFlags)
	}
	if len(crit.WithoutFlags) != 1 || crit.WithoutFlags[0] != "Deleted" {
		t.Fatalf("WithoutFlags 条件错误: %v", crit.WithoutFlags)
	}
}
