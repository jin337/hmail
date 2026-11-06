package utils

import (
	"email-server/model"
	"testing"
	"time"
)

func TestFormatMailName(t *testing.T) {
	addr, info, err := FormatMailName("张三 <zhangsan@example.com>, Li Si <li@example.com>")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if addr == nil || *addr != "zhangsan@example.com, li@example.com" {
		t.Errorf("地址解析错误: %v", addr)
	}
	if len(info) != 2 || info[0].Name != "张三" || info[1].Name != "Li Si" {
		t.Errorf("信息解析错误: %+v", info)
	}

	// 无显示名时回退为邮箱前缀
	_, info, err = FormatMailName("plain@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if info[0].Name != "plain" {
		t.Errorf("默认名称错误: %q", info[0].Name)
	}

	if addr, info, err := FormatMailName(""); err != nil || addr != nil || info != nil {
		t.Errorf("空串应返回 nil: %v %v %v", addr, info, err)
	}

	if _, _, err := FormatMailName("not an address"); err == nil {
		t.Error("非法地址应返回错误")
	}
}

func TestFormatUnitSizeAndFormatize(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1024 * 1024, "1.0MB"},
		{2 * 1024 * 1024 * 1024, "2.0GB"},
	}
	for _, c := range cases {
		if got := FormatUnitSize(c.size); got != c.want {
			t.Errorf("FormatUnitSize(%d)=%q, want %q", c.size, got, c.want)
		}
	}

	if got := Formatize("1.5KB"); got != 1536 {
		t.Errorf("Formatize(1.5KB)=%d", got)
	}
	if got := Formatize("2MB"); got != 2*1024*1024 {
		t.Errorf("Formatize(2MB)=%d", got)
	}
	if got := Formatize("bad"); got != 0 {
		t.Errorf("非法输入应返回 0，实际 %d", got)
	}
}

func TestPreviewText(t *testing.T) {
	got := PreviewText("  Hello   *world*\r\n&lt;b&gt; ", true)
	if got != "Helloworld<b>" {
		t.Errorf("PreviewText 结果错误: %q", got)
	}

	// 星标列表沿用旧行为：不反转义尖括号
	if got := PreviewText("a &lt; b", false); got != "a&lt;b" {
		t.Errorf("不反转义结果错误: %q", got)
	}
}

func TestFormatFilterAndSort(t *testing.T) {
	f := FormatFilter([]string{"unread", "date_asc", "date_desc", "size_desc"})
	if !f.Unread {
		t.Error("unread 未生效")
	}
	if !f.DateAsc || f.DateDesc || f.SizeDesc {
		t.Errorf("排序条件应只保留优先级最高的一项: %+v", f)
	}

	now := time.Now()
	list := []*model.MailItem{
		{SendTime: now.Add(-time.Hour), Size: "10B"},
		{SendTime: now, Size: "2KB"},
		{SendTime: now.Add(-2 * time.Hour), Size: "1KB"},
	}

	SortMailList(list, model.MailFilter{DateDesc: true})
	if !list[0].SendTime.Equal(now) {
		t.Error("按时间降序排序失败")
	}

	SortMailList(list, model.MailFilter{SizeAsc: true})
	if Formatize(list[0].Size) != 10 {
		t.Error("按大小升序排序失败")
	}

	// 未指定排序条件时保持原顺序
	before := list[0]
	SortMailList(list, model.MailFilter{})
	if list[0] != before {
		t.Error("无排序条件时不应改变顺序")
	}
}

func TestCheckRequiredAndHelpers(t *testing.T) {
	if err := CheckRequired(Required("Email", "a@b.com"), Required("Password", "x")); err != nil {
		t.Fatalf("校验通过却报错: %v", err)
	}
	err := CheckRequired(Required("Email", "  "))
	if err == nil || err.Error() != "Email 参数不能为空" {
		t.Errorf("错误消息不符合约定: %v", err)
	}

	if ext := FileExtension("报告.PDF"); ext != "pdf" {
		t.Errorf("扩展名错误: %q", ext)
	}
	if ext := FileExtension("noext"); ext != "" {
		t.Errorf("无扩展名应返回空串: %q", ext)
	}

	if got := SplitAddressList(" a@x.com , b@x.com ,, "); len(got) != 2 || got[1] != "b@x.com" {
		t.Errorf("地址拆分错误: %v", got)
	}
	if got := SplitAddressList("  "); got != nil {
		t.Errorf("空地址应返回 nil: %v", got)
	}
}

func TestParseMailDate(t *testing.T) {
	tm, err := ParseMailDate("Mon, 02 Jan 2006 15:04:05 +0800")
	if err != nil {
		t.Fatal(err)
	}
	if tm.Year() != 2006 {
		t.Errorf("年份错误: %d", tm.Year())
	}

	if _, err := ParseMailDate("Mon, 02 Jan 2006 15:04:05 UTC"); err != nil {
		t.Errorf("UTC 时间解析失败: %v", err)
	}
	if _, err := ParseMailDate("2006-01-02 15:04:05"); err != nil {
		t.Errorf("本地时间解析失败: %v", err)
	}
	if _, err := ParseMailDate(""); err == nil {
		t.Error("空串应报错")
	}
	if _, err := ParseMailDate("garbage"); err == nil {
		t.Error("非法格式应报错")
	}
}

func TestStripHTML(t *testing.T) {
	if got := StripHTML("<p>a&nbsp;b</p><br/>c"); got != "a bc" {
		t.Errorf("StripHTML 结果错误: %q", got)
	}
}
