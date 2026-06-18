package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWT载荷 存放邮箱账号密码
type UserClaims struct {
	Email                string `json:"email"` // 邮箱账号
	Password             string `json:"pwd"`   // 邮箱密码
	jwt.RegisteredClaims        // JWT标准字段
}

// 用户信息
type UserItem struct {
	ID       int64  `json:"id"`        // 用户ID
	Email    string `json:"email"`     // 邮箱账号
	FullName string `json:"full_name"` // 姓名
	IsAdmin  int64  `json:"is_admin"`  // 0:用户，1：域管理员，2：服务器管理员
	Token    string `json:"token"`     // 访问令牌
}

// 用户列表
type UserList struct {
	ID              int64  `json:"id"`                // 用户ID
	Email           string `json:"email"`             // 邮箱账号
	FullName        string `json:"full_name"`         // 姓名
	PersonFirstName string `json:"person_first_name"` // 名
	PersonLastName  string `json:"person_last_name"`  // 姓
	IsAdmin         int64  `json:"is_admin"`          // 0:用户，1：域管理员，2：服务器管理员
	LastLogonTime   string `json:"last_logon_time"`   // 最后登录时间
}

// 邮件信息
type MailItem struct {
	Uid        int64       `json:"uid"`        // 邮件ID
	MessageId  string      `json:"message_id"` // 邮件唯一ID
	ReplyTo    *string     `json:"reply_to"`   // 回复地址
	References *string     `json:"references"` // 引用
	Folder     string      `json:"folder"`     // 文件夹
	From       *string     `json:"from"`       // 发件人
	FromName   *string     `json:"from_name"`  // 发件人姓名
	To         *string     `json:"to"`         // 收件人
	ToInfo     []*MailInfo `json:"to_info"`    // 收件人姓名
	Cc         *string     `json:"cc"`         // 抄送人
	CcInfo     []*MailInfo `json:"cc_info"`    // 抄送人姓名
	Subject    string      `json:"subject"`    // 主题
	SendTime   string      `json:"send_time"`  // 发送时间
	IsRead     bool        `json:"is_read"`    // 是否已读
	HasAttach  bool        `json:"has_attach"` // 是否有附件
	Text       string      `json:"text"`       // 邮件内容
	Size       string      `json:"size"`       // 邮件大小
}

type MailInfo struct {
	Email string `json:"email"` // 邮箱账号
	Name  string `json:"name"`
}

// 邮件详情
type MailDetail struct {
	Content     string           `json:"content"`     // 邮件内容
	Attachments []AttachmentInfo `json:"attachments"` // 附件信息
	AttachSize  string           `json:"attach_size"` // 邮件大小
}

// 附件信息
type AttachmentInfo struct {
	PartID      string `json:"part_id"`      // 附件ID
	FileName    string `json:"file_name"`    // 文件名
	ContentType string `json:"content_type"` // 内容类型
	FileType    string `json:"file_type"`    // 文件类型
	Size        int64  `json:"size"`         // 文件大小
}

// 登录请求
type LoginReq struct {
	Email    string `json:"email" binding:"required"`    // 邮箱账号
	Password string `json:"password" binding:"required"` // 邮箱密码
}

// 用户注册请求
type UserReq struct {
	Email           string `json:"email" binding:"required"`    // 邮箱账号
	Password        string `json:"password" binding:"required"` // 邮箱密码
	PersonFirstName string `json:"person_first_name"`           // 用户名
	PersonLastName  string `json:"person_last_name"`            // 姓名
	IsAdmin         int64  `json:"is_admin"`                    // 0:用户，1：域管理员，2：服务器管理员
}

// 用户注册请求
type UserUpdateReq struct {
	Email           string `json:"email" binding:"required"` // 邮箱账号
	PersonFirstName string `json:"person_first_name"`        // 用户名
	PersonLastName  string `json:"person_last_name"`         // 姓名
	IsAdmin         int64  `json:"is_admin"`                 // 0:用户，1：域管理员，2：服务器管理员
}

// 用户删除请求
type UserDeleteReq struct {
	Email string `json:"email" binding:"required"` // 邮箱账号
}

// 修改密码请求
type PasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"` // 旧密码
	NewPassword string `json:"new_password" binding:"required"` // 新密码
}

// 邮件列表请求
type MailListReq struct {
	Folder  string `json:"folder" binding:"required"` // 文件夹
	Page    int64  `json:"page"`                      // 页码
	Size    int64  `json:"size"`                      // 每页数量
	Keyword string `json:"keyword"`                   // 关键字
}

// 邮件详情请求
type MailDetailReq struct {
	Folder string `json:"folder" binding:"required"` // 文件夹
	Uid    int64  `json:"uid" binding:"required"`    // 邮件ID
}

// 下载附件请求
type DownloadAttachReq struct {
	Folder string `json:"folder" binding:"required"`  // 文件夹
	Uid    int64  `json:"uid" binding:"required"`     // 邮件ID
	PartID string `json:"part_id" binding:"required"` // 附件ID
}

// 邮件移动请求
type MoveMailReq struct {
	FromFolder string  `json:"from_folder" binding:"required"` // 源文件夹
	ToFolder   string  `json:"to_folder" binding:"required"`   // 目标文件夹
	Uids       []int64 `json:"uids" binding:"required"`        // 邮件ID列表
}

// 邮件删除请求
type DelMailReq struct {
	Folder string  `json:"folder" binding:"required"` // 文件夹
	Uids   []int64 `json:"uids" binding:"required"`   // 邮件ID列表
}

// 更新邮件状态请求
type UpdateMailStatusReq struct {
	Folder string `json:"folder" binding:"required"` // 文件夹
	Uid    int64  `json:"uid" binding:"required"`    // 邮件ID
	Status string `json:"status" binding:"required"` // 状态
}
