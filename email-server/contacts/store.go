package contacts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jacoblai/yiyidb"
)

// kvStore 是基于 yiyidb 的联系人存储实现。
type kvStore struct {
	db *yiyidb.Kvdb
}

// Open 打开（或创建）联系人数据库。
func Open(path string) (Store, error) {
	db, err := yiyidb.OpenKvdb(path, 10) // 10MB 内存
	if err != nil {
		return nil, err
	}
	return &kvStore{db: db}, nil
}

type record struct {
	Name string `json:"name"`
}

func (s *kvStore) key(prefix, userEmail, contactEmail string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", prefix, userEmail, contactEmail))
}

func (s *kvStore) prefixOf(prefix, userEmail string) []byte {
	return []byte(fmt.Sprintf("%s:%s:", prefix, userEmail))
}

func (s *kvStore) Save(prefix, userEmail, contactEmail, name string) error {
	key := s.key(prefix, userEmail, contactEmail)

	if strings.TrimSpace(name) != "" {
		return s.put(key, name)
	}

	// 名称为空：已有记录则保留原名称，否则回退为邮箱前缀。
	if old, err := s.db.Get(key, nil); err == nil && len(old) > 0 {
		return nil
	}

	addName := strings.Split(contactEmail, "@")[0]
	return s.put(key, addName)
}

func (s *kvStore) put(key []byte, name string) error {
	val, err := json.Marshal(record{Name: name})
	if err != nil {
		return err
	}
	return s.db.Put(key, val, 0, nil)
}

func (s *kvStore) Delete(prefix, userEmail, contactEmail string) error {
	return s.db.Del(s.key(prefix, userEmail, contactEmail), nil)
}

func (s *kvStore) Clear(prefix, userEmail string) error {
	prefixAll := s.prefixOf(prefix, userEmail)

	tran, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}
	defer s.db.Discard(tran)

	iter, err := s.db.IterStartWith(prefixAll, tran)
	if err != nil {
		return err
	}
	defer s.db.IterRelease(iter)

	var items []yiyidb.BatItem
	for iter.Seek(prefixAll); iter.Valid(); iter.Next() {
		items = append(items, yiyidb.BatItem{Op: "del", Key: iter.Key()})
	}
	if len(items) == 0 {
		return nil
	}
	if err := s.db.BatPutOrDel(&items, tran); err != nil {
		return err
	}
	return s.db.Commit(tran)
}

func (s *kvStore) List(prefix, userEmail string) ([]Item, error) {
	prefixAll := s.prefixOf(prefix, userEmail)

	tran, err := s.db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	defer s.db.Discard(tran)

	iter, err := s.db.IterStartWith(prefixAll, tran)
	if err != nil {
		return nil, err
	}
	defer s.db.IterRelease(iter)

	var list []Item
	for iter.Seek(prefixAll); iter.Valid(); iter.Next() {
		fullKey := string(iter.Key())
		contactEmail := strings.TrimPrefix(fullKey, string(prefixAll))

		var rec record
		_ = json.Unmarshal(iter.Value(), &rec)
		list = append(list, Item{Email: contactEmail, Name: rec.Name})
	}
	return list, nil
}

func (s *kvStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// MemoryStore 是内存实现，供单元测试使用。
type MemoryStore struct {
	Data map[string]map[string]string // prefix:user -> email -> name
}

// NewMemoryStore 创建内存联系人存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{Data: map[string]map[string]string{}}
}

func (m *MemoryStore) bucket(prefix, userEmail string) map[string]string {
	k := prefix + ":" + userEmail
	if m.Data[k] == nil {
		m.Data[k] = map[string]string{}
	}
	return m.Data[k]
}

// Save 实现 Store。
func (m *MemoryStore) Save(prefix, userEmail, contactEmail, name string) error {
	b := m.bucket(prefix, userEmail)
	if strings.TrimSpace(name) == "" {
		if _, ok := b[contactEmail]; ok {
			return nil
		}
		name = strings.Split(contactEmail, "@")[0]
	}
	b[contactEmail] = name
	return nil
}

// Delete 实现 Store。
func (m *MemoryStore) Delete(prefix, userEmail, contactEmail string) error {
	delete(m.bucket(prefix, userEmail), contactEmail)
	return nil
}

// Clear 实现 Store。
func (m *MemoryStore) Clear(prefix, userEmail string) error {
	m.Data[prefix+":"+userEmail] = map[string]string{}
	return nil
}

// List 实现 Store。
func (m *MemoryStore) List(prefix, userEmail string) ([]Item, error) {
	var list []Item
	for email, name := range m.bucket(prefix, userEmail) {
		list = append(list, Item{Email: email, Name: name})
	}
	return list, nil
}

// Close 实现 Store。
func (m *MemoryStore) Close() error { return nil }
