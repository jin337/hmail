package service

import (
	"bytes"
	"email-server/apperr"
	"fmt"
	"strings"
	"time"

	"email-server/contacts"
	"email-server/model"
	"email-server/utils"

	"github.com/jhillyerd/enmime"
)

// MailResources 读取原邮件中的内联图片与指定附件，供"编辑草稿 / 回复 / 转发"复用。
func (s *Service) MailResources(email, pwd, folder string, uid int64, partIDs string) ([]model.MailInline, []model.MailOriginAttach, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, nil, fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return nil, nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	msg, err := client.FetchUID(uint32(uid))
	if err != nil {
		return nil, nil, fmt.Errorf("未找到邮件 UID: %d", uid)
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(msg.Raw))
	if err != nil {
		return nil, nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	var inlines []model.MailInline
	for _, i := range env.Inlines {
		inlines = append(inlines, model.MailInline{
			CID:         i.ContentID,
			Content:     i.Content,
			ContentType: i.ContentType,
			FileName:    i.FileName,
		})
	}

	keep := map[string]bool{}
	for _, id := range strings.Split(partIDs, ",") {
		if id = strings.TrimSpace(id); id != "" {
			keep[id] = true
		}
	}

	var attaches []model.MailOriginAttach
	for _, a := range env.Attachments {
		if len(keep) > 0 && keep[a.PartID] {
			if a.Disposition == "attachment" {
				attaches = append(attaches, model.MailOriginAttach{
					Content:     a.Content,
					ContentType: a.ContentType,
					FileName:    a.FileName,
				})
			} else {
				contentID := strings.Trim(a.Header.Get("Content-ID"), "<>")
				if contentID != "" && len(a.Content) > 0 {
					inlines = append(inlines, model.MailInline{
						CID:         a.ContentID,
						Content:     a.Content,
						ContentType: a.ContentType,
						FileName:    a.FileName,
					})
				}
			}
		}
	}

	return inlines, attaches, nil
}

// SaveDraft 保存草稿：有源 UID 时替换旧草稿，否则新建。
// 返回新的（或保留的）UID 与提示文案。
func (s *Service) SaveDraft(email, pwd string, c Compose) (int64, string, error) {
	raw, err := s.BuildRawEmail(c)
	if err != nil {
		return 0, "", apperr.Internal("构建邮件失败", err)
	}

	s.async(func() { s.saveSentContacts(c.To, email) })

	if c.SourceUID > 0 {
		folder := c.Folder
		if folder == "" {
			folder = s.cfg.FolderDrafts()
		}
		if err := s.UpdateDraft(email, pwd, folder, raw, c.SourceUID); err != nil {
			return 0, "", apperr.Internal("更新草稿失败", err)
		}
		return c.SourceUID, "草稿更新成功", nil
	}

	if err := s.SaveMailToFolder(email, pwd, s.cfg.FolderDrafts(), raw); err != nil {
		return 0, "", apperr.Internal("保存草稿失败", err)
	}
	return 0, "草稿保存成功", nil
}

// SendEmail 发送邮件（立即发送或设置定时发送），返回给前端的提示文案。
func (s *Service) SendEmail(email, pwd string, c Compose) (string, error) {
	if len(c.To) == 0 {
		return "", apperr.BadRequest("收件人不能为空")
	}

	raw, err := s.BuildRawEmail(c)
	if err != nil {
		return "", apperr.Internal("构建邮件失败", err)
	}

	if c.Schedule == "" {
		if msg, err := s.sendNow(email, pwd, c, raw); err != nil {
			return "", err
		} else {
			s.async(func() { s.saveSentContacts(c.To, email) })
			return msg, nil
		}
	}

	msg, err := s.scheduleSend(email, pwd, c, raw)
	if err != nil {
		return "", err
	}
	s.async(func() { s.saveSentContacts(c.To, email) })
	return msg, nil
}

// sendNow 立即发送，并把邮件放入已发送（草稿发送时移动原草稿）。
func (s *Service) sendNow(email, pwd string, c Compose, raw []byte) (string, error) {
	if err := s.ScheduleSendEmail(email, pwd, c.To, c.Cc, raw); err != nil {
		return "", apperr.Internal("发送失败", err)
	}

	var warn string
	if c.SourceUID > 0 {
		if err := s.UpdateMailFlag(email, pwd, s.cfg.FolderDrafts(), []int64{c.SourceUID}, 2, "Draft"); err != nil {
			fmt.Printf("移除 Draft 标记失败: %v\n", err)
		}
		if err := s.MoveMail(email, pwd, s.cfg.FolderDrafts(), s.cfg.FolderSent(), []int64{c.SourceUID}); err != nil {
			warn = "，但移动到已发送失败"
			fmt.Printf("移动到已发送失败: %v\n", err)
		}
	} else {
		if err := s.SaveMailToFolder(email, pwd, s.cfg.FolderSent(), raw); err != nil {
			warn = "，但存入已发送失败"
			fmt.Printf("存入已发送失败: %v\n", err)
		}
	}

	return "发送成功" + warn, nil
}

// scheduleSend 设置定时发送：写入草稿并打上 Draft 标记。
func (s *Service) scheduleSend(email, pwd string, c Compose, raw []byte) (string, error) {
	if err := s.ScheduleSendEmail(email, pwd, c.To, c.Cc, raw); err != nil {
		return "", apperr.Internal("定时发送设置失败", err)
	}

	if err := s.SaveMailToFolder(email, pwd, s.cfg.FolderDrafts(), raw); err != nil {
		return "", apperr.Internal("定时发送已设置，但保存草稿失败", err)
	}

	messageID := utils.ExtractHeader(raw, "Message-ID")
	if messageID == "" {
		fmt.Printf("未找到 Message-ID，跳过定时邮件标记\n")
		return "定时发送已设置", nil
	}

	uid, err := s.draftUIDByMessageID(email, pwd, messageID)
	if err != nil {
		fmt.Printf("定时邮件打标签失败：未找到对应草稿 UID: %v\n", err)
		return "定时发送已设置", nil
	}
	if err := s.UpdateMailFlag(email, pwd, s.cfg.FolderDrafts(), []int64{uid}, 1, "Draft"); err != nil {
		fmt.Printf("标记定时邮件失败: %v\n", err)
	}
	return "定时发送已设置", nil
}

// ScheduleSendEmail 若报文带 X-Schedule-Send 则按时间投递，否则立即投递。
func (s *Service) ScheduleSendEmail(email, pwd string, to, cc []string, raw []byte) error {
	messageID := utils.ExtractHeader(raw, "Message-ID")
	header := utils.ExtractHeader(raw, "X-Schedule-Send")
	if header == "" {
		return s.smtp.Send(email, pwd, to, cc, raw)
	}

	target, err := utils.ParseMailDate(header)
	if err != nil {
		return fmt.Errorf("解析定时时间失败: %w", err)
	}

	wait := time.Until(target)
	if wait <= 0 {
		if err := s.smtp.Send(email, pwd, to, cc, raw); err != nil {
			return err
		}
		if messageID == "" {
			return fmt.Errorf("[定时发送] 未找到 Message-ID，跳过移动操作\n")
		}
		uid, err := s.draftUIDByMessageID(email, pwd, messageID)
		if err != nil {
			return fmt.Errorf("[定时发送] 未找到对应草稿，跳过移动: %v\n", err)
		}
		s.moveDraftToSent(email, pwd, uid)
		return nil
	}

	s.armScheduled(email, pwd, messageID, target, to, cc)
	return nil
}

// runScheduledSend 定时任务到点后的处理：校验草稿仍存在且未取消定时，然后发送并归档。
func (s *Service) runScheduledSend(email, pwd, messageID string, to, cc []string) {
	uid, err := s.draftUIDByMessageID(email, pwd, messageID)
	if err != nil {
		fmt.Printf("[定时发送] 未找到对应草稿，可能已被删除: %v\n", err)
		return
	}

	newRaw, err := s.mailRawByUID(email, pwd, s.cfg.FolderDrafts(), uid)
	if err != nil {
		fmt.Printf("[定时发送] 读取邮件 raw 失败: %v\n", err)
		return
	}

	latest := utils.ExtractHeader(newRaw, "X-Schedule-Send")
	if latest == "" {
		fmt.Printf("[定时发送] 用户已取消定时，跳过发送，Message-ID=%s\n", messageID)
		return
	}

	if err := s.sendWithRetry(email, pwd, to, cc, newRaw); err != nil {
		fmt.Printf("[定时发送] 发送失败 [%s]: %v\n", latest, err)
		return
	}

	s.moveDraftToSent(email, pwd, uid)
	fmt.Printf("[定时发送] 邮件已发送成功并移动到已发送，Message-ID=%s\n", messageID)
}

// sendWithRetry 最多重试 2 次，间隔 5s、15s。
func (s *Service) sendWithRetry(email, pwd string, to, cc []string, raw []byte) error {
	intervals := []time.Duration{5 * time.Second, 15 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= len(intervals); attempt++ {
		if err := s.smtp.Send(email, pwd, to, cc, raw); err != nil {
			lastErr = err
			if attempt < len(intervals) {
				fmt.Printf("[定时发送] 第 %d 次发送失败，%v 后重试: %v\n", attempt+1, intervals[attempt], err)
				s.sleep(intervals[attempt])
				continue
			}
			return fmt.Errorf("重试 %d 次后仍失败: %w", attempt+1, lastErr)
		}
		return nil
	}
	return lastErr
}

// moveDraftToSent 先移除 Draft 标记，再移动草稿到已发送。
func (s *Service) moveDraftToSent(email, pwd string, uid int64) {
	if err := s.UpdateMailFlag(email, pwd, s.cfg.FolderDrafts(), []int64{uid}, 2, "Draft"); err != nil {
		fmt.Printf("[定时发送] 移除 Draft 标记失败: %v\n", err)
	}
	if err := s.MoveMail(email, pwd, s.cfg.FolderDrafts(), s.cfg.FolderSent(), []int64{uid}); err != nil {
		fmt.Printf("[定时发送] 移动到已发送失败（邮件已发出）: %v\n", err)
	}
}

// draftUIDByMessageID 通过 Message-ID 反查草稿 UID。
func (s *Service) draftUIDByMessageID(email, pwd, messageID string) (int64, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return 0, err
	}
	defer client.Logout()

	if err := client.Select(s.cfg.FolderDrafts()); err != nil {
		return 0, fmt.Errorf("选择文件夹失败: %w", err)
	}
	ids, err := client.UIDSearchByHeader("Message-Id", messageID)
	if err != nil {
		return 0, fmt.Errorf("搜索邮件失败: %w", err)
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("未找到对应的邮件")
	}
	return int64(ids[0]), nil
}

// saveSentContacts 异步记录最近联系人（发送成功后调用）。
func (s *Service) saveSentContacts(addresses []string, userEmail string) {
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" || strings.EqualFold(addr, userEmail) {
			continue
		}
		name := ""
		if account, err := s.dir.Get(addr); err == nil {
			name = account.FullName()
		}
		if err := s.contacts.Save(contacts.PrefixSent, userEmail, addr, name); err != nil {
			fmt.Printf("保存联系人失败: %v\n", err)
		}
	}
}
