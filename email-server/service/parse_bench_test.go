package service

import (
	"testing"

	"email-server/internal/testutil"
	"email-server/mailbox"
)

// BenchmarkParseMailItem 衡量列表解析单封邮件的 CPU/内存成本。
// 分别对比轻量邮件与带 1MB 附件的邮件。
func BenchmarkParseMailItem(b *testing.B) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"plain-2KB", testutil.BuildMessage(testutil.MessageOptions{Subject: "hello", Text: "short body"})},
		{"attach-1MB", testutil.BuildMessage(testutil.MessageOptions{
			Subject: "with attachment",
			Text:    "short body",
			Attachments: []testutil.Attachment{
				{FileName: "big.bin", ContentType: "application/octet-stream", Content: make([]byte, 1<<20)},
			},
		})},
	}
	for _, c := range cases {
		msg := mailbox.Message{UID: 1, Raw: c.raw, Size: uint32(len(c.raw))}
		opts := itemOptions{folder: "INBOX", unescape: true, keepDollar: true}
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if parseMailItem(msg, opts) == nil {
					b.Fatal("解析失败")
				}
			}
		})
	}
}
