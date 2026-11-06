package service

import (
	"bytes"
	"encoding/base64"
	"mime/quotedprintable"
	"strings"
	"testing"

	"email-server/mailbox"

	"github.com/jhillyerd/enmime"
)

// wrapBase64 按 76 列折行，可选择是否在末尾补 CRLF（两种真实邮件都存在）。
func wrapBase64(enc string, trailingNewline bool) string {
	var b strings.Builder
	for i := 0; i < len(enc); i += 76 {
		if i > 0 {
			b.WriteString("\r\n")
		}
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end])
	}
	if trailingNewline {
		b.WriteString("\r\n")
	}
	return b.String()
}

func TestBase64DecodedSize(t *testing.T) {
	data := bytes.Repeat([]byte("abcde"), 500) // 2500 字节，末行不足 76 列且带 padding
	enc := base64.StdEncoding.EncodeToString(data)

	for _, trailing := range []bool{false, true} {
		raw := wrapBase64(enc, trailing)
		head := []byte(raw[:detailHeadSampleBytes])
		tail := []byte(raw[len(raw)-detailTailSampleBytes:])
		got, ok := base64DecodedSize(int64(len(raw)), head, tail)
		if !ok {
			t.Fatalf("trailing=%v 推断失败", trailing)
		}
		if got != int64(len(data)) {
			t.Fatalf("trailing=%v 推断=%d 实际=%d", trailing, got, len(data))
		}
	}
}

func TestBase64DecodedSizeUnwrapped(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 1000)
	enc := base64.StdEncoding.EncodeToString(data)
	head := []byte(enc[:detailHeadSampleBytes])
	tail := []byte(enc[len(enc)-detailTailSampleBytes:])
	got, ok := base64DecodedSize(int64(len(enc)), head, tail)
	if !ok || got != int64(len(data)) {
		t.Fatalf("不折行推断错误: ok=%v got=%d want=%d", ok, got, len(data))
	}
}

func TestBase64DecodedSizeRejectsInvalidSamples(t *testing.T) {
	if _, ok := base64DecodedSize(100, nil, nil); ok {
		t.Fatal("空样本不应推断成功")
	}
	enc := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("y"), 900))
	raw := wrapBase64(enc, false)
	// 尾部样本只有 10 字节，无法确认末行长度。
	if _, ok := base64DecodedSize(int64(len(raw)), []byte(raw[:200]), []byte(raw[len(raw)-10:])); ok {
		t.Fatal("过短的尾部样本不应推断成功")
	}
}

func TestSelectDetailContent(t *testing.T) {
	plain := &mailbox.Part{Path: "1.1", Type: "text/plain", Size: 272}
	html := &mailbox.Part{Path: "1.2", Type: "text/html", Size: 1804}
	alt := &mailbox.Part{Path: "1", Type: "multipart/alternative", Children: []*mailbox.Part{plain, html}}
	root := &mailbox.Part{Path: "", Type: "multipart/mixed", Children: []*mailbox.Part{alt}}

	decoded := map[string]string{"1.1": "纯文本", "1.2": "<p>HTML</p>"}
	if got := selectDetailContent(root, decoded); got != "<p>HTML</p>" {
		t.Fatalf("应优先返回 HTML: %q", got)
	}

	decoded = map[string]string{"1.1": "纯文本", "1.2": ""}
	if got := selectDetailContent(root, decoded); got != "纯文本" {
		t.Fatalf("HTML 为空时应返回纯文本: %q", got)
	}

	// 非 alternative 的多段纯文本按 DFS 顺序以 "\n--\n" 连接。
	mixed := &mailbox.Part{Path: "", Type: "multipart/mixed", Children: []*mailbox.Part{
		{Path: "1", Type: "text/plain"},
		{Path: "2", Type: "text/plain"},
	}}
	if got := selectDetailContent(mixed, map[string]string{"1": "a", "2": "b"}); got != "a\n--\nb" {
		t.Fatalf("多段纯文本拼接错误: %q", got)
	}
}

func TestBuildDetailPlan(t *testing.T) {
	root := &mailbox.Part{
		Path: "", Type: "multipart/mixed",
		Children: []*mailbox.Part{
			{
				Path: "1", Type: "multipart/alternative",
				Children: []*mailbox.Part{
					{Path: "1.1", Type: "text/plain", Encoding: "quoted-printable", Size: 272},
					{Path: "1.2", Type: "text/html", Encoding: "base64", Size: 1804},
				},
			},
			{Path: "2", Type: "image/png", Encoding: "base64", Size: 91520, Disposition: "inline", ContentID: "abc"},
			{Path: "3", Type: "application/pdf", Encoding: "base64", Size: 28462, Disposition: "attachment", FileName: "a.pdf"},
			{Path: "4", Type: "text/plain", Encoding: "base64", Size: 32, Disposition: "attachment", FileName: "a.txt"},
		},
	}

	plan, ok := buildDetailPlan(root)
	if !ok {
		t.Fatal("结构应支持分段路径")
	}
	if len(plan.textBodies) != 2 {
		t.Fatalf("正文部件数量错误: %d", len(plan.textBodies))
	}
	if len(plan.attachments) != 2 || len(plan.inlines) != 1 {
		t.Fatalf("附件/内联数量错误: %d/%d", len(plan.attachments), len(plan.inlines))
	}

	sections := map[string]bool{}
	for _, sec := range plan.sections {
		sections[sec.Key()] = true
	}
	for _, want := range []string{"1.1", "1.2", "2", "3@0", "3@28342", "4"} {
		if !sections[want] {
			t.Errorf("缺少分段: %s", want)
		}
	}
}

