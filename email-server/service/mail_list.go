package service

import (
	"bytes"
	"fmt"
	"mime"
	"net/mail"
	"sort"
	"strings"
	"time"

	"email-server/config"
	"email-server/mailbox"
	"email-server/model"
	"email-server/utils"

	"github.com/emersion/go-imap"
	"github.com/jhillyerd/enmime"
)

// itemOptions 控制列表项的解析细节，用于保持不同列表接口的历史行为。
type itemOptions struct {
	folder     string
	unescape   bool // 是否反转义 &lt; &gt;（INBOX 列表沿用旧行为）
	forceSeen  bool // 非 INBOX 星标邮件强制视为已读（沿用旧行为）
	keepDollar bool // 是否保留 $ 开头的自定义标记
}

// previewFetchBytes 是列表抓取单封邮件时保留的字节数。
// 列表只需要头部与正文开头，抓取整封会为展示几行预览而传输数 MB 附件。
const previewFetchBytes = 64 << 10

// MailList 返回指定文件夹的分页邮件列表。
//
// 分页语义与旧实现一致：按序列号升序取"从末尾开始的第 page 页"，
// 即 page=1 为最新一页。
func (s *Service) MailList(email, pwd, folder string, page, size int64, keyword string, filter []string) ([]*model.MailItem, int64, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, 0, err
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return nil, 0, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	f := utils.FormatFilter(filter)
	query := mailbox.Query{Text: keyword}
	if f.Unread {
		query.WithoutFlags = []string{imap.SeenFlag}
	}

	ids, err := client.Search(query)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(ids))
	if total == 0 {
		return nil, 0, nil
	}

	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := total - page*size
	end := total - (page-1)*size
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start >= end {
		return nil, total, nil
	}

	msgs, err := client.FetchPreview(ids[start:end], previewFetchBytes)
	if err != nil {
		return nil, 0, fmt.Errorf("获取邮件失败: %v", err)
	}

	opts := itemOptions{folder: folder, unescape: true, keepDollar: true}
	list := make([]*model.MailItem, 0, len(msgs))
	for _, msg := range msgs {
		if item := parseMailItem(msg, opts); item != nil {
			list = append(list, item)
		}
	}

	utils.SortMailList(list, f)
	return list, total, nil
}

// StarMailList 汇总所有标准文件夹中的星标邮件。
func (s *Service) StarMailList(email, pwd, keyword string, filter []string) ([]*model.MailItem, int64, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, 0, err
	}
	defer client.Logout()

	f := utils.FormatFilter(filter)
	var (
		list  []*model.MailItem
		total int64
	)

	for _, folder := range config.DefaultFolders {
		if err := client.Select(folder); err != nil {
			fmt.Printf("选择文件夹 %s 失败: %v\n", folder, err)
			continue
		}

		query := mailbox.Query{Text: keyword, WithFlags: []string{imap.FlaggedFlag}}
		if f.Unread {
			query.WithoutFlags = []string{imap.SeenFlag}
		}

		ids, err := client.Search(query)
		if err != nil {
			fmt.Printf("在文件夹 %s 中搜索星标邮件失败: %v\n", folder, err)
			continue
		}
		if len(ids) == 0 {
			continue
		}
		total += int64(len(ids))

		msgs, err := client.FetchPreview(ids, previewFetchBytes)
		if err != nil {
			fmt.Printf("获取文件夹 %s 的邮件失败: %v\n", folder, err)
			continue
		}

		opts := itemOptions{folder: folder, unescape: false, forceSeen: folder != config.FolderInbox}
		for _, msg := range msgs {
			if item := parseMailItem(msg, opts); item != nil {
				list = append(list, item)
			}
		}
	}

	utils.SortMailList(list, f)
	return list, total, nil
}

// UnreadMailTotal 统计各文件夹未读数以及星标未读数。
func (s *Service) UnreadMailTotal(email, pwd string) (map[string]int64, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	total := map[string]int64{"Star": 0}
	unread := mailbox.Query{WithoutFlags: []string{imap.SeenFlag}}
	starUnread := mailbox.Query{
		WithFlags:    []string{imap.FlaggedFlag},
		WithoutFlags: []string{imap.SeenFlag},
	}

	for _, folder := range config.DefaultFolders {
		if err := client.Select(folder); err != nil {
			fmt.Printf("选择文件夹 %s 失败: %v\n", folder, err)
			continue
		}

		ids, err := client.Search(unread)
		if err != nil {
			return total, err
		}
		total[folder] = int64(len(ids))

		starIDs, err := client.Search(starUnread)
		if err != nil {
			return total, err
		}
		total["Star"] += int64(len(starIDs))
	}

	return total, nil
}

