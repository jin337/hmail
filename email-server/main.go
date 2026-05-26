package main

import (
	"email-server/config"
	"email-server/router"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 上传文件内存缓存上限:20M
	r.MaxMultipartMemory = 20 << 20

	// 设置路由
	router.SetupRouter(r)

	// 启动服务
	r.Run(fmt.Sprintf(":%d", config.Port))
}