func TestDetailStructureSupported(t *testing.T) {
	text := &mailbox.Part{Path: "", Type: "text/plain", Encoding: "quoted-printable", Params: map[string]string{"charset": "utf-8"}}
	if !detailStructureSupported(text) {
		t.Fatal("单段 UTF-8 纯文本应支持")
	}

	cases := []struct {
		name     string
		root     *mailbox.Part
		rejected bool
	}{
		{"内嵌邮件", &mailbox.Part{Type: "message/rfc822"}, false},
		{"未知编码", &mailbox.Part{Type: "text/plain", Encoding: "x-unknown"}, false},
		{"非 UTF-8 正文", &mailbox.Part{Type: "text/plain", Params: map[string]string{"charset": "gbk"}}, false},
		{"QP 附件", &mailbox.Part{Type: "application/pdf", Encoding: "quoted-printable", Disposition: "attachment"}, false},
		{"空 multipart", &mailbox.Part{Type: "multipart/mixed"}, true},
		{"单段图片", &mailbox.Part{Type: "image/png"}, false},
	}
	for _, c := range cases {
		if got := detailStructureSupported(c.root); got == c.rejected {
			t.Errorf("%s: 期望支持=%v，实际=%v", c.name, !c.rejected, got)
		}
	}
}

func TestDetailAttachmentSize(t *testing.T) {
	data := bytes.Repeat([]byte("z"), 3000)
	enc := base64.StdEncoding.EncodeToString(data)
	part := &mailbox.Part{Path: "3", Type: "application/pdf", Encoding: "base64", Size: int64(len(enc))}

	if size, ok := detailAttachmentSize(part, map[string][]byte{"3": []byte(enc)}, nil); !ok || size != int64(len(data)) {
		t.Fatalf("整段解码尺寸错误: ok=%v size=%d", ok, size)
	}

	raw := wrapBase64(enc, true)
	part.Size = int64(len(raw))
	head := []byte(raw[:detailHeadSampleBytes])
	tail := []byte(raw[len(raw)-detailTailSampleBytes:])
	samples := map[string][]byte{
		mailbox.Section{Path: "3", Offset: 0, Length: detailHeadSampleBytes}.Key():                                 head,
		mailbox.Section{Path: "3", Offset: part.Size - detailTailSampleBytes, Length: detailTailSampleBytes}.Key(): tail,
	}
	if size, ok := detailAttachmentSize(part, samples, nil); !ok || size != int64(len(data)) {
		t.Fatalf("采样推断尺寸错误: ok=%v size=%d", ok, size)
	}
}

// TestDecodeDetailTextExact 验证非 UTF-8 正文与 enmime 整封解析结果一致。
func TestDecodeDetailTextExact(t *testing.T) {
	body := []byte("caf\xe9 na\xefve")
	part := &mailbox.Part{
		Path: "1", Type: "text/plain", Encoding: "8bit",
		Params: map[string]string{"charset": "iso-8859-1"},
	}
	got, err := decodeDetailTextExact(part, body)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	full := "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=iso-8859-1\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" + string(body)
	env, err := enmime.ReadEnvelope(strings.NewReader(full))
	if err != nil {
		t.Fatalf("整封解析失败: %v", err)
	}
	if string(got) != env.Text {
		t.Fatalf("字符集转换结果不一致: got=%q want=%q", got, env.Text)
	}
}

// TestDecodeDetailTextExactRootHTMLMeta 验证单段 HTML 在无 charset 参数时仍按 meta 转换。
func TestDecodeDetailTextExactRootHTMLMeta(t *testing.T) {
	body := []byte(`<html><head><meta charset="iso-8859-1"></head><body>caf` + "\xe9" + `</body></html>`)
	part := &mailbox.Part{Path: "1", Type: "text/html", Encoding: "8bit"}
	got, err := decodeDetailTextExact(part, body)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	full := "MIME-Version: 1.0\r\nContent-Type: text/html\r\nContent-Transfer-Encoding: 8bit\r\n\r\n" + string(body)
	env, err := enmime.ReadEnvelope(strings.NewReader(full))
	if err != nil {
		t.Fatalf("整封解析失败: %v", err)
	}
	if string(got) != env.HTML {
		t.Fatalf("HTML meta 字符集转换不一致: got=%q want=%q", got, env.HTML)
	}
	if !strings.Contains(string(got), "café") {
		t.Fatalf("HTML meta 未转换为 UTF-8: %q", got)
	}
}

