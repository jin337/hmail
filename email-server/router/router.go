package router

import (
	"email-server/config"
	"email-server/constant"
	"email-server/controller"
	"email-server/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	// 判断环境变量
	envValue := config.GetConfig(constant.GinMode)
	apiPrefix := "/api"
	if envValue == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r.Static("/static", "./static")

	// 创建路由组
	public := r.Group(apiPrefix)

	{
		public.POST("/login", controller.Login)
		public.GET("/viewfile", controller.ViewFile)

		// 用户相关
		user := public.Group("/user")
		{
			// 需要 JWT 认证
			useAuth := user.Use(middleware.JWTAuth())
			{
				useAuth.POST("/chgpwd", controller.ChangePassword)
				useAuth.POST("/list", controller.UserList)
				useAuth.POST("/create", controller.CreateUser)
				useAuth.POST("/delete", controller.DeleteUser)
				useAuth.POST("/update", controller.UpdateUser)
				useAuth.POST("/uploadavatar", controller.UploadAvatar)

				// 联系人相关
				contact := user.Group("/contact")
				{
					contact.POST("/list", controller.ContactList)
					contact.POST("/save", controller.SaveContact)
					contact.POST("/delete", controller.DeleteContact)
					contact.POST("/clear", controller.ClearContact)
				}
			}
		}

		// 邮件相关
		mail := public.Group("/mail")
		{
			// 需要 JWT 认证
			mailAuth := mail.Use(middleware.JWTAuth())
			{
				mailAuth.POST("/list", controller.MailList)
				mailAuth.POST("/star-list", controller.StarMailList)
				mailAuth.POST("/detail", controller.MailDetail)
				mailAuth.POST("/status", controller.MarkFlag)
				mailAuth.POST("/download", controller.DownloadAttachment)
				mailAuth.POST("/move", controller.MoveMail)
				mailAuth.POST("/delete", controller.DeleteMail)
				mailAuth.POST("/save-draft", controller.SaveDraft)
				mailAuth.POST("/send", controller.SendEmail)
				mailAuth.POST("/un-schedule", controller.UnScheduleEmail)
			}
		}
	}
}
