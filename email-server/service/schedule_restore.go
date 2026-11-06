package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"email-server/utils"

	"github.com/jhillyerd/enmime"
)

// EnsureScheduledRestored 在进程启动后首次见到某个用户时，扫描其草稿箱恢复定时任务。
//
// 定时任务原本只存在于进程内存中，服务重启会丢失。恢复采用"按用户懒加载"的方式：
// 登录后前端会立刻发起带 JWT 的请求，鉴权中间件调用到这里，因此不需要在启动时
// 拿到所有账号的密码。同一用户只扫描一次；扫描失败时允许下次请求重试。
func (s *Service) EnsureScheduledRestored(email, pwd string) {
	if email == "" || pwd == "" {
		return
	}

	s.restoreMu.Lock()
	if _, ok := s.restored[email]; ok {
		s.restoreMu.Unlock()
		return
	}
	s.restored[email] = struct{}{}
	s.restoreMu.Unlock()

	s.async(func() {
		if err := s.restoreScheduled(email, pwd); err != nil {
			fmt.Printf("[定时发送] 恢复 %s 的定时任务失败: %v\n", email, err)
			s.restoreMu.Lock()
			delete(s.restored, email) // 允许下次请求再次尝试
			s.restoreMu.Unlock()
		}
	})
}

// restoreScheduled 读取草稿箱全部邮件，为仍带 X-Schedule-Send 的草稿重新注册定时任务。
func (s *Service) restoreScheduled(email, pwd string) error {
	client, err := s.imap.Connect(email, pwd)
	if err != nil {
		return fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer client.Logout()

	if err := client.Select(s.cfg.FolderDrafts()); err != nil {
		return fmt.Errorf("选择草稿箱失败: %w", err)
	}
	msgs, err := client.FetchAll()
	if err != nil {
		return fmt.Errorf("读取草稿箱失败: %w", err)
	}

	armed := 0
	for _, msg := range msgs {
		env, err := enmime.ReadEnvelope(bytes.NewReader(msg.Raw))
		if err != nil {
			continue
		}
		messageID := strings.TrimSpace(env.GetHeader("Message-Id"))
		rawSchedule := strings.TrimSpace(env.GetHeader("X-Schedule-Send"))
		if messageID == "" || rawSchedule == "" {
			continue
		}
		target, err := utils.ParseMailDate(rawSchedule)
		if err != nil {
			fmt.Printf("[定时发送] 跳过草稿 UID=%d：定时时间无法解析: %v\n", msg.UID, err)
			continue
		}

		toHeader, _, _ := utils.FormatMailName(env.GetHeader("To"))
		ccHeader, _, _ := utils.FormatMailName(env.GetHeader("Cc"))
		var to, cc []string
		if toHeader != nil {
			to = utils.SplitAddressList(*toHeader)
		}
		if ccHeader != nil {
			cc = utils.SplitAddressList(*ccHeader)
		}
		if len(to) == 0 {
			fmt.Printf("[定时发送] 跳过草稿 UID=%d：缺少收件人\n", msg.UID)
			continue
		}
		s.armScheduled(email, pwd, messageID, target, to, cc)
		armed++
	}

	if armed > 0 {
		fmt.Printf("[定时发送] 已为用户 %s 从草稿箱恢复 %d 个定时任务\n", email, armed)
	}
	return nil
}

// armScheduled 注册一个到点执行的定时任务；同一封草稿（Message-ID）只会注册一次。
func (s *Service) armScheduled(email, pwd, messageID string, target time.Time, to, cc []string) {
	key := schedKey(email, messageID)
	if messageID != "" && !s.markScheduled(key) {
		fmt.Printf("[定时发送] 任务已存在，跳过重复注册，Message-ID=%s\n", messageID)
		return
	}

	s.async(func() {
		if messageID != "" {
			defer s.unmarkScheduled(key)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[定时发送] Message-ID=%s: %v\n", messageID, r)
			}
		}()

		wait := time.Until(target)
		if wait > 0 {
			fmt.Printf("[定时发送] 任务已启动，等待 %v 后发送，Message-ID=%s\n", wait, messageID)
			s.sleep(wait)
		} else {
			// 服务重启导致错过时间：立即补发。
			fmt.Printf("[定时发送] 已错过定时时间 %v，立即补发，Message-ID=%s\n", -wait, messageID)
		}
		s.runScheduledSend(email, pwd, messageID, to, cc)
	})
}

func schedKey(email, messageID string) string {
	return strings.ToLower(email) + "|" + messageID
}

func (s *Service) markScheduled(key string) bool {
	s.schedMu.Lock()
	defer s.schedMu.Unlock()
	if _, ok := s.sched[key]; ok {
		return false
	}
	s.sched[key] = struct{}{}
	return true
}

func (s *Service) unmarkScheduled(key string) {
	s.schedMu.Lock()
	delete(s.sched, key)
	s.schedMu.Unlock()
}

// ScheduledCount 仅供测试与诊断：返回当前已注册的定时任务数量。
func (s *Service) ScheduledCount() int {
	s.schedMu.Lock()
	defer s.schedMu.Unlock()
	return len(s.sched)
}
