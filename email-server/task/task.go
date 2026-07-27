package task

import (
	"email-server/utils"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// CleanExpireImageDir 执行单次过期文件清理
func CleanExpireImageDir(cleanDir string, expire time.Duration) {
	log.Printf("【启动清理】开始扫描过期临时图片目录 %s", cleanDir)
	if err := utils.CleanDirExpire(cleanDir, expire); err != nil {
		log.Printf("【启动清理】执行异常：%v", err)
	}
}

// InitCronTask 初始化定时调度
func InitCronTask(cleanDir string, expire time.Duration) *cron.Cron {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("【定时任务】加载时区失败: %v", err)
	}
	c := cron.New(cron.WithLocation(loc))

	var cleanLock bool
	_, err = c.AddFunc("0 0 * * *", func() {
		if cleanLock {
			log.Printf("【定时任务】目录清理任务运行中，跳过本次调度")
			return
		}
		cleanLock = true
		defer func() {
			cleanLock = false
		}()

		log.Printf("【定时任务】执行过期图片清理 %s", cleanDir)
		if err := utils.CleanDirExpire(cleanDir, expire); err != nil {
			log.Printf("【定时任务】清理失败：%v", err)
		}
	})
	if err != nil {
		log.Fatalf("注册定时任务失败: %v", err)
	}

	c.Start()
	return c
}
