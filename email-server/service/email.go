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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"email-server/config"
	"email-server/model"
	"email-server/utils"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/google/uuid"
	"github.com/jhillyerd/enmime"
)

// formatFilter 将字符串数组转换为 MailFilter 结构体
func formatFilter(filter []string) model.MailFilter {
	mailFilter := model.MailFilter{}
	for _, f := range filter {
		switch f {
		case "unread":
			mailFilter.Unread = true
		case "date_asc":
			mailFilter.DateAsc = true
		case "date_desc":
			mailFilter.DateDesc = true
		case "size_asc":
			mailFilter.SizeAsc = true
		case "size_desc":
			mailFilter.SizeDesc = true
		}
	}
	return mailFilter
}

// sortMailList 根据 MailFilter 对邮件列表进行排序
func sortMailList(list []*model.MailItem, mailFilter model.MailFilter) {
	if len(list) <= 1 {
		return
	}

	// 按时间升序
	if mailFilter.DateAsc {
		sort.Slice(list, func(i, j int) bool {
			return list[i].SendTime.Before(list[j].SendTime)
		})
	} else if mailFilter.DateDesc {
		// 按时间降序
		sort.Slice(list, func(i, j int) bool {
			return list[i].SendTime.After(list[j].SendTime)
		})
	} else if mailFilter.SizeAsc {
		// 按大小升序
		sort.Slice(list, func(i, j int) bool {
			sizeI := utils.Formatize(list[i].Size)
			sizeJ := utils.Formatize(list[j].Size)
			return sizeI < sizeJ
		})
	} else if mailFilter.SizeDesc {
		// 按大小降序
		sort.Slice(list, func(i, j int) bool {
			sizeI := utils.Formatize(list[i].Size)
			sizeJ := utils.Formatize(list[j].Size)
			return sizeI > sizeJ
		})
	}
}

