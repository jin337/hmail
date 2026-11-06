// Package testutil 提供内存版的 IMAP/SMTP/账号目录实现，用于单元测试与接口测试。
package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"email-server/directory"
	"email-server/mailbox"
	"email-server/utils"

	"github.com/jhillyerd/enmime"
)

// Message 是内存邮箱中的一封邮件。
type Message struct {
	UID   uint32
	Raw   []byte
	Flags []string
	Date  time.Time
}

// Server 是内存版 IMAP 服务，可被多个 Client 视图共享。
type Server struct {
	mu       sync.Mutex
	folders  map[string][]*Message
	nextUID  uint32
	accounts map[string]string // email -> password
	Connects int
	Selects  int
}

// NewServer 创建内存邮箱，默认带一个账号和 5 个标准文件夹。
func NewServer(email, password string) *Server {
	s := &Server{
		folders:  map[string][]*Message{},
		nextUID:  1,
		accounts: map[string]string{email: password},
	}
	for _, f := range []string{"INBOX", "Sent", "Drafts", "Deleted", "Junk"} {
		s.folders[f] = nil
	}
	return s
}

// AddAccount 增加可登录账号。
func (s *Server) AddAccount(email, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[email] = password
}

// AddMessage 直接向文件夹追加邮件并返回 UID。
func (s *Server) AddMessage(folder string, raw []byte, flags ...string) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.add(folder, raw, flags, time.Now())
}

func (s *Server) add(folder string, raw []byte, flags []string, date time.Time) uint32 {
	uid := s.nextUID
	s.nextUID++
	s.folders[folder] = append(s.folders[folder], &Message{UID: uid, Raw: raw, Flags: append([]string{}, flags...), Date: date})
	return uid
}

// Messages 返回文件夹内邮件的快照（按加入顺序）。
func (s *Server) Messages(folder string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Message, 0, len(s.folders[folder]))
	for _, m := range s.folders[folder] {
		out = append(out, Message{UID: m.UID, Raw: m.Raw, Flags: append([]string{}, m.Flags...), Date: m.Date})
	}
	return out
}

// FolderLen 返回文件夹邮件数。
func (s *Server) FolderLen(folder string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.folders[folder])
}

// Connect 实现 mailbox.Factory。
func (s *Server) Connect(email, password string) (mailbox.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Connects++
	if pwd, ok := s.accounts[email]; !ok || pwd != password {
		return nil, mailbox.ErrAuth
	}
	return &client{server: s, email: email}, nil
}

type client struct {
	server *Server
	email  string
	folder string
	closed bool
}

func (c *client) Select(folder string) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	c.server.Selects++
	if _, ok := c.server.folders[folder]; !ok {
		return fmt.Errorf("mailbox not found: %s", folder)
	}
	c.folder = folder
	return nil
}

func (c *client) Search(q mailbox.Query) ([]uint32, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	msgs := c.server.folders[c.folder]
	var seqs []uint32
	for i, m := range msgs {
		if !matchQuery(m, q) {
			continue
		}
		seqs = append(seqs, uint32(i+1))
	}
	return seqs, nil
}

func matchQuery(m *Message, q mailbox.Query) bool {
	for _, f := range q.WithFlags {
		if !hasFlag(m.Flags, f) {
			return false
		}
	}
	for _, f := range q.WithoutFlags {
		if hasFlag(m.Flags, f) {
			return false
		}
	}
	if q.Text == "" {
		return true
	}
	text := strings.ToLower(q.Text)
	if bytes.Contains(bytes.ToLower(m.Raw), []byte(text)) {
		return true
	}
	env, err := enmime.ReadEnvelope(bytes.NewReader(m.Raw))
	if err != nil {
		return false
	}
	for _, v := range []string{env.GetHeader("Subject"), env.GetHeader("From"), env.GetHeader("To")} {
		if strings.Contains(strings.ToLower(v), text) {
			return true
		}
	}
	return false
}

func hasFlag(flags []string, target string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, target) {
			return true
		}
	}
	return false
}

