package service

import (
	"bytes"
	"email-server/constant"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"email-server/config"
	"email-server/model"
	"email-server/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/emersion/go-imap"
	"github.com/google/uuid"
	"github.com/jhillyerd/enmime"
)

// MailList 获取邮件列表
var reg = regexp.MustCompile(`\s+`)

func MailList(email, pwd, folder string, page, size int64, keyword string) ([]*model.MailItem, int, error) {
	// 验证用户
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return nil, 0, err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return nil, 0, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	// 搜索邮件
	searchCrit := &imap.SearchCriteria{}
	if keyword != "" {
		searchCrit.Text = []string{keyword}
	}

	ids, err := imapClient.Search(searchCrit)
	if err != nil {
		return nil, 0, err
	}

	total := len(ids)
	if total == 0 {
		return nil, 0, nil
	}

	// 分页
	startIdx := int64(total) - page*size
	endIdx := int64(total) - (page-1)*size
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > int64(total) {
		endIdx = int64(total)
	}
	// 反转索引以获取正确的分页范围
	pageIdx := ids[startIdx:endIdx]

	// 构建序列号
	seqSet := new(imap.SeqSet)
	for _, id := range pageIdx {
		seqSet.AddNum(id)
	}

	// 获取完整原始邮件
	mailMsg := make(chan *imap.Message, len(pageIdx))
	done := make(chan error, 1)
	go func() {
		done <- imapClient.Fetch(seqSet, []imap.FetchItem{
			imap.FetchItem("BODY.PEEK[]"), // 邮件内容,不标记已读
			imap.FetchUid,
			imap.FetchFlags,
			imap.FetchEnvelope,
			imap.FetchRFC822Size,
		}, mailMsg)
	}()

	// 时间戳
	type timeStamp struct {
		timeStamp int64
	}
	var tmpList []timeStamp
	// 获取邮件列表
	var list []*model.MailItem
	for msg := range mailMsg {

		// 获取邮件体
		section := &imap.BodySectionName{}
		r := msg.GetBody(section)
		if r == nil {
			continue
		}

		env, err := enmime.ReadEnvelope(r)
		if err != nil {
			fmt.Printf("解析邮件失败: %v\n", err)
			continue
		}

		// 处理邮件正文（精简显示）
		showText := strings.TrimSpace(env.Text)
		showText = reg.ReplaceAllString(showText, "")
		showText = strings.ReplaceAll(showText, "*", "")

		fromMail, formInfo, _ := utils.GetNameInfo(env.GetHeader("From"))
		toMail, toInfo, _ := utils.GetNameInfo(env.GetHeader("To"))
		ccMail, ccInfo, _ := utils.GetNameInfo(env.GetHeader("Cc"))

		sendTime, sendTimeUnix := utils.ParseMailTime(env.GetHeader("Date"))

		inReplyToVal := env.GetHeader("In-Reply-To")
		referencesVal := env.GetHeader("References")

		var flags []string
		for _, flag := range msg.Flags {
			// 以 \ 开头的系统标记，截取后面文本
			if strings.HasPrefix(flag, "\\") {
				flags = append(flags, flag[1:])
			}
		}

		item := &model.MailItem{
			Uid:        int64(msg.Uid),
			MessageId:  env.GetHeader("Message-Id"),
			ReplyTo:    &inReplyToVal,
			References: &referencesVal,
			From:       fromMail,
			FromInfo:   formInfo[0],
			To:         toMail,
			ToInfo:     toInfo,
			Cc:         ccMail,
			CcInfo:     ccInfo,
			Subject:    env.GetHeader("Subject"),
			SendTime:   sendTime,
			Text:       showText,
			HasAttach:  len(env.Attachments) > 0,
			Folder:     folder,
			Size:       utils.FormatFileSize(msg.Size),
			Flags:      flags,
		}
		tmpList = append(tmpList, timeStamp{
			timeStamp: sendTimeUnix,
		})
		list = append(list, item)
	}

	if err := <-done; err != nil {
		return nil, 0, fmt.Errorf("获取邮件失败: %v", err)
	}

	// 按照时间倒序
	sort.Slice(list, func(i, j int) bool {
		return tmpList[i].timeStamp > tmpList[j].timeStamp
	})

	return list, total, nil
}

