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
	var apiPrefix string
	if envValue == "debug" {
		apiPrefix = "/api"
		gin.SetMode(gin.DebugMode)
	} else {
		apiPrefix = "/"
		gin.SetMode(gin.ReleaseMode)
	}

	r.Static("/static", "./static") // 路径映射

	// 创建路由组
	public := r.Group(apiPrefix)

	{
		public.POST("/login", controller.Login)      // 登录
		public.GET("/viewfile", controller.ViewFile) // 查看文件

		// 用户相关
		user := public.Group("/user")
		{
			// 需要 JWT 认证
			useAuth := user.Use(middleware.JWTAuth())
			{
				useAuth.POST("/chgpwd", controller.ChangePassword)     // 修改密码
				useAuth.POST("/list", controller.UserList)             // 用户列表
				useAuth.POST("/create", controller.CreateUser)         // 创建用户
				useAuth.POST("/delete", controller.DeleteUser)         // 删除用户
				useAuth.POST("/update", controller.UpdateUser)         // 修改用户信息
				useAuth.POST("/uploadavatar", controller.UploadAvatar) // 上传头像

				// 联系人相关
				contact := user.Group("/contact")
				{
					contact.POST("/list", controller.ContactList)     // 联系人列表
					contact.POST("/save", controller.SaveContact)     // 保存联系人
					contact.POST("/delete", controller.DeleteContact) // 删除联系人
					contact.POST("/clear", controller.ClearContact)   // 清空联系人
				}
			}
		}

		// 邮件相关
		mail := public.Group("/mail")
		{
			// 需要 JWT 认证
			mailAuth := mail.Use(middleware.JWTAuth())
			{
				mailAuth.POST("/list", controller.MailList)               // 邮件列表
				mailAuth.POST("/star-list", controller.StarMailList)      // 星标邮件列表
				mailAuth.POST("/detail", controller.MailDetail)           // 邮件详情
				mailAuth.POST("/status", controller.MarkFlag)             // 标记邮件状态
				mailAuth.POST("/download", controller.DownloadAttachment) // 下载附件
				mailAuth.POST("/move", controller.MoveMail)               // 移动邮件
				mailAuth.POST("/delete", controller.DeleteMail)           // 删除邮件
				mailAuth.POST("/save-draft", controller.SaveDraft)        // 保存草稿
				mailAuth.POST("/send", controller.SendEmail)              // 发送邮件
				mailAuth.POST("/un-schedule", controller.UnScheduleEmail) // 取消定时发送
			}
		}
	}
}
