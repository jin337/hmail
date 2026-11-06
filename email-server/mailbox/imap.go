package mailbox

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
)

// IMAPFactory 通过明文 IMAP 连接 hMailServer，并在连接建立后立即登录。
type IMAPFactory struct {
	Host string
	Port string
}

// NewIMAPFactory 构造 IMAP 会话工厂。
func NewIMAPFactory(host string, port int) *IMAPFactory {
	return &IMAPFactory{Host: host, Port: fmt.Sprint(port)}
}

// Connect 建立连接并登录；连接成功但登录失败时会先登出，避免连接泄漏。
func (f *IMAPFactory) Connect(email, password string) (Client, error) {
	address := net.JoinHostPort(f.Host, f.Port)
	raw, err := imapclient.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("连接IMAP失败: %w", err)
	}
	if err := raw.Login(email, password); err != nil {
		_ = raw.Logout()
		return nil, ErrAuth
	}
	return &imapSession{client: raw}, nil
}

// imapSession 把 go-imap 的 API 收敛到 mailbox.Client 接口。
type imapSession struct {
	client *imapclient.Client
}

func (s *imapSession) Select(folder string) error {
	_, err := s.client.Select(folder, false)
	return err
}

func (s *imapSession) Search(q Query) ([]uint32, error) {
	return s.client.Search(toCriteria(q))
}

func (s *imapSession) UIDSearchByHeader(field, value string) ([]uint32, error) {
	crit := headerCriteria(field, value)
	return s.client.UidSearch(crit)
}

// headerCriteria 构造按邮件头检索的条件。
// SearchCriteria.Header 是 textproto.MIMEHeader（map 类型），零值为 nil，
// 直接 Add 会 panic，因此必须先初始化。
func headerCriteria(field, value string) *imap.SearchCriteria {
	crit := &imap.SearchCriteria{Header: textproto.MIMEHeader{}}
	crit.Header.Add(field, value)
	return crit
}

func (s *imapSession) Fetch(seqNums []uint32) ([]Message, error) {
	if len(seqNums) == 0 {
		return nil, nil
	}
	set := new(imap.SeqSet)
	for _, n := range seqNums {
		set.AddNum(n)
	}
	ch := make(chan *imap.Message, len(seqNums))
	done := make(chan error, 1)
	go func() {
		done <- s.client.Fetch(set, []imap.FetchItem{
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchRFC822Size,
			imap.FetchItem("BODY.PEEK[]"),
		}, ch)
	}()

	msgs := collect(ch)
	if err := <-done; err != nil {
		return nil, err
	}
	sortMessages(msgs)
	return msgs, nil
}

// FetchPreview 抓取报文前 maxBytes 字节，用于列表摘要。
// 只取 BODY[] 的前段，避免为了显示几行预览而传输整封（含附件）邮件。
func (s *imapSession) FetchPreview(seqNums []uint32, maxBytes int64) ([]Message, error) {
	if len(seqNums) == 0 {
		return nil, nil
	}
	if maxBytes <= 0 {
		maxBytes = 65536
	}
	set := new(imap.SeqSet)
	for _, n := range seqNums {
		set.AddNum(n)
	}
	section := imap.FetchItem(fmt.Sprintf("BODY.PEEK[]<0.%d>", maxBytes))
	ch := make(chan *imap.Message, len(seqNums))
	done := make(chan error, 1)
	go func() {
		done <- s.client.Fetch(set, []imap.FetchItem{
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchRFC822Size,
			imap.FetchBodyStructure,
			section,
		}, ch)
	}()

	msgs := collect(ch)
	if err := <-done; err != nil {
		return nil, err
	}
	sortMessages(msgs)
	return msgs, nil
}

// FetchAll 用序列号区间 1:* 抓取当前文件夹的全部邮件，不依赖 SEARCH 条件。
func (s *imapSession) FetchAll() ([]Message, error) {
	set := new(imap.SeqSet)
	set.AddRange(1, 0) // AddRange(1, 0) 表示 1:*

	ch := make(chan *imap.Message, 256)
	done := make(chan error, 1)
	go func() {
		done <- s.client.Fetch(set, []imap.FetchItem{
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchRFC822Size,
			imap.FetchItem("BODY.PEEK[]"),
		}, ch)
	}()

	msgs := collect(ch)
	if err := <-done; err != nil {
		return nil, err
	}
	sortMessages(msgs)
	return msgs, nil
}