// MailDetail 获取邮件详情
func MailDetail(email, pwd string, folder string, uid int64) (*model.MailDetail, error) {
	// 验证用户
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return nil, err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	uidSet := new(imap.SeqSet)
	uidSet.AddNum(uint32(uid))

	bodyMail := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.UidFetch(uidSet, []imap.FetchItem{
			imap.FetchRFC822,
			imap.FetchUid,
			imap.FetchRFC822Size,
		}, bodyMail)
	}()

	// 从channel中获取邮件消息
	msg, ok := <-bodyMail
	if !ok {
		if err := <-done; err != nil {
			return nil, fmt.Errorf("获取邮件失败: %w", err)
		}
		return nil, fmt.Errorf("未找到邮件 UID: %d", uid)
	}

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		return nil, fmt.Errorf("无法获取邮件内容")
	}

	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	// 构建附件列表
	var attachments []model.AttachmentInfo
	for _, att := range env.Attachments {

		filetype := strings.Split(att.FileName, ".")[1]

		size := uint32(len(att.Content))

		attachments = append(attachments, model.AttachmentInfo{
			PartID:      att.PartID,
			FileName:    att.FileName,
			ContentType: att.ContentType,
			FileType:    strings.ToLower(filetype),
			Size:        utils.FormatFileSize(size),
		})
	}

	// 获取邮件正文
	content := env.HTML
	if content == "" {
		content = env.Text
	}

	// 处理内联图片：将 cid:xxx 替换为图片接口地址
	if content != "" && len(env.Inlines) > 0 {
		// 构建 CID 到 PartID 的映射
		cidMap := make(map[string]string)
		for _, inline := range env.Inlines {
			// Content-ID 通常格式为 <xxx>，去掉尖括号
			contentID := strings.Trim(inline.Header.Get("Content-Id"), "<>")
			if contentID != "" {
				cidMap[contentID] = inline.PartID
			}
		}

		// 批量替换 HTML 中的 cid: 引用
		for cid, partID := range cidMap {
			// 对 partID 进行 URL 编码
			encodedPartID := url.QueryEscape(partID)
			// 构建图片接口地址
			imageUrl := fmt.Sprintf("/api/mail/inline?folder=%s&uid=%d&part_id=%s", folder, uid, encodedPartID)
			// 替换所有 cid:xxx 引用
			content = strings.ReplaceAll(content, "cid:"+cid, imageUrl)
		}
	}

	var totalSize uint32
	for _, a := range env.Attachments {
		totalSize += uint32(len(a.Content))
	}
	for _, i := range env.Inlines {
		totalSize += uint32(len(i.Content))
	}

	detail := &model.MailDetail{
		Content:     content,
		Attachments: attachments,
		AttachSize:  utils.FormatFileSize(totalSize),
	}

	return detail, nil
}