// parseMailItem 把原始邮件解析为列表项；解析失败返回 nil（与旧实现跳过该封的行为一致）。
func parseMailItem(msg mailbox.Message, opts itemOptions) *model.MailItem {
	if len(msg.Raw) == 0 {
		return nil
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(msg.Raw))
	if err != nil {
		// 列表只抓取报文前段，极端 MIME 结构可能无法完整解析；
		// 此时退回只解析头部，保证列表不丢邮件、元信息不缺失。
		fmt.Printf("解析邮件失败，改用头部信息: %v\n", err)
		return parseMailItemHeader(msg, opts)
	}

	fromMail, fromInfo, _ := utils.FormatMailName(env.GetHeader("From"))
	toMail, toInfo, _ := utils.FormatMailName(env.GetHeader("To"))
	ccMail, ccInfo, _ := utils.FormatMailName(env.GetHeader("Cc"))

	inReplyTo := env.GetHeader("In-Reply-To")
	references := env.GetHeader("References")
	sendTime, _ := utils.ParseMailDate(env.GetHeader("Date"))

	// 定时时间：头部保存为 UTC；解析后统一成本地时区，已过期则置零。
	schedule, _ := utils.ParseMailDate(env.GetHeader("X-Schedule-Send"))
	if !schedule.IsZero() && schedule.Before(time.Now()) {
		schedule = time.Time{}
	}

	flags := parseFlags(msg.Flags, opts)

	var fromInfoItem *model.MailInfo
	if len(fromInfo) > 0 {
		fromInfoItem = fromInfo[0]
	}

	hasAttach := hasAttachment(env) || hasAttachmentHeader(msg.Raw)
	if msg.Attachment != nil {
		hasAttach = *msg.Attachment
	}

	return &model.MailItem{
		Uid:        int64(msg.UID),
		MessageId:  env.GetHeader("Message-Id"),
		ReplyTo:    &inReplyTo,
		References: &references,
		From:       fromMail,
		FromInfo:   fromInfoItem,
		To:         toMail,
		ToInfo:     toInfo,
		Cc:         ccMail,
		CcInfo:     ccInfo,
		Subject:    env.GetHeader("Subject"),
		SendTime:   sendTime,
		Schedule:   schedule,
		Text:       utils.PreviewText(env.Text, opts.unescape),
		HasAttach:  hasAttach,
		Folder:     opts.folder,
		Size:       utils.FormatUnitSize(int64(msg.Size)),
		Flags:      flags,
	}
}

// parseMailItemHeader 只解析邮件头，用于报文被抓断时的兜底。
// 正文预览置空，其余元信息（发件人、主题、时间、大小、标记）保持不变。
func parseMailItemHeader(msg mailbox.Message, opts itemOptions) *model.MailItem {
	r, err := mail.ReadMessage(bytes.NewReader(msg.Raw))
	if err != nil {
		return nil
	}
	dec := new(mime.WordDecoder)
	subject, _ := dec.DecodeHeader(r.Header.Get("Subject"))

	fromMail, fromInfo, _ := utils.FormatMailName(r.Header.Get("From"))
	toMail, toInfo, _ := utils.FormatMailName(r.Header.Get("To"))
	ccMail, ccInfo, _ := utils.FormatMailName(r.Header.Get("Cc"))

	inReplyTo := r.Header.Get("In-Reply-To")
	references := r.Header.Get("References")
	sendTime, _ := utils.ParseMailDate(r.Header.Get("Date"))

	schedule, _ := utils.ParseMailDate(r.Header.Get("X-Schedule-Send"))
	if !schedule.IsZero() && schedule.Before(time.Now()) {
		schedule = time.Time{}
	}

	flags := parseFlags(msg.Flags, opts)
	var fromInfoItem *model.MailInfo
	if len(fromInfo) > 0 {
		fromInfoItem = fromInfo[0]
	}
	hasAttach := hasAttachmentHeader(msg.Raw)
	if msg.Attachment != nil {
		hasAttach = *msg.Attachment
	}

	return &model.MailItem{
		Uid:        int64(msg.UID),
		MessageId:  r.Header.Get("Message-Id"),
		ReplyTo:    &inReplyTo,
		References: &references,
		From:       fromMail,
		FromInfo:   fromInfoItem,
		To:         toMail,
		ToInfo:     toInfo,
		Cc:         ccMail,
		CcInfo:     ccInfo,
		Subject:    subject,
		SendTime:   sendTime,
		Schedule:   schedule,
		Text:       "",
		HasAttach:  hasAttach,
		Folder:     opts.folder,
		Size:       utils.FormatUnitSize(int64(msg.Size)),
		Flags:      flags,
	}
}

// hasAttachmentHeader 用头部/分段内容兜底判断是否存在附件。
// 报文被截断时 enmime 可能看不到完整的附件部件，用关键头部做补充判断。
func hasAttachmentHeader(raw []byte) bool {
	head := raw
	if len(head) > previewFetchBytes {
		head = head[:previewFetchBytes]
	}
	lower := strings.ToLower(string(head))
	return strings.Contains(lower, "content-disposition: attachment") ||
		strings.Contains(lower, "content-disposition:attachment")
}

// parseFlags 把 IMAP 标记转换为前端使用的短名称。
func parseFlags(raw []string, opts itemOptions) []string {
	flagSet := make(map[string]struct{}, len(raw))
	for _, flag := range raw {
		switch {
		case strings.HasPrefix(flag, "\\"):
			flagSet[flag[1:]] = struct{}{}
		case opts.keepDollar && strings.HasPrefix(flag, "$"):
			flagSet[flag] = struct{}{}
		}
	}
	if opts.forceSeen {
		flagSet["Seen"] = struct{}{}
	}

	if len(flagSet) == 0 {
		return nil
	}
	flags := make([]string, 0, len(flagSet))
	for f := range flagSet {
		flags = append(flags, f)
	}
	// 标记来自 map，顺序随机；排序保证同一封邮件多次返回的 flags 稳定。
	sort.Strings(flags)
	return flags
}

func hasAttachment(env *enmime.Envelope) bool {
	for _, att := range env.Attachments {
		if att.Disposition == "attachment" {
			return true
		}
	}
	return false
}