func (c *client) Fetch(seqNums []uint32) ([]mailbox.Message, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	msgs := c.server.folders[c.folder]
	out := make([]mailbox.Message, 0, len(seqNums))
	sorted := append([]uint32{}, seqNums...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	for _, seq := range sorted {
		if seq == 0 || int(seq) > len(msgs) {
			continue
		}
		m := msgs[seq-1]
		out = append(out, mailbox.Message{
			SeqNum: seq,
			UID:    m.UID,
			Flags:  append([]string{}, m.Flags...),
			Size:   uint32(len(m.Raw)),
			Raw:    m.Raw,
		})
	}
	return out, nil
}

func (c *client) FetchPreview(seqNums []uint32, maxBytes int64) ([]mailbox.Message, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	msgs := c.server.folders[c.folder]
	out := make([]mailbox.Message, 0, len(seqNums))
	sorted := append([]uint32{}, seqNums...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	limit := int(maxBytes)
	if limit <= 0 {
		limit = 65536
	}

	for _, seq := range sorted {
		if seq == 0 || int(seq) > len(msgs) {
			continue
		}
		m := msgs[seq-1]
		raw := m.Raw
		if len(raw) > limit {
			raw = raw[:limit]
		}
		hasAttach := strings.Contains(strings.ToLower(string(m.Raw)), "content-disposition: attachment")
		out = append(out, mailbox.Message{
			SeqNum:     seq,
			UID:        m.UID,
			Flags:      append([]string{}, m.Flags...),
			Size:       uint32(len(m.Raw)),
			Raw:        raw,
			Attachment: &hasAttach,
		})
	}
	return out, nil
}

func (c *client) FetchUID(uid uint32) (*mailbox.Message, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	for i, m := range c.server.folders[c.folder] {
		if m.UID == uid {
			return &mailbox.Message{
				SeqNum: uint32(i + 1),
				UID:    m.UID,
				Flags:  append([]string{}, m.Flags...),
				Size:   uint32(len(m.Raw)),
				Raw:    m.Raw,
			}, nil
		}
	}
	return nil, mailbox.ErrNotFound
}

func (c *client) FetchStructure(uid uint32) (*mailbox.Part, error) {
	// 内存实现不模拟 BODYSTRUCTURE；service 会回退到整封解析路径。
	return nil, mailbox.ErrUnsupported
}

func (c *client) FetchSections(uid uint32, sections []mailbox.Section) (map[string][]byte, error) {
	return nil, mailbox.ErrUnsupported
}

func (c *client) FetchAll() ([]mailbox.Message, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	msgs := c.server.folders[c.folder]
	out := make([]mailbox.Message, 0, len(msgs))
	for i, m := range msgs {
		out = append(out, mailbox.Message{
			SeqNum: uint32(i + 1),
			UID:    m.UID,
			Flags:  append([]string{}, m.Flags...),
			Size:   uint32(len(m.Raw)),
			Raw:    m.Raw,
		})
	}
	return out, nil
}

func (c *client) UIDSearchByHeader(field, value string) ([]uint32, error) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	var uids []uint32
	for _, m := range c.server.folders[c.folder] {
		if strings.EqualFold(strings.Trim(utils.ExtractHeader(m.Raw, field), "<>"), strings.Trim(value, "<>")) {
			uids = append(uids, m.UID)
		}
	}
	return uids, nil
}

func (c *client) StoreFlags(uids []uint32, add bool, flags []string) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	for _, m := range c.server.folders[c.folder] {
		for _, uid := range uids {
			if m.UID != uid {
				continue
			}
			if add {
				for _, f := range flags {
					if !hasFlag(m.Flags, f) {
						m.Flags = append(m.Flags, f)
					}
				}
			} else {
				kept := m.Flags[:0]
				for _, f := range m.Flags {
					if !strings.EqualFold(f, flags[0]) {
						kept = append(kept, f)
					}
				}
				m.Flags = kept
			}
		}
	}
	return nil
}

func (c *client) Move(uids []uint32, dest string) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	if _, ok := c.server.folders[dest]; !ok {
		return fmt.Errorf("mailbox not found: %s", dest)
	}
	src := c.server.folders[c.folder]
	kept := make([]*Message, 0, len(src))
	for _, m := range src {
		if containsUID(uids, m.UID) {
			c.server.folders[dest] = append(c.server.folders[dest], m)
			continue
		}
		kept = append(kept, m)
	}
	c.server.folders[c.folder] = kept
	return nil
}

func (c *client) DeleteForever(uids []uint32) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	src := c.server.folders[c.folder]
	kept := make([]*Message, 0, len(src))
	for _, m := range src {
		if !containsUID(uids, m.UID) {
			kept = append(kept, m)
		}
	}
	c.server.folders[c.folder] = kept
	return nil
}