// UpdateMailFlag 更新邮件状态
func UpdateMailFlag(email, pwd string, folder string, uid int64, opType int64, status string) error {
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	uidSet := new(imap.SeqSet)
	uidSet.AddNum(uint32(uid))

	// 验证状态参数
	validStatuses := []string{
		imap.SeenFlag,     // "\\Seen" - 已读
		imap.AnsweredFlag, // "\\Answered" - 已回复
		imap.FlaggedFlag,  // "\\Flagged" - 星标
		imap.DeletedFlag,  // "\\Deleted" - 删除
		imap.DraftFlag,    // "\\Draft" - 草稿
	}

	// 验证状态参数
	var flag = "\\" + status
	isValid := false
	for _, vs := range validStatuses {
		if flag == vs {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("无效的状态标志: %s，必须是以下之一: %v", flag, validStatuses)
	}

	// 映射操作类型
	var storeOp imap.StoreItem
	switch opType {
	case 1:
		storeOp = imap.AddFlags // 添加标记
	case 2:
		storeOp = imap.RemoveFlags // 删除指定标记
	default:
		return fmt.Errorf("操作类型仅支持 1(添加)、2(删除)，传入值：%d", opType)
	}

	flags := []interface{}{flag}
	err = imapClient.UidStore(uidSet, storeOp, flags, nil)
	if err != nil {
		return fmt.Errorf("更新邮件状态失败: %w", err)
	}

	return nil
}

// DownloadAttachment 下载附件
func DownloadAttachment(email, pwd string, folder string, uid int64, partID string) (string, []byte, error) {
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return "", nil, err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return "", nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	uidSet := new(imap.SeqSet)
	uidSet.AddNum(uint32(uid))

	bodyMail := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.UidFetch(uidSet, []imap.FetchItem{
			imap.FetchRFC822,
			imap.FetchUid,
		}, bodyMail)
	}()

	// 从channel中获取邮件消息
	msg, ok := <-bodyMail
	if !ok {
		if err := <-done; err != nil {
			return "", nil, fmt.Errorf("获取邮件失败: %w", err)
		}
		return "", nil, fmt.Errorf("未找到附件 PartID: %s", partID)
	}

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		return "", nil, fmt.Errorf("无法获取邮件内容")
	}
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return "", nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	var fileName string
	var fileData []byte
	for _, att := range env.Attachments {
		if att.PartID == partID {
			fmt.Println("Found attachment:", att.FileName)
			fileName = att.FileName
			fileData = att.Content
		}
	}

	return fileName, fileData, nil
}

// InlineImage 下载内联图片
func InlineImage(email, pwd string, folder string, uid int64, partID string) (string, []byte, error) {
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return "", nil, err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return "", nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	uidSet := new(imap.SeqSet)
	uidSet.AddNum(uint32(uid))

	bodyMail := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.UidFetch(uidSet, []imap.FetchItem{
			imap.FetchRFC822,
			imap.FetchUid,
		}, bodyMail)
	}()

	// 从channel中获取邮件消息
	msg, ok := <-bodyMail
	if !ok {
		if err := <-done; err != nil {
			return "", nil, fmt.Errorf("获取邮件失败: %w", err)
		}
		return "", nil, fmt.Errorf("未找到内联图片 PartID: %s", partID)
	}

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		return "", nil, fmt.Errorf("无法获取邮件内容")
	}
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return "", nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	var fileName string
	var fileData []byte
	var contentType string
	for _, inline := range env.Inlines {
		if inline.PartID == partID {
			fileName = inline.FileName
			fileData = inline.Content
			contentType = inline.ContentType
			break
		}
	}

	if fileData == nil {
		return "", nil, fmt.Errorf("未找到内联图片 PartID: %s", partID)
	}

	// 如果没有文件名，使用默认名称
	if fileName == "" {
		fileName = "image.png"
	}

	// 如果没有 Content-Type，根据文件扩展名推断
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fileName))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	return fileName, fileData, nil
}

// MoveMail 移动邮件
func MoveMail(email, pwd string, folder string, toFolder string, uids []int64) error {
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	// 构建UID集合
	uidSet := new(imap.SeqSet)
	for _, uid := range uids {
		uidSet.AddNum(uint32(uid))
	}

	// 使用UidMove移动邮件（复制+删除）操作
	err = imapClient.UidMove(uidSet, toFolder)
	if err != nil {
		return fmt.Errorf("移动邮件失败: %w", err)
	}

	return nil
}

// DeleteMail 删除邮件
func DeleteMail(email, pwd string, folder string, uids []int64) error {
	// 建立IMAP连接
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	// 构建UID集合
	uidSet := new(imap.SeqSet)
	for _, uid := range uids {
		uidSet.AddNum(uint32(uid))
	}

	// 标记为删除
	flags := []interface{}{imap.DeletedFlag}
	err = imapClient.UidStore(uidSet, imap.AddFlags, flags, nil)
	if err != nil {
		return fmt.Errorf("标记邮件为删除状态失败: %w", err)
	}

	// 永久删除已标记的邮件
	if err := imapClient.Expunge(nil); err != nil {
		return fmt.Errorf("执行永久删除失败: %w", err)
	}

	return nil
}

