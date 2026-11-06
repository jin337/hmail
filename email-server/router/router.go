// Package router 负责 HTTP 路由注册。
package router

import (
	"email-server/config"
	"email-server/controller"
	"email-server/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 注册所有路由。
//
// 路由前缀取自配置（debug 默认为 /api，release 默认为 /），
// 前端固定调用 /api/...，因此生产环境建议显式设置 api.prefix=/api。
func Setup(r *gin.Engine, h *controller.Handler, cfg config.Config) {
	switch cfg.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	}

	r.Static("/static", cfg.StaticDir)

	public := r.Group(cfg.APIPrefix)
	auth := middleware.JWTAuth(cfg.JWT.Secret, h.EnsureScheduledRestored)

	public.POST("/login", h.Login)
	public.GET("/viewfile", h.ViewFile)

	user := public.Group("/user")
	{
		user.POST("/chgpwd", auth, h.ChangePassword)
		user.POST("/list", auth, h.UserList)
		user.POST("/create", auth, h.CreateUser)
		user.POST("/delete", auth, h.DeleteUser)
		user.POST("/update", auth, h.UpdateUser)
		user.POST("/uploadavatar", auth, h.UploadAvatar)

		contact := user.Group("/contact")
		{
			contact.POST("/list", auth, h.ContactList)
			contact.POST("/save", auth, h.SaveContact)
			contact.POST("/delete", auth, h.DeleteContact)
			contact.POST("/clear", auth, h.ClearContact)
		}
	}

	mail := public.Group("/mail")
	{
		mail.POST("/list", auth, h.MailList)
		mail.POST("/star-list", auth, h.StarMailList)
		mail.POST("/unread-total", auth, h.UnreadMailTotal)
		mail.POST("/detail", auth, h.MailDetail)
		mail.POST("/status", auth, h.MarkFlag)
		mail.POST("/download", auth, h.DownloadAttachment)
		mail.POST("/move", auth, h.MoveMail)
		mail.POST("/delete", auth, h.DeleteMail)
		mail.POST("/save-draft", auth, h.SaveDraft)
		mail.POST("/send", auth, h.SendEmail)
		mail.POST("/un-schedule", auth, h.UnScheduleEmail)
	}
}
