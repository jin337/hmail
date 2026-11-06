// Package directory 抽象 hMailServer 的账号目录管理。
//
// Windows 下通过 COM 调用 hMailServer.Application 实现；
// 其它平台提供返回 ErrUnsupported 的实现，便于在非 Windows 环境编译与测试。
package directory

import (
	"errors"
	"time"
)

// ErrUnsupported 表示当前平台不支持账号目录管理。
var ErrUnsupported = errors.New("当前平台不支持 hMailServer 账号管理")

// Account 是 hMailServer 中的一个邮箱账号。
type Account struct {
	ID         int64
	Address    string
	FirstName  string
	LastName   string
	AdminLevel int64
	LastLogon  time.Time
}

// FullName 返回"名+姓"拼接的显示名称，与原实现一致（不加空格）。
func (a *Account) FullName() string { return a.FirstName + a.LastName }

// Directory 定义账号目录操作。
type Directory interface {
	// Get 按完整邮箱地址查找账号。
	Get(email string) (*Account, error)
	// List 列出指定域名下的全部账号。
	List(domain string) ([]Account, error)
	// Create 创建账号（调用方负责后续创建标准文件夹）。
	Create(email, password, firstName, lastName string, adminLevel int64) error
	// Update 更新姓名与管理员级别；adminLevel < 0 表示不修改。
	Update(email, firstName, lastName string, adminLevel int64) error
	// Delete 删除账号。
	Delete(email string) error
	// SetPassword 直接设置账号密码（调用方需先验证旧密码）。
	SetPassword(email, password string) error
	// DisplayName 返回 RFC2047 编码后的 "姓名 <邮箱>"，用于写入邮件头。
	DisplayName(email string) (string, error)
}