// BuildRawEmail 构建原始邮件内容
func BuildRawEmail(email, pwd, folder string, uid int64, partIDs string, from, to []string, cc []string, subject, body string, files []*multipart.FileHeader, inReplyTo, references string) ([]byte, error) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	defer writer.Close()

	// 设置邮件分隔符
	boundary := writer.Boundary()
	_ = writer.SetBoundary(boundary)

	//  写入邮件头
	headers := make(map[string]string)
	headers["MIME-Version"] = "1.0"
	headers["Date"] = time.Now().UTC().Format(time.RFC1123)
	headers["Subject"] = mime.BEncoding.Encode("utf-8", subject) // 中文不乱码
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=%s", boundary)
	headers["Message-ID"] = fmt.Sprintf("<%s@%s>", uuid.NewString(), strings.Split(email, "@")[1])

	// in-reply-to
	if inReplyTo != "" {
		headers["In-Reply-To"] = inReplyTo
	}
	// references
	if references != "" {
		headers["References"] = references
	}

	// 发件人
	headers["From"] = utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), from[0])
	// 收件人
	var toAddrs []string
	for _, email := range to {
		name := utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), email)

		toAddrs = append(toAddrs, name)
	}
	headers["To"] = strings.Join(toAddrs, ", ")

	// 抄送人
	if len(cc) > 0 {
		var ccAddrs []string
		for _, email := range cc {
			name := utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), email)
			fmt.Printf("ccName:%s\n", name)
			ccAddrs = append(ccAddrs, name)
		}
		headers["Cc"] = strings.Join(ccAddrs, ", ")
	}

	// 写入头
	for k, v := range headers {
		_, _ = fmt.Fprintf(buf, "%s: %s\r\n", k, v)
	}
	// 头结束
	_, _ = buf.WriteString("\r\n")

	// 写入正文（HTML）
	textBodyHeader := textproto.MIMEHeader{}
	textBodyHeader.Set("Content-Type", "text/html; charset=utf-8")
	textBodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")

	part, err := writer.CreatePart(textBodyHeader)
	if err != nil {
		return nil, err
	}
	// 正文编码
	_, _ = part.Write([]byte(body))

	// 旧附件（根据 partIDs 保留）
	if uid > 0 && partIDs != "" && folder != "" {
		// 解析需要保留的 partIDs
		keepIDs := make(map[string]bool)
		for _, idStr := range strings.Split(partIDs, ",") {
			idStr = strings.TrimSpace(idStr)
			if idStr != "" {
				keepIDs[idStr] = true
			}
		}

		// 获取旧邮件详情
		imapClient, err := utils.DialIMAPClient(email, pwd)
		if err == nil {
			defer imapClient.Logout()

			_, err = imapClient.Select(folder, false)
			if err == nil {
				uidSet := new(imap.SeqSet)
				uidSet.AddNum(uint32(uid))

				bodyMail := make(chan *imap.Message, 1)
				done := make(chan error, 1)
				go func() {
					done <- imapClient.UidFetch(uidSet, []imap.FetchItem{
						imap.FetchRFC822,
						imap.FetchUid,
					}, bodyMail)
				}()

				msg, ok := <-bodyMail
				if ok {
					section := &imap.BodySectionName{}
					r := msg.GetBody(section)
					if r != nil {
						env, err := enmime.ReadEnvelope(r)
						if err == nil {
							// 写入需要保留的旧附件
							for _, att := range env.Attachments {
								if keepIDs[att.PartID] {
									fileName := mime.QEncoding.Encode("utf-8", att.FileName)

									attachHeader := textproto.MIMEHeader{}
									attachHeader.Set("Content-Type", att.ContentType)
									attachHeader.Set("Content-Transfer-Encoding", "base64")
									attachHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))

									part, err := writer.CreatePart(attachHeader)
									if err == nil {
										// Base64 编码
										encodedContent := base64.StdEncoding.EncodeToString(att.Content)
										var bufLines bytes.Buffer
										for i := 0; i < len(encodedContent); i += 76 {
											end := i + 76
											if end > len(encodedContent) {
												end = len(encodedContent)
											}
											bufLines.WriteString(encodedContent[i:end])
											bufLines.WriteString("\r\n")
										}
										_, _ = part.Write(bufLines.Bytes())
									}
								}
							}
						}
					}
				}
				<-done
			}
		}
	}

	// 写入新附件
	for _, file := range files {
		fileName := mime.QEncoding.Encode("utf-8", file.Filename)

		// 读取附件内容
		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			return nil, err
		}

		contentType := mime.TypeByExtension(strings.ToLower(file.Filename))
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(file.Filename))
		}
		// 如果依然无法识别，默认使用 octet-stream
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		// 附件头
		attachHeader := textproto.MIMEHeader{}
		attachHeader.Set("Content-Type", contentType)
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))

		// 写入附件
		part, err := writer.CreatePart(attachHeader)
		if err != nil {
			return nil, err
		}

		//  Base64 编码
		encodedContent := base64.StdEncoding.EncodeToString(content)
		// SMTP 要求每行不超过 998 字符，MIME 建议 76 字符
		var bufLines bytes.Buffer
		for i := 0; i < len(encodedContent); i += 76 {
			end := i + 76
			if end > len(encodedContent) {
				end = len(encodedContent)
			}
			bufLines.WriteString(encodedContent[i:end])
			bufLines.WriteString("\r\n")
		}

		// 写入分行后的内容
		_, _ = part.Write(bufLines.Bytes())
	}

	return buf.Bytes(), nil
}

