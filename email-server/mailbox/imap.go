package mailbox

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"sort"
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
		section := &imap.BodySectionName{}
		if body := msg.GetBody(section); body != nil {
			if raw, err := io.ReadAll(body); err == nil {
				m.Raw = raw
			}
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func sortMessages(msgs []Message) {
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].SeqNum < msgs[j].SeqNum })
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
