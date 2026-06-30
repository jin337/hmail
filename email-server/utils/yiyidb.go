package utils

import (
	"email-server/model"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jacoblai/yiyidb"
)

var globalDB *yiyidb.Kvdb

const prefixUserSent = "user_sent:"

// InitYiyiDB 初始化数据库
func InitYiyiDB(path string) error {
	db, err := yiyidb.OpenKvdb(path, 10) // 10MB 内存
	if err != nil {
		return err
	}
	globalDB = db
	return nil
}

func GetDB() *yiyidb.Kvdb {
	return globalDB
}

func CloseDB() error {
	if globalDB != nil {
		return globalDB.Close()
	}
	return nil
}

// ContactItem 存储到yiyidb的value结构体
type ContactItem struct {
	Name string `json:"name"`
}

// AddSentContact 新增/更新联系人（自动覆盖，可修改昵称）
// userEmail: 当前登录用户邮箱
// contactEmail: 对方邮箱
// name: 联系人昵称，空则默认等于邮箱
func AddSentContact(userEmail, contactEmail, name string) error {
	db := GetDB()
	key := []byte(fmt.Sprintf("%s%s:%s", prefixUserSent, userEmail, contactEmail))

	// 昵称为空时默认用邮箱
	if name == "" {
		addname := strings.Split(contactEmail, "@")
		name = addname[0]
	}
	val, _ := json.Marshal(ContactItem{Name: name})
	return db.Put(key, val, 0, nil)
}

// DelContact 删除单个联系人
func DelContact(userEmail, contactEmail string) error {
	db := GetDB()
	key := []byte(fmt.Sprintf("%s%s:%s", prefixUserSent, userEmail, contactEmail))
	return db.Del(key, nil)
}

// ClearAllContact 清空账号所有联系人
func ClearAllContact(userEmail string) error {
	db := GetDB()
	prefix := []byte(prefixUserSent + userEmail + ":")

	tran, err := db.OpenTransaction()
	if err != nil {
		return err
	}
	defer db.Discard(tran)

	iter, err := db.IterStartWith(prefix, tran)
	if err != nil {
		return err
	}
	defer db.IterRelease(iter)
	var items []yiyidb.BatItem
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		items = append(items, yiyidb.BatItem{
			Key:   iter.Key(),
			Value: nil, // nil表示删除
		})
	}

	if len(items) > 0 {
		return db.BatPutOrDel(&items, tran)
	}
	return nil
}

// ListUserContacts 查询用户全部联系人
func ListUserContacts(userEmail string) ([]model.Contact, error) {
	db := GetDB()
	prefix := []byte(prefixUserSent + userEmail + ":")
	var list []model.Contact

	tran, err := db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	defer db.Discard(tran)

	iter, err := db.IterStartWith(prefix, tran)
	if err != nil {
		return nil, err
	}
	defer db.IterRelease(iter)

	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		fullKey := string(iter.Key())
		contactEmail := fullKey[len(prefix):]

		// 获取昵称
		var item ContactItem
		_ = json.Unmarshal(iter.Value(), &item)

		list = append(list, model.Contact{
			Email: contactEmail,
			Name:  item.Name,
		})
	}
	return list, nil
}