// MailList 获取邮件列表
func MailList(email, pwd, folder string, page, size int64, keyword string, filter []string) ([]*model.MailItem, int64, error) {
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

	var total int64
	var list []*model.MailItem

	// 处理筛选条件
	mailFilter := formatFilter(filter)

	// 搜索邮件
	searchCrit := &imap.SearchCriteria{}
	if keyword != "" {
		searchCrit.Text = []string{keyword}
	}
	// 未读
	if mailFilter.Unread {
		searchCrit.WithoutFlags = []string{"\\Seen"}
	}

	ids, err := imapClient.Search(searchCrit)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	total = int64(len(ids))
	if total == 0 {
		return nil, 0, nil
	}

	// 分页
	startIdx := total - page*size
	endIdx := total - (page-1)*size
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > total {
		endIdx = total
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

		// 处理邮件正文
		showText := ""
		if env.Text != "" {
			showText = strings.TrimSpace(env.Text)
			showText = regexp.MustCompile(`\s+`).ReplaceAllString(showText, "")
			showText = strings.ReplaceAll(showText, "*", "")
		}

		fromMail, formInfo, _ := utils.FormatMailName(env.GetHeader("From"))
		toMail, toInfo, _ := utils.FormatMailName(env.GetHeader("To"))
		ccMail, ccInfo, _ := utils.FormatMailName(env.GetHeader("Cc"))

		inReplyToVal := env.GetHeader("In-Reply-To")
		referencesVal := env.GetHeader("References")

		sendTime, _ := utils.FormatDate(env.GetHeader("Date"))

		// 处理定时发送
		Schedule, _ := utils.FormatDate(env.GetHeader("X-Schedule-Send"))
		now := time.Now()
		Schedule = Schedule.Add(-8 * time.Hour)
		if Schedule.Before(now) {
			Schedule = time.Time{}
		}


		// 标签处理
		var flagMap = make(map[string]struct{})
		for _, flag := range msg.Flags {
			if strings.HasPrefix(flag, "\\") {
				short := flag[1:]
				flagMap[short] = struct{}{}
			} else if strings.HasPrefix(flag, "$") {
				flagMap[flag] = struct{}{}
			}
		}
		var flags []string
		for f := range flagMap {
			flags = append(flags, f)
		}

		// 附件处理
		hasAttach := false
		for _, att := range env.Attachments {
			if att.Disposition == "attachment" {
				hasAttach = true
				break
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
			Schedule:   Schedule,
			Text:       showText,
			HasAttach:  hasAttach,
			Folder:     folder,
			Size:       utils.FormatUnitSize(int64(msg.Size)),
			Flags:      flags,
		}
		list = append(list, item)
	}

	if err := <-done; err != nil {
		return nil, 0, fmt.Errorf("获取邮件失败: %v", err)
	}

	// 排序
	sortMailList(list, mailFilter)

	return list, total, nil
}

// StarMailList 星标邮件
func StarMailList(email, pwd string, page, size int64, keyword string, filter []string) ([]*model.MailItem, int64, error) {
	// 验证用户
	imapClient, err := utils.DialIMAPClient(email, pwd)
	if err != nil {
		return nil, 0, err
	}
	defer imapClient.Logout()

	var total int64
	var list []*model.MailItem

	// 处理筛选条件
	mailFilter := formatFilter(filter)

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
		// 未读
		if mailFilter.Unread {
			searchCrit.WithoutFlags = []string{"\\Seen"}
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

			// 处理邮件正文
			showText := ""
			if env.Text != "" {
				showText = strings.TrimSpace(env.Text)
				showText = regexp.MustCompile(`\s+`).ReplaceAllString(showText, "")
				showText = strings.ReplaceAll(showText, "*", "")
			}

			fromMail, formInfo, _ := utils.FormatMailName(env.GetHeader("From"))
			toMail, toInfo, _ := utils.FormatMailName(env.GetHeader("To"))
			ccMail, ccInfo, _ := utils.FormatMailName(env.GetHeader("Cc"))

			sendTime, _ := utils.FormatDate(env.GetHeader("Date"))

			inReplyToVal := env.GetHeader("In-Reply-To")
			referencesVal := env.GetHeader("References")

			// 标签处理
			var flagMap = make(map[string]struct{})
			for _, flag := range msg.Flags {
				if strings.HasPrefix(flag, "\\") {
					short := flag[1:]
					flagMap[short] = struct{}{}
				}
			}
			if folder != "INBOX" {
				flagMap["Seen"] = struct{}{}
			}
			var flags []string
			for f := range flagMap {
				flags = append(flags, f)
			}

			// 附件处理
			hasAttach := false
			for _, att := range env.Attachments {
				if att.Disposition == "attachment" {
					hasAttach = true
					break
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
				HasAttach:  hasAttach,
				Folder:     folder,
				Size:       utils.FormatUnitSize(int64(msg.Size)),
				Flags:      flags,
			}
			list = append(list, item)
		}

		if err := <-done; err != nil {
			fmt.Printf("获取文件夹 %s 的邮件失败: %v\n", folder, err)
		}
	}

	// 排序
	sortMailList(list, mailFilter)

	return list, total, nil
}

// MailDetail 获取邮件详情
func MailDetail(email, pwd string, token string, folder string, uid int64, host string) (*model.MailDetail, error) {
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

	// 获取邮件正文
	content := env.HTML
	if content == "" {
		content = env.Text
	}

	//	cid内容
	cidMap := make(map[string]string)

	// 构建附件列表
	var attachments []model.AttachmentInfo
	for _, att := range env.Attachments {
		if att.Disposition == "attachment" {
			filetype := strings.Split(att.FileName, ".")[1]
			size := int64(len(att.Content))
			attachments = append(attachments, model.AttachmentInfo{
				PartID:      att.PartID,
				FileName:    att.FileName,
				ContentType: att.ContentType,
				FileType:    strings.ToLower(filetype),
				Size:        utils.FormatUnitSize(size),
			})
		} else {
			contentID := strings.Trim(att.Header.Get("Content-ID"), "<>")
			if contentID != "" && len(att.Content) > 0 {
				// 生成本地文件路径
				ext := filepath.Ext(att.FileName)
				fileName := fmt.Sprintf("%s_%s%s", "image", att.PartID, ext)

				// 保存到静态资源目录
				staticDir := filepath.Join("static", "images", email, folder, fmt.Sprint(uid))
				if err := os.MkdirAll(staticDir, 0755); err != nil {
					fmt.Printf("创建静态目录失败: %v\n", err)
					continue
				}

				localPath := filepath.Join(staticDir, fileName)
				if err := os.WriteFile(localPath, att.Content, 0644); err != nil {
					fmt.Printf("保存内联图片失败: %v\n", err)
					continue
				}

				// 构建 HTTP 访问 URL
				imageURL := fmt.Sprintf("http://%s/api/viewfile?url=static/images/%s/%s/%d/%s",
					host, email, folder, uid, fileName)
				cidMap[contentID] = imageURL
			}
		}
	}

	// 处理内联图片
	if content != "" && len(env.Inlines) > 0 {
		for _, inline := range env.Inlines {
			contentID := strings.Trim(inline.Header.Get("Content-ID"), "<>")
			if contentID != "" && len(inline.Content) > 0 {
				// 生成本地文件路径
				ext := filepath.Ext(inline.FileName)
				fileName := fmt.Sprintf("%s_%s%s", "image", inline.PartID, ext)

				// 保存到静态资源目录
				staticDir := filepath.Join("static", "images", email, folder, fmt.Sprint(uid))
				if err := os.MkdirAll(staticDir, 0755); err != nil {
					fmt.Printf("创建静态目录失败: %v\n", err)
					continue
				}

				localPath := filepath.Join(staticDir, fileName)
				if err := os.WriteFile(localPath, inline.Content, 0644); err != nil {
					fmt.Printf("保存内联图片失败: %v\n", err)
					continue
				}
				// 构建 HTTP 访问 URL
				imageURL := fmt.Sprintf("http://%s/api/viewfile?url=static/images/%s/%s/%d/%s",
					host, email, folder, uid, fileName)
				cidMap[contentID] = imageURL
			}
		}
	}

	// 批量给 cid 图片img标签追加 data-href，不修改原有src
	re := regexp.MustCompile(`(?i)<img([^>]*?)src="cid:([^"]+)"([^>]*?)>`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) != 4 {
			return match
		}
		prefix := sub[1]
		cidVal := sub[2]
		suffix := sub[3]

		targetUrl, ok := cidMap[cidVal]
		if !ok {
			return match
		}
		return fmt.Sprintf(`<img%s src="cid:%s" data-href="%s"%s>`, prefix, cidVal, targetUrl, suffix)
	})

	var totalSize int64
	for _, a := range env.Attachments {
		totalSize += int64(len(a.Content))
	}
	for _, i := range env.Inlines {
		totalSize += int64(len(i.Content))
	}

	detail := &model.MailDetail{
		Content:     content,
		Attachments: attachments,
		AttachSize:  utils.FormatUnitSize(totalSize),
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
	flag := "\\" + status
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
	if err = imapClient.UidStore(uidSet, storeOp, flags, nil); err != nil {
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

// BuildRawEmail 构建原始邮件报文
func BuildRawEmail(email, pwd string, from, to []string, cc []string, subject, body string, files []*multipart.FileHeader, extra model.EmailExtra,
) ([]byte, error) {

	// base64 76字符自动换行
	var encodeBase64LineWrap = func(data []byte) string {
		raw := base64.StdEncoding.EncodeToString(data)
		var sb strings.Builder
		sb.Grow(len(raw) + len(raw)/76*2)
		for start := 0; start < len(raw); start += 76 {
			end := start + 76
			if end > len(raw) {
				end = len(raw)
			}
			sb.WriteString(raw[start:end])
			sb.WriteString("\r\n")
		}
		return sb.String()
	}

	// RFC2047 文件名编码
	var encodeFileName = func(name string) string {
		return mime.QEncoding.Encode("utf-8", name)
	}

	// 返回替换后的 HTML 和 提取出的内联图片列表
	var processInlineBase64Images = func(htmlBody string) (string, []model.MailInline) {
		var images []model.MailInline
		// 匹配 data:image/xxx;base64,YYYY
		re := regexp.MustCompile(`(?i)(src\s*=\s*["'])data:image/([a-zA-Z0-9]+);base64,([^"']+)["']`)

		newHTML := re.ReplaceAllStringFunc(htmlBody, func(match string) string {
			submatches := re.FindStringSubmatch(match)
			if len(submatches) < 4 {
				return match
			}

			prefix := submatches[1]                   // src="
			imgType := strings.ToLower(submatches[2]) // png, jpeg 等
			b64Data := submatches[3]                  // base64 字符串

			// 解码 base64
			decoded, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				// 如果解码失败，保留原始 data URI
				return match
			}

			// 生成唯一 CID
			cid := uuid.NewString()

			// 推断 Content-Type
			contentType := "image/" + imgType
			if imgType == "png" {
				contentType = "image/png"
			}

			images = append(images, model.MailInline{
				CID:         cid,
				ContentType: contentType,
				Content:     decoded,
			})

			// 替换为 cid 引用
			return fmt.Sprintf(`%scid:%s"`, prefix, cid)
		})

		return newHTML, images
	}

	buf := new(bytes.Buffer)

	// 模仿QQ boundary命名规则
	mixedBoundary := fmt.Sprintf("----=_NextPart_%s", strings.ReplaceAll(uuid.NewString(), "-", "_"))

	headers := make(map[string]string)
	headers["MIME-Version"] = "1.0"
	headers["Date"] = time.Now().UTC().Format(time.RFC1123)
	headers["Subject"] = mime.BEncoding.Encode("utf-8", subject)
	// 重点：换行 + Tab + 带引号boundary
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed;\n\tboundary=\"%s\"", mixedBoundary)
	headers["Message-ID"] = fmt.Sprintf("<%s@%s>", uuid.NewString(), strings.Split(email, "@")[1])

	if extra.InReplyTo != "" {
		headers["In-Reply-To"] = extra.InReplyTo
	}
	if extra.References != "" {
		headers["References"] = extra.References
	}
	if extra.XScheduleSend != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", extra.XScheduleSend, time.UTC)
		if err == nil {
			headers["X-Schedule-Send"] = t.Format(time.RFC1123)
		}
	}

	// 发件人
	fromAddr := utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), from[0])
	headers["From"] = fromAddr

	// 收件人
	toAddrs := make([]string, 0, len(to))
	for _, addr := range to {
		toAddrs = append(toAddrs, utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), addr))
	}
	headers["To"] = strings.Join(toAddrs, ", ")

	// 抄送
	if len(cc) > 0 {
		ccAddrs := make([]string, 0, len(cc))
		for _, addr := range cc {
			ccAddrs = append(ccAddrs, utils.FormatMailAddr(config.GetConfig(constant.AdminPassword), addr))
		}
		headers["Cc"] = strings.Join(ccAddrs, ", ")
	}

	// 写入头部
	for k, v := range headers {
		_, _ = fmt.Fprintf(buf, "%s: %s\r\n", k, v)
	}
	_, _ = buf.WriteString("\r\n")
	_, _ = fmt.Fprintf(buf, "This is a multi-part message in MIME format.\r\n")

	// 内容
	altBoundary := fmt.Sprintf("----=_NextPart_%s", strings.ReplaceAll(uuid.NewString(), "-", "_"))
	altHeader := textproto.MIMEHeader{}
	altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative;\n\tboundary=\"%s\"", altBoundary))

	altPart, err := createMimePart(buf, altHeader, mixedBoundary)
	if err != nil {
		return nil, err
	}

	// text/plain
	plainHeader := textproto.MIMEHeader{}
	plainHeader.Set("Content-Type", "text/plain;\n\tcharset=\"utf-8\"")
	plainHeader.Set("Content-Transfer-Encoding", "base64")
	plainPart, err := createMimePart(altPart, plainHeader, altBoundary)
	if err == nil {
		plainText := utils.StripHTML(body)
		_, _ = plainPart.Write([]byte(encodeBase64LineWrap([]byte(plainText))))
	}

	// text/html
	processedHTML, inlineImages := processInlineBase64Images(body)
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html;\n\tcharset=\"utf-8\"")
	htmlHeader.Set("Content-Transfer-Encoding", "base64")
	htmlPart, err := createMimePart(altPart, htmlHeader, altBoundary)
	if err != nil {
		return nil, fmt.Errorf("创建html part失败: %w", err)
	}
	_, _ = htmlPart.Write([]byte(encodeBase64LineWrap([]byte(processedHTML))))

	// 闭合 alternative
	_, _ = fmt.Fprintf(altPart, "\r\n--%s--\r\n", altBoundary)

	// 从 HTML 中提取的内联图片
	for _, img := range inlineImages {
		imgHeader := textproto.MIMEHeader{}
		imgHeader.Set("Content-Type", fmt.Sprintf("%s;\n\tname=\"%s\"", img.ContentType, encodeFileName(img.CID)))
		imgHeader.Set("Content-Transfer-Encoding", "base64")
		imgHeader.Set("Content-ID", fmt.Sprintf("<%s>", img.CID))
		imgHeader.Set("Content-Disposition", fmt.Sprintf("inline;\n\tfilename=\"%s\"", encodeFileName(img.CID)))

		imgPart, err := createMimePart(buf, imgHeader, mixedBoundary)
		if err != nil {
			fmt.Printf("写入HTML提取的内联图片失败 cid=%s err=%v\n", img.CID, err)
			continue
		}
		_, _ = imgPart.Write([]byte(encodeBase64LineWrap(img.Content)))
	}

	// 内联图片 inline
	for _, inline := range extra.OriginInlines {
		imgHeader := textproto.MIMEHeader{}
		imgHeader.Set("Content-Type", fmt.Sprintf("%s;\n\tname=\"%s\"", inline.ContentType, encodeFileName(inline.FileName)))
		imgHeader.Set("Content-Transfer-Encoding", "base64")
		imgHeader.Set("Content-ID", fmt.Sprintf("<%s>", inline.CID))
		imgHeader.Set("Content-Disposition", fmt.Sprintf("inline;\n\tfilename=\"%s\"", encodeFileName(inline.FileName)))

		imgPart, err := createMimePart(buf, imgHeader, mixedBoundary)
		if err != nil {
			fmt.Printf("写入内联图片失败 cid=%s err=%v\n", inline.CID, err)
			continue
		}
		_, _ = imgPart.Write([]byte(encodeBase64LineWrap(inline.Content)))
	}

	// 原邮件附件
	for _, att := range extra.OriginAttaches {
		attachHeader := textproto.MIMEHeader{}
		attachHeader.Set("Content-Type", fmt.Sprintf("%s;\n\tname=\"%s\"", att.ContentType, encodeFileName(att.FileName)))
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment;\n\tfilename=\"%s\"", encodeFileName(att.FileName)))

		attPart, err := createMimePart(buf, attachHeader, mixedBoundary)
		if err != nil {
			fmt.Printf("写入预加载附件失败 filename=%s err=%v\n", att.FileName, err)
			continue
		}
		_, _ = attPart.Write([]byte(encodeBase64LineWrap(att.Content)))
	}

	// 本次新上传附件
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			fmt.Printf("打开上传附件失败: %s err=%v\n", file.Filename, err)
			continue
		}
		content, err := io.ReadAll(src)
		_ = src.Close()
		if err != nil {
			fmt.Printf("读取附件内容失败: %s err=%v\n", file.Filename, err)
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		attachHeader := textproto.MIMEHeader{}
		attachHeader.Set("Content-Type", fmt.Sprintf("%s;\n\tname=\"%s\"", contentType, encodeFileName(file.Filename)))
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment;\n\tfilename=\"%s\"", encodeFileName(file.Filename)))

		attPart, err := createMimePart(buf, attachHeader, mixedBoundary)
		if err != nil {
			fmt.Printf("新建附件part失败: %s err=%v\n", file.Filename, err)
			continue
		}
		_, _ = attPart.Write([]byte(encodeBase64LineWrap(content)))
	}

	_, _ = fmt.Fprintf(buf, "\r\n--%s--\r\n", mixedBoundary)

	return buf.Bytes(), nil
}

