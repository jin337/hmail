package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"email-server/config"
	"email-server/utils"

	"github.com/emersion/go-imap"
	"github.com/jhillyerd/enmime"
)

// 允许修改的邮件标记（对应 IMAP 系统标记，去掉反斜杠后的名称）。
var allowedFlags = map[string]string{
	"Seen":     imap.SeenFlag,
	"Answered": imap.AnsweredFlag,
	"Flagged":  imap.FlaggedFlag,
	"Deleted":  imap.DeletedFlag,
	"Draft":    imap.DraftFlag,
}

// UpdateMailFlag 添加或移除邮件标记。
func (s *Service) UpdateMailFlag(email, pwd, folder string, uids []int64, opType int64, status string) error {
	flag, ok := allowedFlags[status]
	if !ok {
		return fmt.Errorf("无效的状态标志: \\%s，必须是以下之一: [\\Seen \\Answered \\Flagged \\Deleted \\Draft]", status)
	}

	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return err
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	var add bool
	switch opType {
	case 1:
		add = true
	case 2:
		add = false
	default:
		return fmt.Errorf("操作类型仅支持 1(添加)、2(删除)，传入值：%d", opType)
	}

	if err := client.StoreFlags(toUIDs(uids), add, []string{flag}); err != nil {
		return fmt.Errorf("更新邮件状态失败: %w", err)
	}
	return nil
}

// MoveMail 把邮件移动到目标文件夹。
func (s *Service) MoveMail(email, pwd, fromFolder, toFolder string, uids []int64) error {
	if fromFolder == toFolder {
		return fmt.Errorf("源文件夹和目标文件夹不能相同")
	}
	if err := s.validateFolders(fromFolder, toFolder); err != nil {
		return err
	}

	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return err
	}
	defer client.Logout()

	if err := client.Select(fromFolder); err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", fromFolder, err)
	}
	if err := client.Move(toUIDs(uids), toFolder); err != nil {
		return fmt.Errorf("移动邮件失败: %w", err)
	}
	return nil
}

// DeleteMail 永久删除邮件，并清理该邮件产生的内联图片缓存。
func (s *Service) DeleteMail(email, pwd, folder string, uids []int64) error {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return err
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}
	if err := client.DeleteForever(toUIDs(uids)); err != nil {
		return fmt.Errorf("删除邮件失败: %w", err)
	}

	// 缓存清理失败不应影响删除结果（旧实现会因此返回 500）。
	base := filepath.Join(s.cfg.StaticDir, "images", safePathSegment(email))
	for _, uid := range uids {
		if err := os.RemoveAll(filepath.Join(base, fmt.Sprint(uid))); err != nil {
			_ = err
		}
	}
	return nil
}

// DownloadAttachment 读取指定附件的文件名与内容。
func (s *Service) DownloadAttachment(email, pwd, folder string, uid int64, partID string) (string, []byte, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return "", nil, err
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return "", nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	msg, err := client.FetchUID(uint32(uid))
	if err != nil {
		return "", nil, fmt.Errorf("获取邮件失败: %w", err)
	}

	env, err := enmime.ReadEnvelope(bytes.NewReader(msg.Raw))
	if err != nil {
		return "", nil, fmt.Errorf("解析邮件失败: %w", err)
	}

	for _, att := range env.Attachments {
		if att.PartID == partID {
			return att.FileName, att.Content, nil
		}
	}
	for _, inline := range env.Inlines {
		if inline.PartID == partID {
			return inline.FileName, inline.Content, nil
		}
	}

	return "", nil, fmt.Errorf("未找到附件 PartID: %s", partID)
}

// SaveMailToFolder 追加一封原始邮件到指定文件夹。
func (s *Service) SaveMailToFolder(email, pwd, folder string, raw []byte) error {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}
	if err := client.Append(folder, []string{imap.SeenFlag}, time.Now(), raw); err != nil {
		return fmt.Errorf("保存邮件到文件夹 %s 失败: %w", folder, err)
	}
	return nil
}