// SmtpSendEmail 发送邮件
func SmtpSendEmail(email, pwd string, to []string, cc []string, raw []byte) error {
	smtpClient, err := utils.DialSMTPClient(email, pwd)
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %w", err)
	}
	defer smtpClient.Close()

	// 发件人
	if err := smtpClient.Mail(email); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	// 收件人
	for _, e := range to {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if err := smtpClient.Rcpt(e); err != nil {
			return fmt.Errorf("设置收件人 %s 失败: %w", e, err)
		}
	}
	// 抄送
	for _, e := range cc {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if err := smtpClient.Rcpt(e); err != nil {
			return fmt.Errorf("设置抄送 %s 失败: %w", e, err)
		}
	}

	// 邮件数据
	s, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("获取邮件数据失败: %w", err)
	}
	_, err = s.Write(raw)
	if err != nil {
		return fmt.Errorf("写入邮件数据失败: %w", err)
	}
	err = s.Close()
	if err != nil {
		return fmt.Errorf("关闭邮件数据失败: %w", err)
	}

	return nil
}

// SaveMailToFolder 保存邮件到指定文件夹
func SaveMailToFolder(email, pwd, folder string, raw []byte) error {
	// 建立IMAP连接
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	// 根据文件夹类型设置邮件标志
	flag := []string{imap.SeenFlag}
	if folder == "Drafts" {
		flag = []string{imap.DraftFlag}
	}

	// 追加邮件到文件夹
	rawMail := bytes.NewReader(raw)
	err = imapClient.Append(folder, flag, time.Now(), rawMail)
	if err != nil {
		return fmt.Errorf("保存邮件到文件夹 %s 失败: %w", folder, err)
	}

	return nil
}

// UpdateDraft 更新草稿邮件
func UpdateDraft(email, pwd, folder string, raw []byte, uid int64) error {
	// 删除旧草稿
	if err := DeleteMail(email, pwd, folder, []int64{uid}); err != nil {
		return fmt.Errorf("删除旧草稿失败: %w", err)
	}

	// 保存新草稿
	if err := SaveMailToFolder(email, pwd, folder, raw); err != nil {
		return fmt.Errorf("保存新草稿失败: %w", err)
	}

	return nil
}

