// Package mailbox 封装邮件协议访问：IMAP 邮箱操作与 SMTP 投递。
//
// 通过 Client/Factory/Sender 接口把外部邮件服务器隔离开，
// 使 service 层可以在测试中使用内存实现，而无需真实 IMAP/SMTP 服务。
package mailbox

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound 表示目标邮件/附件不存在。
var ErrNotFound = errors.New("邮件不存在")

// ErrAuth 表示账号或密码不正确。
var ErrAuth = errors.New("用户名或密码错误")

// ErrUnsupported 表示当前实现不支持该操作（例如测试用的内存实现未模拟 MIME 分段）。
var ErrUnsupported = errors.New("当前实现不支持该操作")

// Query 描述一次服务端搜索条件。
type Query struct {
	Text         string   // 全文/主题/发件人关键字
	WithFlags    []string // 必须包含的标记，如 \Flagged
	WithoutFlags []string // 必须不包含的标记，如 \Seen
}

// Message 是抓取到的原始邮件及其元信息。
type Message struct {
	SeqNum uint32
	UID    uint32
	Flags  []string
	Size   uint32
	Raw    []byte
	// Attachment 来自 IMAP BODYSTRUCTURE，用于在不下载附件的情况下判断是否含附件。
	// 为 nil 表示该抓取方式没有获取结构信息。
	Attachment *bool
}

// Part 描述 BODYSTRUCTURE 中的一个 MIME 部件。
// Path 是 IMAP section 编号（如 "1"、"1.1"），根部件为空字符串。
type Part struct {
	Path        string
	Type        string // 小写 MIME 类型，如 text/html、image/png
	Params      map[string]string
	Encoding    string // Content-Transfer-Encoding，如 base64、quoted-printable
	Disposition string // 小写，如 attachment、inline
	FileName    string
	ContentID   string
	Size        int64 // 服务端报告的编码后字节数
	Children    []*Part
}

// IsMultipart 判断是否为 multipart 容器。
func (p *Part) IsMultipart() bool {
	return p != nil && strings.HasPrefix(p.Type, "multipart/")
}

// Section 描述一次 MIME 分段抓取请求。
// Length 为 0 表示抓取整段；否则从 Offset 起抓取 Length 字节。
type Section struct {
	Path   string
	Offset int64
	Length int64
}

// Key 返回本次分段在 FetchSections 结果中的键。
func (s Section) Key() string {
	if s.Length <= 0 {
		return s.Path
	}
	return fmt.Sprintf("%s@%d", s.Path, s.Offset)
}

// Client 是一次已登录的 IMAP 会话。
// 所有方法均作用于最近一次 Select 选中的文件夹。
type Client interface { // Select 选中文件夹，文件夹不存在时返回错误。
	Select(folder string) error
	// Search 按条件搜索，返回序列号（升序）。
	Search(q Query) ([]uint32, error)
	// Fetch 按序列号批量抓取完整邮件（不改变已读状态）。
	Fetch(seqNums []uint32) ([]Message, error)
	// FetchPreview 按序列号批量抓取邮件前 maxBytes 字节（头部与正文开头），
	// 用于列表预览，避免为显示摘要下载大附件。
	FetchPreview(seqNums []uint32, maxBytes int64) ([]Message, error)
	// FetchAll 抓取当前文件夹的全部邮件（用于启动后恢复定时任务）。
	FetchAll() ([]Message, error)
	// FetchUID 按 UID 抓取单封完整邮件。
	FetchUID(uid uint32) (*Message, error)
	// FetchStructure 只抓取邮件 MIME 结构（BODYSTRUCTURE），不下载正文与附件。
	FetchStructure(uid uint32) (*Part, error)
	// FetchSections 按 section 抓取指定 MIME 分段（可带偏移），不下载未请求的部分。
	FetchSections(uid uint32, sections []Section) (map[string][]byte, error)
	// UIDSearchByHeader 按邮件头搜索，返回 UID 列表。
	UIDSearchByHeader(field, value string) ([]uint32, error)
	// StoreFlags 为指定 UID 添加（add=true）或移除标记。
	StoreFlags(uids []uint32, add bool, flags []string) error
	// Move 把指定 UID 移动到目标文件夹。
	Move(uids []uint32, dest string) error
	// DeleteForever 把指定 UID 标记删除并执行 EXPUNGE。
	DeleteForever(uids []uint32) error
	// Append 追加原始邮件到文件夹。
	Append(folder string, flags []string, date time.Time, raw []byte) error
	// CreateFolder 创建文件夹（已存在时由实现方决定是否忽略）。
	CreateFolder(folder string) error
	// Logout 关闭会话。
	Logout() error
}

// Factory 建立已登录的 IMAP 会话。
type Factory interface {
	Connect(email, password string) (Client, error)
}

// Sender 通过 SMTP 投递原始邮件。
type Sender interface {
	Send(from, password string, to, cc []string, raw []byte) error
}