func (s *imapSession) FetchUID(uid uint32) (*Message, error) {
	set := new(imap.SeqSet)
	set.AddNum(uid)
	ch := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- s.client.UidFetch(set, []imap.FetchItem{
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchRFC822Size,
			imap.FetchItem("BODY.PEEK[]"),
		}, ch)
	}()

	msgs := collect(ch)
	if err := <-done; err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, ErrNotFound
	}
	return &msgs[0], nil
}

// FetchStructure 只取 BODYSTRUCTURE，用于在不下载报文的情况下规划详情抓取。
func (s *imapSession) FetchStructure(uid uint32) (*Part, error) {
	set := uidSet([]uint32{uid})
	ch := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- s.client.UidFetch(set, []imap.FetchItem{imap.FetchUid, imap.FetchBodyStructure}, ch)
	}()

	var root *Part
	for msg := range ch {
		if msg == nil || msg.BodyStructure == nil {
			continue
		}
		root = toPart(msg.BodyStructure, "")
		// 单段邮件的 BODYSTRUCTURE 没有层级编号，但抓取正文要用 section 1（BODY[1]）；
		// 空路径发出的 BODY[]<0.0> 会被服务端判为非法，因此这里归一为 "1"。
		if root != nil && !root.IsMultipart() {
			root.Path = "1"
		}
	}
	if err := <-done; err != nil {
		return nil, err
	}
	if root == nil {
		return nil, ErrNotFound
	}
	return root, nil
}

// FetchSections 在一次 FETCH 中抓取多个 MIME 分段，避免下载未请求的附件内容。
func (s *imapSession) FetchSections(uid uint32, sections []Section) (map[string][]byte, error) {
	if len(sections) == 0 {
		return map[string][]byte{}, nil
	}

	items := make([]imap.FetchItem, 0, len(sections)+1)
	items = append(items, imap.FetchUid)
	keys := make([]string, 0, len(sections))
	for _, sec := range sections {
		item := sectionFetchItem(sec)
		items = append(items, item)
		keys = append(keys, sec.Key())
	}

	set := uidSet([]uint32{uid})
	ch := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() { done <- s.client.UidFetch(set, items, ch) }()

	out := make(map[string][]byte, len(keys))
	for msg := range ch {
		if msg == nil {
			continue
		}
		for name, literal := range msg.Body {
			if literal == nil {
				continue
			}
			raw, err := io.ReadAll(literal)
			if err != nil {
				continue
			}
			out[bodySectionKey(name)] = raw
		}
	}
	if err := <-done; err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}

// sectionFetchItem 把分段请求转换为 IMAP FETCH 项。
func sectionFetchItem(sec Section) imap.FetchItem {
	path := sec.Path
	if path == "" {
		// 空路径代表整封 BODY[]；带长度的空路径没有合法写法，按单段正文处理。
		if sec.Length > 0 {
			path = "1"
		} else {
			return imap.FetchItem("BODY.PEEK[]")
		}
	}
	if sec.Length <= 0 {
		return imap.FetchItem(fmt.Sprintf("BODY.PEEK[%s]", path))
	}
	return imap.FetchItem(fmt.Sprintf("BODY.PEEK[%s]<%d.%d>", path, sec.Offset, sec.Length))
}

// bodySectionKey 与 Section.Key 保持一致，用于把响应映射回请求。
func bodySectionKey(name *imap.BodySectionName) string {
	parts := make([]string, len(name.Path))
	for i, n := range name.Path {
		parts[i] = fmt.Sprint(n)
	}
	path := strings.Join(parts, ".")
	// go-imap 只为带偏移的分段填充 Partial（长度为 1），按偏移区分同一部件的多次抓取。
	if len(name.Partial) > 0 {
		return fmt.Sprintf("%s@%d", path, name.Partial[0])
	}
	return path
}

// toPart 把 go-imap 的 BODYSTRUCTURE 转成与协议无关的分段模型。
func toPart(bs *imap.BodyStructure, path string) *Part {
	p := &Part{
		Path:        path,
		Type:        strings.ToLower(bs.MIMEType + "/" + bs.MIMESubType),
		Params:      bs.Params,
		Encoding:    strings.ToLower(bs.Encoding),
		Disposition: strings.ToLower(bs.Disposition),
		ContentID:   strings.Trim(strings.TrimSpace(bs.Id), "<>"),
		Size:        int64(bs.Size),
	}
	p.FileName = partFileName(bs)
	for i, child := range bs.Parts {
		childPath := fmt.Sprint(i + 1)
		if path != "" {
			childPath = path + "." + childPath
		}
		p.Children = append(p.Children, toPart(child, childPath))
	}
	return p
}