func (c *client) Append(folder string, flags []string, date time.Time, raw []byte) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()

	if _, ok := c.server.folders[folder]; !ok {
		return fmt.Errorf("mailbox not found: %s", folder)
	}
	c.server.add(folder, raw, flags, date)
	return nil
}

func (c *client) CreateFolder(folder string) error {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	if _, ok := c.server.folders[folder]; ok {
		return errors.New("mailbox exists")
	}
	c.server.folders[folder] = nil
	return nil
}

func (c *client) Logout() error {
	c.closed = true
	return nil
}

func containsUID(uids []uint32, uid uint32) bool {
	for _, u := range uids {
		if u == uid {
			return true
		}
	}
	return false
}

// Sender 记录通过 SMTP 发送的邮件。
type Sender struct {
	mu      sync.Mutex
	Sent    []SentMessage
	Errors  []error // 依次返回的错误，用完后按 nil 处理
	SendCnt int
}

// SentMessage 是一条发送记录。
type SentMessage struct {
	From     string
	Password string
	To       []string
	Cc       []string
	Raw      []byte
}

// Send 实现 mailbox.Sender。
func (s *Sender) Send(from, password string, to, cc []string, raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.SendCnt++
	if len(s.Errors) > 0 {
		err := s.Errors[0]
		s.Errors = s.Errors[1:]
		if err != nil {
			return err
		}
	}
	s.Sent = append(s.Sent, SentMessage{
		From: from, Password: password,
		To: append([]string{}, to...), Cc: append([]string{}, cc...),
		Raw: append([]byte{}, raw...),
	})
	return nil
}

// Last 返回最近一次发送记录。
func (s *Sender) Last() (SentMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Sent) == 0 {
		return SentMessage{}, false
	}
	return s.Sent[len(s.Sent)-1], true
}

// Directory 是内存账号目录。
type Directory struct {
	mu       sync.Mutex
	Accounts map[string]*directory.Account
	FailWith error
}

// NewDirectory 创建内存账号目录。
func NewDirectory() *Directory {
	return &Directory{Accounts: map[string]*directory.Account{}}
}

// Add 添加账号。
func (d *Directory) Add(email, first, last string, admin int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Accounts[email] = &directory.Account{ID: int64(len(d.Accounts) + 1), Address: email, FirstName: first, LastName: last, AdminLevel: admin}
}

func (d *Directory) Get(email string) (*directory.Account, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.FailWith != nil {
		return nil, d.FailWith
	}
	acc, ok := d.Accounts[email]
	if !ok {
		return nil, fmt.Errorf("当前域名下此账号不存在")
	}
	copy := *acc
	return &copy, nil
}

func (d *Directory) List(domain string) ([]directory.Account, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []directory.Account
	for email, acc := range d.Accounts {
		if strings.HasSuffix(email, "@"+domain) {
			out = append(out, *acc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out, nil
}

func (d *Directory) Create(email, password, firstName, lastName string, admin int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.Accounts[email]; ok {
		return fmt.Errorf("邮箱地址 [%s] 已存在", email)
	}
	d.Accounts[email] = &directory.Account{ID: int64(len(d.Accounts) + 1), Address: email, FirstName: firstName, LastName: lastName, AdminLevel: admin}
	return nil
}

func (d *Directory) Update(email, firstName, lastName string, admin int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	acc, ok := d.Accounts[email]
	if !ok {
		return fmt.Errorf("当前域名下此账号不存在")
	}
	if firstName != "" {
		acc.FirstName = firstName
	}
	if lastName != "" {
		acc.LastName = lastName
	}
	if admin >= 0 {
		acc.AdminLevel = admin
	}
	return nil
}

func (d *Directory) Delete(email string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.Accounts[email]; !ok {
		return fmt.Errorf("当前域名下此账号不存在")
	}
	delete(d.Accounts, email)
	return nil
}

func (d *Directory) SetPassword(email, password string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.Accounts[email]; !ok {
		return fmt.Errorf("当前域名下此账号不存在")
	}
	return nil
}

func (d *Directory) DisplayName(email string) (string, error) {
	acc, err := d.Get(email)
	if err != nil {
		return email, err
	}
	name := strings.TrimSpace(acc.FirstName + acc.LastName)
	if name == "" {
		return email, nil
	}
	return fmt.Sprintf("%s <%s>", name, email), nil
}
