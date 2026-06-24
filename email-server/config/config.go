package config

import "github.com/ltyyz/goprofile"

// 标准邮件文件夹
const (
	FolderInbox   = "INBOX"
	FolderSent    = "Sent"
	FolderDrafts  = "Drafts"
	FolderDeleted = "Deleted"
	FolderJunk    = "Junk"
	FolderArchive = "Archive"
)

// 默认文件夹 - 登录时验证
var DefaultFolders = []string{
	FolderInbox,
	FolderSent,
	FolderDrafts,
	FolderDeleted,
	FolderJunk,
	FolderArchive,
}

func Init() {
	goprofile.Load()
}

func GetConfig(name string) string {
	return goprofile.GetEnv(name)
}