// partFileName 依次从 Content-Disposition 与 Content-Type 参数中取文件名。
func partFileName(bs *imap.BodyStructure) string {
	for _, m := range []map[string]string{bs.DispositionParams, bs.Params} {
		for key, value := range m {
			if value == "" {
				continue
			}
			if strings.EqualFold(key, "filename") || strings.EqualFold(key, "name") {
				return value
			}
		}
	}
	return ""
}

func (s *imapSession) StoreFlags(uids []uint32, add bool, flags []string) error {
	if len(uids) == 0 {
		return nil
	}
	set := uidSet(uids)
	var op imap.StoreItem = imap.AddFlags
	if !add {
		op = imap.RemoveFlags
	}
	quoted := make([]interface{}, 0, len(flags))
	for _, f := range flags {
		quoted = append(quoted, f)
	}
	return s.client.UidStore(set, op, quoted, nil)
}

func (s *imapSession) Move(uids []uint32, dest string) error {
	if len(uids) == 0 {
		return nil
	}
	return s.client.UidMove(uidSet(uids), dest)
}

func (s *imapSession) DeleteForever(uids []uint32) error {
	if len(uids) == 0 {
		return nil
	}
	set := uidSet(uids)
	if err := s.client.UidStore(set, imap.AddFlags, []interface{}{imap.DeletedFlag}, nil); err != nil {
		return err
	}
	return s.client.Expunge(nil)
}

func (s *imapSession) Append(folder string, flags []string, date time.Time, raw []byte) error {
	return s.client.Append(folder, flags, date, bytes.NewReader(raw))
}

func (s *imapSession) CreateFolder(folder string) error {
	return s.client.Create(folder)
}

func (s *imapSession) Logout() error {
	return s.client.Logout()
}

// collect 读完 channel 并把 imap.Message 转换为 mailbox.Message。
func collect(ch chan *imap.Message) []Message {
	msgs := make([]Message, 0, cap(ch))
	for msg := range ch {
		if msg == nil {
			continue
		}
		m := Message{
			SeqNum: msg.SeqNum,
			UID:    msg.Uid,
			Flags:  msg.Flags,
			Size:   msg.Size,
		}
		if msg.BodyStructure != nil {
			has := structureHasAttachment(msg.BodyStructure)
			m.Attachment = &has
		}
		section := &imap.BodySectionName{}
		if body := msg.GetBody(section); body != nil {
			if raw, err := io.ReadAll(body); err == nil {
				m.Raw = raw
			}
		} else {
			// 部分抓取（BODY[]<0.N>）的 section 带 Partial 偏移，
			// GetBody 精确匹配不到，退化为取任意返回的 body。
			for _, body := range msg.Body {
				if body == nil {
					continue
				}
				if raw, err := io.ReadAll(body); err == nil {
					m.Raw = raw
				}
				break
			}
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func sortMessages(msgs []Message) {
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].SeqNum < msgs[j].SeqNum })
}

// structureHasAttachment 递归检查 BODYSTRUCTURE 中是否存在 disposition=attachment 的部件。
func structureHasAttachment(bs *imap.BodyStructure) bool {
	if bs == nil {
		return false
	}
	if strings.EqualFold(bs.Disposition, "attachment") {
		return true
	}
	for _, part := range bs.Parts {
		if structureHasAttachment(part) {
			return true
		}
	}
	return structureHasAttachment(bs.BodyStructure)
}

func uidSet(uids []uint32) *imap.SeqSet {
	set := new(imap.SeqSet)
	for _, uid := range uids {
		set.AddNum(uid)
	}
	return set
}

func toCriteria(q Query) *imap.SearchCriteria {
	crit := &imap.SearchCriteria{}
	if q.Text != "" {
		crit.Text = []string{q.Text}
	}
	if len(q.WithFlags) > 0 {
		crit.WithFlags = q.WithFlags
	}
	if len(q.WithoutFlags) > 0 {
		crit.WithoutFlags = q.WithoutFlags
	}
	return crit
}