// 创建MIME分隔
func createMimePart(w io.Writer, header textproto.MIMEHeader, boundary string) (io.Writer, error) {
	_, err := fmt.Fprintf(w, "\r\n--%s\r\n", boundary)
	if err != nil {
		return nil, err
	}
	for k, vals := range header {
		for _, v := range vals {
			_, err = fmt.Fprintf(w, "%s: %s\r\n", k, v)
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = w.Write([]byte("\r\n"))
	return w, err
}

// ScheduleSendEmail 定时发送邮件
func ScheduleSendEmail(email, pwd string, to []string, cc []string, raw []byte) error {
	// 提取 Message-ID
	messageID := utils.GetExtractHeader(raw, "Message-ID")
	// 提取自定义头部 X-Schedule-Send
	scheduleSend := utils.GetExtractHeader(raw, "X-Schedule-Send")

	// 如果没有定时时间，立即发送
	if scheduleSend == "" {
		return SmtpSendEmail(email, pwd, to, cc, raw)
	}

	targetUTC, err := time.Parse("Mon, 02 Jan 2006 15:04:05 UTC", scheduleSend)
	if err != nil {
		return fmt.Errorf("解析UTC定时时间失败: %w", err)
	}
	// 解析时间
	targetLocal := targetUTC.Add(-8 * time.Hour)
	nowUTC := time.Now().UTC()
	duration := targetLocal.Sub(nowUTC)

	// 如果时间已过，立即发送
	if duration <= 0 {
		return SmtpSendEmail(email, pwd, to, cc, raw)
	}

	// 启动一个独立的协程，定时发送
	go func(msgID string, waitDur time.Duration) {
		fmt.Printf("定时任务已启动，等待 %v 后发送，\nMessage-ID:%s\n", waitDur, msgID)
		time.Sleep(waitDur)

		// 根据messageID查找草稿箱当前邮件UID
		uid, err := utils.GetUid(email, pwd, msgID, config.FolderDrafts)
		if err != nil {
			fmt.Printf("定时发送终止：未找到对应草稿:%v\n", err)
			return
		}
		// 拉取当前最新完整raw
		newRaw, err := utils.GetMailRawByUID(email, pwd, config.FolderDrafts, uid)
		if err != nil {
			fmt.Printf("定时发送终止：读取邮件raw失败:%v\n", err)
			return
		}
		// 校验最新邮件是否还有 X-Schedule-Send 头部
		latestSchedule := utils.GetExtractHeader(newRaw, "X-Schedule-Send")
		if latestSchedule == "" {
			return
		}

		// 校验通过，使用新raw发送邮件
		if err := SmtpSendEmail(email, pwd, to, cc, newRaw); err != nil {
			fmt.Printf("定时发送邮件失败 [%s]: %v\n", latestSchedule, err)
			return
		}

		// 发送成功后，将邮件从草稿箱移动到已发送文件夹
		if msgID != "" {
			if uid, err := utils.GetUid(email, pwd, msgID, config.FolderDrafts); err == nil {
				// 移除Draft标记
				if err = UpdateMailFlag(email, pwd, config.FolderDrafts, uid, 2, "Draft"); err != nil {
					fmt.Printf("标记邮件失败: %v\n", err)
				}
				// 移动邮件
				if err = MoveMail(email, pwd, config.FolderDrafts, config.FolderSent, []int64{uid}); err != nil {
					fmt.Printf("移动邮件失败: %v\n", err)
				} else {
					fmt.Printf("邮件已发送成功，并已移动到已发送文件夹\n")
				}
			}
		} else {
			fmt.Printf("未找到 Message-ID，跳过移动操作\n")
		}
	}(messageID, duration)

	return nil
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

// UnScheduleEmail 取消定时发送的邮件
func UnScheduleEmail(email, pwd string, folder string, uid int64, opType int64, status string) error {
	// 修改状态
	if err := UpdateMailFlag(email, pwd, folder, uid, opType, status); err != nil {
		return fmt.Errorf("更新邮件标志失败: %w", err)
	}

	// 获取邮件原始数据
	raw, err := utils.GetMailRawByUID(email, pwd, folder, uid)
	if err != nil {
		return fmt.Errorf("获取邮件原始数据失败: %w", err)
	}

	// 解析邮件并删除 X-Schedule-Send 头部
	reader := bytes.NewReader(raw)
	mailReader, err := mail.CreateReader(reader)
	if err != nil {
		return fmt.Errorf("创建邮件读取器失败: %w", err)
	}
	defer mailReader.Close()

	// 获取所有头部
	header := mailReader.Header

	// 检查是否有 X-Schedule-Send 头部
	if header.Get("X-Schedule-Send") == "" {
		return nil
	}

	// 重新构建邮件
	var newRaw bytes.Buffer

	// 逐行读取原始邮件
	lines := strings.Split(string(raw), "\r\n")
	inHeaders := true
	for _, line := range lines {
		// 检测头部结束（空行）
		if inHeaders && line == "" {
			inHeaders = false
			newRaw.WriteString("\r\n")
			continue
		}

		// 在头部区域，跳过 X-Schedule-Send 行
		if inHeaders && strings.HasPrefix(line, "X-Schedule-Send:") {
			continue
		}

		newRaw.WriteString(line)
		newRaw.WriteString("\r\n")
	}

	// 删除旧邮件
	if err := DeleteMail(email, pwd, folder, []int64{uid}); err != nil {
		return fmt.Errorf("删除旧邮件失败: %w", err)
	}

	// 保存新邮件（不含 X-Schedule-Send 头部）
	if err := SaveMailToFolder(email, pwd, folder, newRaw.Bytes()); err != nil {
		return fmt.Errorf("保存新邮件失败: %w", err)
	}
	fmt.Printf("定时任务已取消，Message-ID:%s\n", header.Get("Message-ID"))
	return nil
}
