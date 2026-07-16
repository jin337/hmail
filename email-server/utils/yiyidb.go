package utils

import (
	"email-server/model"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jacoblai/yiyidb"
)

var globalDB *yiyidb.Kvdb

// InitYiyiDB 初始化数据库
func InitYiyiDB(path string) error {
	db, err := yiyidb.OpenKvdb(path, 10) // 10MB 内存
	if err != nil {
		return err
	}
	globalDB = db
	return nil
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

// SaveSentContact 新增/更新联系人（自动覆盖，可修改昵称）
func SaveSentContact(prefixUnit, userEmail, contactEmail, name string) error {
	db := globalDB
	prefix := prefixUnit + ":"
	key := []byte(fmt.Sprintf("%s%s:%s", prefix, userEmail, contactEmail))

	// 昵称不为空时，保存昵称
	if name != "" {
		val, err := json.Marshal(ContactItem{Name: name})
		if err != nil {
			return err
		}
		return db.Put(key, val, 0, nil)
	}

	// 获取旧数据
	oldVal, err := db.Get(key, nil)
	if err == nil && len(oldVal) > 0 {
		// 存在旧记录，直接不执行写入，保留原有数据
		return nil
	}

	// 昵称为空时，使用邮箱前缀作为昵称
	addname := strings.Split(contactEmail, "@")
	name = addname[0]
	val, err := json.Marshal(ContactItem{Name: name})
	if err != nil {
		return err
	}
	return db.Put(key, val, 0, nil)
}

// DelContact 删除单个联系人
func DelContact(prefixUnit, userEmail, contactEmail string) error {
	db := globalDB
	prefix := prefixUnit + ":"
	key := []byte(fmt.Sprintf("%s%s:%s", prefix, userEmail, contactEmail))
	return db.Del(key, nil)
}

// ClearAllContact 清空账号所有联系人
func ClearAllContact(prefixUnit, userEmail string) error {
	db := globalDB
	prefix := prefixUnit + ":"
	prefixAll := []byte(prefix + userEmail + ":")

	tran, err := db.OpenTransaction()
	if err != nil {
		return err
	}
	defer db.Discard(tran)

	iter, err := db.IterStartWith(prefixAll, tran)
	if err != nil {
		return err
	}
	defer db.IterRelease(iter)
	var items []yiyidb.BatItem
	for iter.Seek(prefixAll); iter.Valid(); iter.Next() {
		items = append(items, yiyidb.BatItem{
			Op:    "del",
			Ttl:   0,
			Key:   iter.Key(),
			Value: nil,
		})
	}

	if len(items) > 0 {
		err = db.BatPutOrDel(&items, tran)
		if err != nil {
			return err
		}
		return db.Commit(tran)
	}
	return nil
}

// ListUserContacts 查询用户全部联系人
func ListUserContacts(prefixUnit, userEmail string) ([]model.Contact, error) {
	db := globalDB
	prefix := prefixUnit + ":"
	prefixAll := []byte(prefix + userEmail + ":")
	var list []model.Contact

	tran, err := db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	defer db.Discard(tran)

	iter, err := db.IterStartWith(prefixAll, tran)
	if err != nil {
		return nil, err
	}
	defer db.IterRelease(iter)

	for iter.Seek(prefixAll); iter.Valid(); iter.Next() {
		fullKey := string(iter.Key())
		contactEmail := fullKey[len(prefixAll):]

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
