package config

var Port = 8058

// 邮件服务器配置
var (
	HmailHost = "127.0.0.1"
	// HmailHost = "221.226.24.62" // 局域网hMailServer地址
	SmtpPort = 25
	ImapPort = 143
	AdminPwd = "a123456"
)

// JWT配置
const (
	JwtSecretKey  = "email-server-hmail-20260525"
	TokenPrefix   = "Bearer "
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
