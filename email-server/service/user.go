package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"email-server/config"
	"email-server/contacts"
	"email-server/directory"
	"email-server/model"
	"email-server/utils"

	"github.com/golang-jwt/jwt/v5"
)

// Login 校验邮箱密码并签发 JWT，同时确保标准文件夹存在。
func (s *Service) Login(email, password string) (*model.UserItem, error) {
	client, err := s.imap.Connect(email, password)
	if err != nil {
		return nil, err
	}
	defer client.Logout()

	for _, folder := range config.DefaultFolders {
		if strings.EqualFold(folder, config.FolderInbox) {
			continue // INBOX 由服务器自动创建
		}
		_ = client.CreateFolder(folder) // 已存在时忽略错误
	}

	account, err := s.dir.Get(email)
	if err != nil {
		// 非 Windows 等不支持账号目录的环境下仍允许登录，只是缺少显示名等信息。
		if !errors.Is(err, directory.ErrUnsupported) {
			return nil, err
		}
		account = &directory.Account{Address: email}
	}

	token, err := s.signToken(email, password)
	if err != nil {
		return nil, err
	}

	_ = utils.EnsureAvatar(s.cfg.StaticDir, []string{email})

	return &model.UserItem{
		ID:       account.ID,
		Email:    account.Address,
		FullName: account.FullName(),
		IsAdmin:  account.AdminLevel,
		Token:    token,
	}, nil
}

// signToken 生成与旧实现同构的 JWT（载荷含明文邮箱与密码）。
func (s *Service) signToken(email, password string) (string, error) {
	claims := model.UserClaims{
		Email:    email,
		Password: password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.cfg.JWT.LifeHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("生成Token失败: %v", err)
	}
	return str, nil
}

// ChangePassword 校验旧密码后更新账号密码。
func (s *Service) ChangePassword(email, oldPwd, newPwd string) error {
	client, err := s.imap.Connect(email, oldPwd)
	if err != nil {
		return fmt.Errorf("旧密码验证失败: %w", err)
	}
	client.Logout()

	return s.dir.SetPassword(email, newPwd)
}

// UserList 返回同域名下的账号列表。
func (s *Service) UserList(email string) ([]*model.UserList, int64, error) {
	domain, err := domainOf(email)
	if err != nil {
		return nil, 0, err
	}

	accounts, err := s.dir.List(domain)
	if err != nil {
		return nil, 0, err
	}

	list := make([]*model.UserList, 0, len(accounts))
	for _, a := range accounts {
		list = append(list, &model.UserList{
			ID:              a.ID,
			Email:           a.Address,
			FullName:        a.FullName(),
			PersonFirstName: a.FirstName,
			PersonLastName:  a.LastName,
			IsAdmin:         a.AdminLevel,
			LastLogonTime:   utils.FormatDateTime(a.LastLogon),
		})
	}
	return list, int64(len(list)), nil
}

// CreateUser 创建邮箱账号并尽力创建标准文件夹。
func (s *Service) CreateUser(email, password, firstName, lastName string, isAdmin int64) error {
	if err := s.dir.Create(email, password, firstName, lastName, isAdmin); err != nil {
		return err
	}

	client, err := s.imap.Connect(email, password)
	if err != nil {
		fmt.Printf("警告: 无法建立 IMAP 连接以创建文件夹: %v\n", err)
		return nil
	}
	defer client.Logout()

	for _, folder := range config.DefaultFolders {
		if strings.EqualFold(folder, config.FolderInbox) {
			continue
		}
		if err := client.CreateFolder(folder); err != nil {
			fmt.Printf("警告: 创建文件夹 [%s] 失败: %v\n", folder, err)
		}
	}
	return nil
}

// DeleteUser 删除账号。
func (s *Service) DeleteUser(email string) error {
	return s.dir.Delete(email)
}

// UpdateUser 更新账号姓名与管理员级别。
func (s *Service) UpdateUser(email, firstName, lastName string, isAdmin int64) error {
	return s.dir.Update(email, firstName, lastName, isAdmin)
}

// ContactList 返回联系人列表。
func (s *Service) ContactList(prefix, email string) ([]*model.Contact, int64, error) {
	items, err := s.contacts.List(prefix, email)
	if err != nil {
		return nil, 0, fmt.Errorf("读取联系人失败: %w", err)
	}

	list := make([]*model.Contact, 0, len(items))
	for _, item := range items {
		list = append(list, &model.Contact{Email: item.Email, Name: item.Name})
	}
	return list, int64(len(list)), nil
}

// SaveContact 保存联系人；自己给自己不记录。
func (s *Service) SaveContact(prefix, email, to, name string) error {
	if strings.EqualFold(email, to) {
		return nil
	}
	return s.contacts.Save(prefix, email, to, name)
}

// DeleteContact 删除联系人。
func (s *Service) DeleteContact(prefix, email, to string) error {
	return s.contacts.Delete(prefix, email, to)
}

// ClearContact 清空联系人。
func (s *Service) ClearContact(prefix, email string) error {
	return s.contacts.Clear(prefix, email)
}

// 头像大小上限 5MB。
const maxAvatarBytes = 5 * 1024 * 1024

// UploadAvatar 校验并保存头像文件。
func (s *Service) UploadAvatar(email string, file *multipart.FileHeader) error {
	if file == nil {
		return fmt.Errorf("未获取到头像文件")
	}

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}
	if len(content) > maxAvatarBytes {
		return fmt.Errorf("头像文件大小不能超过 5MB")
	}
	if ct := http.DetectContentType(content); !strings.HasPrefix(ct, "image/") {
		return fmt.Errorf("上传的文件不是有效的图片格式")
	}

	if _, err := utils.SaveAvatar(s.cfg.StaticDir, email, content); err != nil {
		return err
	}
	return nil
}

// Contacts 前缀导出，便于 controller 不必直接依赖 contacts 包。
const (
	ContactPrefixSent    = contacts.PrefixSent
	ContactPrefixContact = contacts.PrefixContact
)