// UpdateDraft 用新内容替换草稿：删除旧草稿再追加新草稿。
// 注意：该操作会改变草稿的 UID（与旧实现一致）。
func (s *Service) UpdateDraft(email, pwd, folder string, raw []byte, uid int64) error {
	if err := s.DeleteMail(email, pwd, folder, []int64{uid}); err != nil {
		return fmt.Errorf("删除旧草稿失败: %w", err)
	}
	if err := s.SaveMailToFolder(email, pwd, folder, raw); err != nil {
		return fmt.Errorf("保存新草稿失败: %w", err)
	}
	return nil
}

// UnScheduleEmail 取消定时发送：移除 X-Schedule-Send 头部后重写草稿。
//
// IMAP 没有"就地修改邮件"的能力，因此沿用"删除 + 重新追加"的方式，
// 但头部处理改为按行解析，支持折叠头，避免旧实现的正切分误伤正文。
func (s *Service) UnScheduleEmail(email, pwd, folder string, uid int64, opType int64, status string) error {
	if err := s.UpdateMailFlag(email, pwd, folder, []int64{uid}, opType, status); err != nil {
		return fmt.Errorf("更新邮件标志失败: %w", err)
	}

	raw, err := s.mailRawByUID(email, pwd, folder, uid)
	if err != nil {
		return fmt.Errorf("获取邮件原始数据失败: %w", err)
	}

	messageID := utils.ExtractHeader(raw, "Message-ID")
	if utils.ExtractHeader(raw, "X-Schedule-Send") == "" {
		return nil
	}

	newRaw := RemoveHeader(raw, "X-Schedule-Send")
	if err := s.DeleteMail(email, pwd, folder, []int64{uid}); err != nil {
		return fmt.Errorf("删除旧邮件失败: %w", err)
	}
	if err := s.SaveMailToFolder(email, pwd, folder, newRaw); err != nil {
		return fmt.Errorf("保存新邮件失败: %w", err)
	}
	fmt.Printf("定时任务已取消，Message-ID:%s\n", messageID)
	return nil
}

// mailRawByUID 读取原始邮件，供取消定时等场景使用。
func (s *Service) mailRawByUID(email, pwd, folder string, uid int64) ([]byte, error) {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return nil, fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer client.Logout()

	if err := client.Select(folder); err != nil {
		return nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}
	msg, err := client.FetchUID(uint32(uid))
	if err != nil {
		return nil, fmt.Errorf("未找到邮件 UID: %d", uid)
	}
	return msg.Raw, nil
}

// validateFolders 校验文件夹是否属于标准文件夹（Star 是前端虚拟文件夹，不能作为真实操作目标）。
func (s *Service) validateFolders(folders ...string) error {
	for _, folder := range folders {
		valid := false
		for _, f := range config.DefaultFolders {
			if strings.EqualFold(f, folder) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("folder 参数无效: %s", folder)
		}
	}
	return nil
}

func toUIDs(uids []int64) []uint32 {
	out := make([]uint32, 0, len(uids))
	for _, uid := range uids {
		if uid > 0 {
			out = append(out, uint32(uid))
		}
	}
	return out
}

// RemoveHeader 删除邮件头部中的指定字段（含折叠续行），保留其余内容不变。
func RemoveHeader(raw []byte, key string) []byte {
	headerEnd := bytes.Index(raw, []byte("\r\n\r\n"))
	if headerEnd < 0 {
		return raw
	}
	headerBlock := raw[:headerEnd]
	body := raw[headerEnd:]

	lines := strings.Split(strings.ReplaceAll(string(headerBlock), "\r\n", "\n"), "\n")
	keyPrefix := strings.ToLower(key) + ":"

	var out []string
	skipping := false
	for _, line := range lines {
		if skipping && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			continue // 折叠续行
		}
		skipping = false
		if strings.HasPrefix(strings.ToLower(line), keyPrefix) {
			skipping = true
			continue
		}
		out = append(out, line)
	}

	rebuild := strings.Join(out, "\r\n")
	// body 自带 "\r\n\r\n" 分隔符，直接拼接即可恢复原结构。
	return append([]byte(rebuild), body...)
}
