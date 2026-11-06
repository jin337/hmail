// Package contacts 提供联系人（最近联系人 / 通讯录）的持久化抽象。
//
// 生产实现基于 yiyidb 的 KV 存储，key 形如 "{prefix}:{userEmail}:{contactEmail}"。
package contacts

// Item 是一位联系人。
type Item struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Store 定义联系人读写接口。
type Store interface {
	// Save 新增或更新联系人；name 为空时沿用旧名称，旧名称也没有则使用邮箱前缀。
	Save(prefix, userEmail, contactEmail, name string) error
	// Delete 删除单个联系人。
	Delete(prefix, userEmail, contactEmail string) error
	// Clear 清空某用户在指定前缀下的全部联系人。
	Clear(prefix, userEmail string) error
	// List 返回某用户在指定前缀下的全部联系人。
	List(prefix, userEmail string) ([]Item, error)
	// Close 释放底层资源。
	Close() error
}

// 常用前缀，避免调用方散落字符串字面量。
const (
	PrefixSent    = "user_sent"
	PrefixContact = "user_contact"
)
