package config

import (
	"os"
)

// 邮件服务器配置
var (
	Port      = 8058
	HmailHost = "127.0.0.1"
	SmtpPort  = 25
	ImapPort  = 993
	AdminPwd  string
)

func init() {
	envValue := os.Getenv("GIN_MODE")
	if envValue == "debug" {
		AdminPwd = "a123456"
	} else {
		AdminPwd = "a123321"
	}
}

// JWT配置
const (
	JwtSecretKey  = "email-server-hmail-20260525"
	JwtExpireHour = 24
)

// 标准邮件文件夹
const (
	FolderInbox   = "INBOX"
	FolderSent    = "Sent"
	FolderDrafts  = "Drafts"
	FolderDeleted = "Deleted"
	FolderJunk    = "Junk"
)

// 默认文件夹 - 登录时验证
var DefaultFolders = []string{
	FolderInbox,
	FolderSent,
	FolderDrafts,
	FolderDeleted,
	FolderJunk,
}
