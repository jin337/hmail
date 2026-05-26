package router

import (
	"email-server/controller"
	"email-server/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	public := r.Group("/api/mail")
	{
		public.POST("/login", controller.Login)

		// 需要 JWT 认证
		use := public.Use(middleware.JWTAuth())
		{
			use.POST("/list", controller.MailList)
			use.POST("/detail", controller.MailDetail)
			use.POST("/status", controller.MarkRead)
			use.POST("/download", controller.DownloadAttachment)
			use.POST("/move", controller.MoveMail)
			use.POST("/delete", controller.DeleteMail)
			use.POST("/save-draft", controller.SaveDraft)
			use.POST("/send", controller.SendEmail)
			use.POST("/chgpwd", controller.ChangePassword)
		}
	}
}
