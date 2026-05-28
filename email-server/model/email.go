package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWT载荷 存放邮箱账号密码
type UserClaims struct {
	Email    string `json:"email"`
	Password string `json:"pwd"`
	jwt.RegisteredClaims
}

// 用户信息
type UserItem struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	IsAdmin  int64  `json:"is_admin"`
	Token    string `json:"token"`
}

// 邮件信息
type MailItem struct {
	Uid       uint32 `json:"uid"`
	Folder    string `json:"folder"`
	From      string `json:"from"`
	To        string `json:"to"`
	Cc        string `json:"cc"`
	Subject   string `json:"subject"`
	SendTime  string `json:"send_time"`
	IsRead    bool   `json:"is_read"`
	HasAttach bool   `json:"has_attach"`
	Text      string `json:"text"`
}

// 登录请求
type LoginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 用户注册请求
type UserReq struct {
	Email           string `json:"email" binding:"required"`
	Password        string `json:"password" binding:"required"`
	PersonFirstName string `json:"person_first_name"`
	PersonLastName  string `json:"person_last_name"`
	IsAdmin         int64  `json:"is_admin" binding:"required"`
}

// 用户删除请求
type UserDeleteReq struct {
	Email string `json:"email" binding:"required"`
}

// 修改密码请求
type PasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// 邮件列表请求
type MailListReq struct {
	Folder  string `json:"folder" binding:"required"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	Keyword string `json:"keyword"`
}

// 邮件详情请求
type MailDetailReq struct {
	Folder string `json:"folder" binding:"required"`
	Uid    uint32 `json:"uid" binding:"required"`
}

// 邮件详情
type MailDetail struct {
	Content     string           `json:"content"`
	Attachments []AttachmentInfo `json:"attachments"`
}

type AttachmentInfo struct {
	PartID      string `json:"part_id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// 下载附件请求
type DownloadAttachReq struct {
	Folder string `json:"folder" binding:"required"`
	Uid    uint32 `json:"uid" binding:"required"`
	PartID string `json:"part_id" binding:"required"`
}

// 邮件移动请求
type MoveMailReq struct {
	FromFolder string   `json:"from_folder" binding:"required"`
	ToFolder   string   `json:"to_folder" binding:"required"`
	Uids       []uint32 `json:"uids" binding:"required"`
}

// 邮件删除请求
type DelMailReq struct {
	Folder string   `json:"folder" binding:"required"`
	Uids   []uint32 `json:"uids" binding:"required"`
}

// 更新邮件状态请求
type UpdateMailStatusReq struct {
	Folder string `json:"folder" binding:"required"`
	Uid    uint32 `json:"uid" binding:"required"`
	Status string `json:"status" binding:"required"`
}
