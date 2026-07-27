package main

import (
	"email-server/config"
	"email-server/constant"
	"email-server/router"
	"email-server/task"
	"email-server/utils"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 配置注册
	config.Init()

	// 初始化yiyidb
	err := utils.InitYiyiDB("./cache/contact_db")
	if err != nil {
		fmt.Printf("数据库初始化失败: %v\n", err)
	}
	defer utils.CloseDB()

	// 定时任务，文件超过24小时自动清理
	imageDir := "./static/images"
	fileExpire := 24 * time.Hour

	// 服务启动立刻执行一次过期清理
	task.CleanExpireImageDir(imageDir, fileExpire)

	// 初始化定时任务：每日0点清理
	cronScheduler := task.InitCronTask(imageDir, fileExpire)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		ctx := cronScheduler.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(30 * time.Second):
		}
		log.Println("定时任务已退出")
		os.Exit(0)
	}()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Client-Host"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.MaxMultipartMemory = 20 << 20
	router.SetupRouter(r)

	r.Run(fmt.Sprintf(":%s", config.GetConfig(constant.MailServerPort)))
}