// TestDetailAttachmentSizeQuotedPrintable 验证 QP 附件按 enmime 的软换行规则计算尺寸。
func TestDetailAttachmentSizeQuotedPrintable(t *testing.T) {
	data := append(bytes.Repeat([]byte{0x00, 0xff, 'A'}, 1000), []byte("中文")...)
	var buf bytes.Buffer
	qw := quotedprintable.NewWriter(&buf)
	if _, err := qw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := qw.Close(); err != nil {
		t.Fatal(err)
	}
	part := &mailbox.Part{
		Path: "2", Type: "application/octet-stream", Encoding: "quoted-printable",
		Disposition: "attachment", Size: int64(buf.Len()),
	}
	decoded, err := decodeDetailPartExact(part, buf.Bytes())
	if err != nil {
		t.Fatalf("QP 解码失败: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Fatalf("QP 解码内容不一致: got %d bytes want %d", len(decoded), len(data))
	}
	size, ok := detailAttachmentSize(part, map[string][]byte{"2": buf.Bytes()}, nil)
	if !ok || size != int64(len(data)) {
		t.Fatalf("QP 附件尺寸错误: ok=%v size=%d want=%d", ok, size, len(data))
	}
}

// TestBuildDetailPlanRootLeaf 覆盖单段邮件的正文、附件、内联与根路径归一。
func TestBuildDetailPlanRootLeaf(t *testing.T) {
	rootText := &mailbox.Part{
		Path: "1", Type: "text/plain", Encoding: "8bit",
		Params: map[string]string{"charset": "gbk"}, Size: 10,
	}
	plan, ok := buildDetailPlan(rootText)
	if !ok || len(plan.textBodies) != 1 || plan.rootAttachment != nil {
		t.Fatalf("单段正文规划错误: ok=%v plan=%+v", ok, plan)
	}
	if len(plan.sections) != 1 || plan.sections[0].Path != "1" {
		t.Fatalf("单段正文分段错误: %+v", plan.sections)
	}

	rootAttachment := &mailbox.Part{
		Path: "1", Type: "image/png", Encoding: "base64",
		Disposition: "attachment", FileName: "a.png", Size: 100,
	}
	plan, ok = buildDetailPlan(rootAttachment)
	if !ok || plan.rootAttachment != rootAttachment || len(plan.sections) != 1 {
		t.Fatalf("单段附件规划错误: ok=%v plan=%+v", ok, plan)
	}

	rootInline := &mailbox.Part{
		Path: "1", Type: "image/gif", Encoding: "base64", Disposition: "inline", ContentID: "x", Size: 100,
	}
	plan, ok = buildDetailPlan(rootInline)
	if !ok || plan.rootInline != rootInline || len(plan.contentParts) != 1 {
		t.Fatalf("单段内联规划错误: ok=%v plan=%+v", ok, plan)
	}

	// 无 disposition 的非文本根仍按 enmime 规则计入附件，即使带 Content-ID。
	rootPlain := &mailbox.Part{
		Path: "1", Type: "application/pdf", Encoding: "base64", ContentID: "doc", Size: 100,
	}
	plan, ok = buildDetailPlan(rootPlain)
	if !ok || plan.rootAttachment != rootPlain || len(plan.contentParts) != 0 {
		t.Fatalf("无 disposition 根附件规划错误: ok=%v plan=%+v", ok, plan)
	}
}

// TestBuildDetailPlanQuotedPrintableAttachment 验证 QP 附件直接整段抓取、未知编码不需要抓取。
func TestBuildDetailPlanQuotedPrintableAttachment(t *testing.T) {
	qpPart := &mailbox.Part{
		Path: "2", Type: "application/pdf", Encoding: "quoted-printable",
		Disposition: "attachment", FileName: "a.pdf", Size: 200000,
	}
	plan, ok := buildDetailPlan(qpPart)
	if !ok {
		t.Fatal("QP 附件结构应支持")
	}
	if plan.rootAttachment != qpPart || len(plan.sections) != 1 || plan.sections[0].Key() != "2" {
		t.Fatalf("QP 附件应抓取整段: %+v", plan.sections)
	}

	unknown := &mailbox.Part{
		Path: "2", Type: "application/octet-stream", Encoding: "x-custom",
		Disposition: "attachment", FileName: "a.bin", Size: 20,
	}
	plan, ok = buildDetailPlan(unknown)
	if !ok || plan.rootAttachment != unknown || len(plan.sections) != 0 {
		t.Fatalf("未知编码附件不应抓取正文: ok=%v sections=%+v", ok, plan.sections)
	}
	if size, ok := detailAttachmentSize(unknown, nil, nil); !ok || size != 20 {
		t.Fatalf("未知编码附件尺寸应取服务端大小: ok=%v size=%d", ok, size)
	}
}
