package mailbox

import (
	"testing"

	"github.com/emersion/go-imap"
)

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

// TestSectionFetchItem 验证分段抓取的 IMAP 语法。
func TestSectionFetchItem(t *testing.T) {
	cases := []struct {
		name string
		sec  Section
		want string
	}{
		{"整封", Section{}, "BODY.PEEK[]"},
		{"单段正文", Section{Path: "", Offset: 0, Length: 100}, "BODY.PEEK[1]<0.100>"},
		{"整部件", Section{Path: "3"}, "BODY.PEEK[3]"},
		{"部件偏移", Section{Path: "4", Offset: 100, Length: 120}, "BODY.PEEK[4]<100.120>"},
	}
	for _, c := range cases {
		if got := string(sectionFetchItem(c.sec)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// TestBodySectionKey 防止同一部件的多次分段抓取在结果 map 中互相覆盖。
func TestBodySectionKey(t *testing.T) {
	full := &imap.BodySectionName{BodyPartName: imap.BodyPartName{Path: []int{4}}}
	if got := bodySectionKey(full); got != "4" {
		t.Fatalf("整部件键错误: %q", got)
	}
	head := &imap.BodySectionName{BodyPartName: imap.BodyPartName{Path: []int{4}}, Partial: []int{0}}
	if got := bodySectionKey(head); got != "4@0" {
		t.Fatalf("头部样本键错误: %q", got)
	}
	tail := &imap.BodySectionName{BodyPartName: imap.BodyPartName{Path: []int{4}}, Partial: []int{2139030}}
	if got := bodySectionKey(tail); got != "4@2139030" {
		t.Fatalf("尾部样本键错误: %q", got)
	}
}

// TestToPart 验证 BODYSTRUCTURE 转换后的类型、路径、文件名与 Content-ID。
func TestToPart(t *testing.T) {
	bs := &imap.BodyStructure{
		MIMEType: "multipart", MIMESubType: "MIXED",
		Parts: []*imap.BodyStructure{
			{MIMEType: "TEXT", MIMESubType: "PLAIN", Encoding: "BASE64", Size: 272},
			{
				MIMEType: "image", MIMESubType: "png", Encoding: "base64", Size: 91520,
				Id: "<abc@example.com>", Disposition: "INLINE",
				DispositionParams: map[string]string{"filename": "inline.png"},
			},
			{
				MIMEType: "application", MIMESubType: "pdf", Encoding: "base64", Size: 28462,
				Disposition: "attachment",
				Params:      map[string]string{"name": "report.pdf"},
			},
		},
	}

	root := toPart(bs, "")
	if root.Type != "multipart/mixed" || len(root.Children) != 3 {
		t.Fatalf("根节点转换错误: %+v", root)
	}
	text := root.Children[0]
	if text.Path != "1" || text.Type != "text/plain" || text.Encoding != "base64" || text.Size != 272 {
		t.Fatalf("文本节点转换错误: %+v", text)
	}
	inline := root.Children[1]
	if inline.Path != "2" || inline.Type != "image/png" || inline.Disposition != "inline" {
		t.Fatalf("内联节点转换错误: %+v", inline)
	}
	if inline.ContentID != "abc@example.com" || inline.FileName != "inline.png" {
		t.Fatalf("Content-ID/文件名错误: %q %q", inline.ContentID, inline.FileName)
	}
	pdf := root.Children[2]
	if pdf.Path != "3" || pdf.FileName != "report.pdf" || pdf.Disposition != "attachment" {
		t.Fatalf("附件节点转换错误: %+v", pdf)
	}
}

// TestPartFileName 验证文件名取值优先级：Content-Disposition 优先于 Content-Type。
func TestPartFileName(t *testing.T) {
	bs := &imap.BodyStructure{
		DispositionParams: map[string]string{"Filename": "a.txt"},
		Params:            map[string]string{"name": "b.txt"},
	}
	if got := partFileName(bs); got != "a.txt" {
		t.Fatalf("应优先取 disposition 文件名: %q", got)
	}
	if got := partFileName(&imap.BodyStructure{Params: map[string]string{"NAME": "b.txt"}}); got != "b.txt" {
		t.Fatalf("应回退到 Content-Type 的 name: %q", got)
	}
	if got := partFileName(&imap.BodyStructure{}); got != "" {
		t.Fatalf("无文件名时应返回空: %q", got)
	}
}

func TestStructureHasAttachment(t *testing.T) {
	cases := []struct {
		name string
		bs   *imap.BodyStructure
		want bool
	}{
		{"nil", nil, false},
		{"纯文本", &imap.BodyStructure{MIMEType: "text", MIMESubType: "plain"}, false},
		{"直接附件", &imap.BodyStructure{MIMEType: "application", MIMESubType: "octet-stream", Disposition: "attachment"}, true},
		{"内联图片", &imap.BodyStructure{MIMEType: "image", MIMESubType: "png", Disposition: "inline"}, false},
		{"嵌套附件", &imap.BodyStructure{
			MIMEType: "multipart", MIMESubType: "mixed",
			Parts: []*imap.BodyStructure{
				{MIMEType: "text", MIMESubType: "plain"},
				{MIMEType: "application", MIMESubType: "pdf", Disposition: "ATTACHMENT"},
			},
		}, true},
		{"转发邮件内嵌附件", &imap.BodyStructure{
			MIMEType: "message", MIMESubType: "rfc822",
			BodyStructure: &imap.BodyStructure{MIMEType: "application", MIMESubType: "pdf", Disposition: "attachment"},
		}, true},
	}

	for _, c := range cases {
		if got := structureHasAttachment(c.bs); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