// StarMailList 星标邮件
func StarMailList(email, pwd string, keyword string) ([]*model.MailItem, int64, error) {
	// 验证用户
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return nil, 0, err
	}
	defer imapClient.Logout()

	// 时间戳
	type timeStamp struct {
		timeStamp int64
	}
	var tmpList []timeStamp

	var total int64
	var list []*model.MailItem

	// 遍历所有文件夹config.DefaultFolders
	for _, folder := range config.DefaultFolders {
		// 选择文件夹
		_, err = imapClient.Select(folder, false)
		if err != nil {
			fmt.Printf("选择文件夹 %s 失败: %v\n", folder, err)
			continue
		}

		// 搜索星标邮件
		searchCrit := &imap.SearchCriteria{}
		searchCrit.WithFlags = []string{imap.FlaggedFlag}
		if keyword != "" {
			searchCrit.Text = []string{keyword}
		}

		ids, err := imapClient.Search(searchCrit)
		if err != nil {
			fmt.Printf("在文件夹 %s 中搜索星标邮件失败: %v\n", folder, err)
			continue
		}

		if len(ids) == 0 {
			continue
		}

		total += int64(len(ids))

		// 构建序列号
		seqSet := new(imap.SeqSet)
		for _, id := range ids {
			seqSet.AddNum(id)
		}

		// 获取完整原始邮件
		mailMsg := make(chan *imap.Message, len(ids))
		done := make(chan error, 1)
		go func() {
			done <- imapClient.Fetch(seqSet, []imap.FetchItem{
				imap.FetchItem("BODY.PEEK[]"), // 邮件内容,不标记已读
				imap.FetchUid,
				imap.FetchFlags,
				imap.FetchEnvelope,
				imap.FetchRFC822Size,
			}, mailMsg)
		}()

		// 获取邮件列表
		for msg := range mailMsg {
			// 获取邮件体
			section := &imap.BodySectionName{}
			r := msg.GetBody(section)
			if r == nil {
				continue
			}

			env, err := enmime.ReadEnvelope(r)
			if err != nil {
				fmt.Printf("解析邮件失败: %v\n", err)
				continue
			}

			// 处理邮件正文（精简显示）
			showText := strings.TrimSpace(env.Text)
			showText = reg.ReplaceAllString(showText, "")
			showText = strings.ReplaceAll(showText, "*", "")

			fromMail, formInfo, _ := utils.GetNameInfo(env.GetHeader("From"))
			toMail, toInfo, _ := utils.GetNameInfo(env.GetHeader("To"))
			ccMail, ccInfo, _ := utils.GetNameInfo(env.GetHeader("Cc"))

			sendTime, sendTimeUnix := utils.ParseMailTime(env.GetHeader("Date"))

			inReplyToVal := env.GetHeader("In-Reply-To")
			referencesVal := env.GetHeader("References")

			var flags []string
			for _, flag := range msg.Flags {
				// 以 \ 开头的系统标记，截取后面文本
				if strings.HasPrefix(flag, "\\") {
					flags = append(flags, flag[1:])
				}
			}

			item := &model.MailItem{
				Uid:        int64(msg.Uid),
				MessageId:  env.GetHeader("Message-Id"),
				ReplyTo:    &inReplyToVal,
				References: &referencesVal,
				From:       fromMail,
				FromInfo:   formInfo[0],
				To:         toMail,
				ToInfo:     toInfo,
				Cc:         ccMail,
				CcInfo:     ccInfo,
				Subject:    env.GetHeader("Subject"),
				SendTime:   sendTime,
				Text:       showText,
				HasAttach:  len(env.Attachments) > 0,
				Folder:     folder,
				Size:       utils.FormatFileSize(msg.Size),
				Flags:      flags,
			}
			tmpList = append(tmpList, timeStamp{
				timeStamp: sendTimeUnix,
			})
			list = append(list, item)
		}

		if err := <-done; err != nil {
			fmt.Printf("获取文件夹 %s 的邮件失败: %v\n", folder, err)
		}
	}

	// 按照时间倒序
	sort.Slice(list, func(i, j int) bool {
		spew.Dump(tmpList[i], tmpList[j])
		return tmpList[i].timeStamp > tmpList[j].timeStamp
	})

	return list, total, nil
}
