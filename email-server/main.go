package main

import (
	"email-server/config"
	"email-server/constant"
	"email-server/router"
	"email-server/utils"
	"fmt"
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

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Client-Host"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.MaxMultipartMemory = 20 << 20 // 20M
	router.SetupRouter(r)

	r.Run(fmt.Sprintf(":%s", config.GetConfig(constant.MailServerPort)))
}
