// email-server 是 hMailServer 的 Web 邮件网关：后端提供 IMAP/SMTP 代理与账号管理接口。
package main

import (
	"context"
	"email-server/config"
	"email-server/contacts"
	"email-server/controller"
	"email-server/directory"
	"email-server/mailbox"
	"email-server/router"
	"email-server/service"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	fmt.Printf("配置加载完成: mode=%s prefix=%s listen=:%d imap=%s:%d smtp=%s:%d\n",
		cfg.Mode, cfg.APIPrefix, cfg.Server.Port, cfg.IMAP.Host, cfg.IMAP.Port, cfg.SMTP.Host, cfg.SMTP.Port)

	store, err := contacts.Open(cfg.ContactsPath)
	if err != nil {
		log.Fatalf("联系人数据库初始化失败: %v", err)
	}
	defer store.Close()

	svc := service.New(service.Deps{
		Config:    cfg,
		IMAP:      mailbox.NewIMAPFactory(cfg.IMAP.Host, cfg.IMAP.Port),
		SMTP:      mailbox.NewSMTPSender(cfg.SMTP.Host, cfg.SMTP.Port),
		Directory: directory.New(cfg.AdminPassword),
		Contacts:  store,
	})

	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Client-Host"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	engine.MaxMultipartMemory = cfg.MaxUploadMB << 20

	router.Setup(engine, controller.New(svc, cfg), cfg)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: engine,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待退出信号后优雅关闭，便于测试与本地调试时清理连接。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("服务关闭异常: %v\n", err)
	}
}
