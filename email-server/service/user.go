package service

import (
	"email-server/constant"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gogf/gf/v2/util/gconv"

	"email-server/config"
	"email-server/model"
	"email-server/utils"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/golang-jwt/jwt/v5"
)

// Login 登录
func Login(adminPassword, email, password string, Folders []string) (*model.UserItem, error) {
	// 首先通过 IMAP 验证邮箱和密码是否正确
	imapClient, err := utils.DialIMAPClient(email, password)
	if err != nil {
		return nil, err
	}
	defer imapClient.Logout()

	// 获取 hMailServer 账号对象
	account, err := utils.GetHmailAccount(adminPassword, email)
	if err != nil {
		return nil, err
	}
	// 这里只需要释放 account 对象和 COM 资源
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 获取用户信息
	idVar, err := oleutil.GetProperty(account, "Id")
	addressVar, err := oleutil.GetProperty(account, "Address")
	PersonFirstNameVar, err := oleutil.GetProperty(account, "PersonFirstName")
	PersonLastNameVar, err := oleutil.GetProperty(account, "PersonLastName")
	adminVar, err := oleutil.GetProperty(account, "Adminlevel")

	id := idVar.Val
	address := addressVar.ToString()
	name := PersonFirstNameVar.ToString() + PersonLastNameVar.ToString()
	isadmin := adminVar.Val

	var tokenLifeSpan int = 0
	tokenLifeSpan = gconv.Int(config.GetConfig((constant.JwtLifeWebSpan)))

	// 生成token
	claims := model.UserClaims{
		Email:    email,
		Password: password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(tokenLifeSpan) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.GetConfig(constant.JwtSecret)))
	if err != nil {
		return nil, fmt.Errorf("生成Token失败: %v", err)
	}

	// 验证文件夹是否存在，不存在就创建
	for _, folder := range Folders {
		// 跳过 INBOX，因为它已经自动创建
		if strings.ToUpper(folder) == "INBOX" {
			continue
		}

		// 使用 IMAP CREATE 命令创建文件夹
		createErr := imapClient.Create(folder)
		if createErr != nil {
			// 记录错误但不中断流程，因为文件夹可能已存在
		}
	}

	user := &model.UserItem{
		ID:       id,
		Email:    address,
		FullName: name,
		IsAdmin:  isadmin,
		Token:    tokenStr,
	}

	// 返回用户信息
	return user, nil
}

// CreateUser 创建用户
func CreateUser(Folders []string, adminPassword, email, password, firstName, lastName string, isAdmin int64) error {
	// 获取 hMailServer Application 对象
	app, err := utils.InitHmailApp(adminPassword)
	if err != nil {
		return err
	}
	defer func() {
		app.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 自动从邮箱地址中提取域名
	domainName := ""
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		domainName = email[idx+1:]
	} else {
		return fmt.Errorf("邮箱地址格式不正确，未找到 '@' 符号")
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	// 遍历 Domains 集合查找匹配的域名
	var domain *ole.IDispatch
	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return fmt.Errorf("获取域名数量失败: %v", err)
	}
	count := int(countResult.Val)

	found := false

	for i := 0; i < count; i++ {
		itemResult, err := oleutil.GetProperty(domains, "Item", int32(i))
		if err != nil {
			continue
		}

		currentDomain := itemResult.ToIDispatch()

		// 获取域名名称
		nameResult, err := oleutil.GetProperty(currentDomain, "Name")
		if err == nil {
			currentName := nameResult.ToString()
			if strings.EqualFold(currentName, domainName) {
				domain = currentDomain
				found = true
				break
			}
		}
		currentDomain.Release()
	}

	if !found {
		return fmt.Errorf("未找到域名 [%s]，请检查是否配置该域名", domainName)
	}
	defer domain.Release()

	// 获取该域名下的 Accounts 集合
	accountsObj, err := oleutil.GetProperty(domain, "Accounts")
	if err != nil {
		return fmt.Errorf("获取 Accounts 属性失败: %v", err)
	}
	accounts := accountsObj.ToIDispatch()
	defer accounts.Release()

	// 检查邮箱地址是否已存在
	_, err = oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err == nil {
		return fmt.Errorf("邮箱地址 [%s] 已存在", email)
	}

	// 创建新账号
	newAccountObj, err := oleutil.CallMethod(accounts, "Add")
	if err != nil {
		return fmt.Errorf("创建账号失败: %v", err)
	}
	newAccount := newAccountObj.ToIDispatch()
	defer newAccount.Release()

	// 设置账号属性
	_, err = oleutil.PutProperty(newAccount, "Address", email)
	if err != nil {
		return fmt.Errorf("设置邮箱地址失败: %v", err)
	}

	_, err = oleutil.PutProperty(newAccount, "Password", password)
	if err != nil {
		return fmt.Errorf("设置密码失败: %v", err)
	}

	// 设置名字和姓氏（可选）
	if firstName != "" {
		_, err = oleutil.PutProperty(newAccount, "PersonFirstName", firstName)
		if err != nil {
			return fmt.Errorf("设置名字失败: %v", err)
		}
	}

	if lastName != "" {
		_, err = oleutil.PutProperty(newAccount, "PersonLastName", lastName)
		if err != nil {
			return fmt.Errorf("设置姓氏失败: %v", err)
		}
	}

	// 设置激活状态
	_, err = oleutil.PutProperty(newAccount, "Active", true)
	if err != nil {
		return fmt.Errorf("设置激活状态失败: %v", err)
	}

	// 设置管理员级别
	_, err = oleutil.PutProperty(newAccount, "Adminlevel", isAdmin)
	if err != nil {
		return fmt.Errorf("设置管理员级别失败: %v", err)
	}

	// 保存账号
	_, err = oleutil.CallMethod(newAccount, "Save")
	if err != nil {
		return fmt.Errorf("保存账号失败: %v", err)
	}

	// 建立 IMAP 连接并创建文件夹
	if len(Folders) > 0 {
		// 等待账户完全创建，确保可以成功建立 IMAP 连接
		imapClient, imapErr := utils.DialIMAPClient(email, password)
		if imapErr == nil {
			defer imapClient.Logout()

			// 创建指定的文件夹
			for _, folder := range Folders {
				// 跳过 INBOX，因为它已经自动创建
				if strings.ToUpper(folder) == "INBOX" {
					continue
				}

				// 使用 IMAP CREATE 命令创建文件夹
				createErr := imapClient.Create(folder)
				if createErr != nil {
					// 记录错误但不中断流程，因为文件夹可能已存在
					fmt.Printf("警告: 创建文件夹 [%s] 失败: %v\n", folder, createErr)
				}
			}
		} else {
			// IMAP 连接失败不影响账户创建，只记录警告
			fmt.Printf("警告: 无法建立 IMAP 连接以创建文件夹: %v\n", imapErr)
		}
	}

	return nil
}

// UserList 获取用户列表
func UserList(adminPassword, email string) ([]*model.UserList, int64, error) {
	// 获取 hMailServer Application 对象
	app, err := utils.InitHmailApp(adminPassword)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		app.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 从入参email提取目标域名
	var targetDomain string
	if idx := strings.LastIndex(email, "@"); idx == -1 {
		return nil, 0, fmt.Errorf("邮箱格式非法，缺少@符号")
	} else {
		targetDomain = email[idx+1:]
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return nil, 0, fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return nil, 0, fmt.Errorf("获取域名数量失败: %v", err)
	}
	domainCount := int(countResult.Val)

	var userList []*model.UserList
	var total int64
	var targetDomainObj *ole.IDispatch
	var accountList []string

	// 只查找匹配的目标域名
	for i := 0; i < domainCount; i++ {
		domainItem, err := oleutil.GetProperty(domains, "Item", int32(i))
		if err != nil {
			continue
		}
		domain := domainItem.ToIDispatch()

		nameResult, err := oleutil.GetProperty(domain, "Name")
		if err == nil {
			domainName := nameResult.ToString()
			// 忽略大小写匹配域名
			if strings.EqualFold(domainName, targetDomain) {
				targetDomainObj = domain
				break
			}
		}
		domain.Release()
	}

	// 未找到对应域名直接返回
	if targetDomainObj == nil {
		return nil, 0, fmt.Errorf("未找到域名: %s", targetDomain)
	}
	defer targetDomainObj.Release()

	// 仅遍历当前域名下的账号
	accountsObj, err := oleutil.GetProperty(targetDomainObj, "Accounts")
	if err != nil {
		return nil, 0, fmt.Errorf("获取 Accounts 属性失败: %v", err)
	}
	accounts := accountsObj.ToIDispatch()
	defer accounts.Release()

	accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
	if err != nil {
		return nil, 0, fmt.Errorf("获取账号数量失败: %v", err)
	}
	accountsCount := int(accountsCountResult.Val)

	// 遍历账号
	for j := 0; j < accountsCount; j++ {
		acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
		if err != nil {
			continue
		}
		account := acctItem.ToIDispatch()

		idVar, _ := oleutil.GetProperty(account, "Id")
		addressVar, _ := oleutil.GetProperty(account, "Address")
		PersonFirstNameVar, _ := oleutil.GetProperty(account, "PersonFirstName")
		PersonLastNameVar, _ := oleutil.GetProperty(account, "PersonLastName")
		adminVar, _ := oleutil.GetProperty(account, "Adminlevel")
		lastLogonTimeVar, _ := oleutil.GetProperty(account, "LastLogonTime")

		id := idVar.Val
		address := addressVar.ToString()
		firstNameVar := PersonFirstNameVar.ToString()
		lastNameVar := PersonLastNameVar.ToString()
		isadmin := adminVar.Val

		lastLogonTime, _ := ole.GetVariantDate(uint64(lastLogonTimeVar.Val))
		loginTimeStr := lastLogonTime.Format("2006-01-02 15:04:05")

		user := &model.UserList{
			ID:              id,
			Email:           address,
			FullName:        firstNameVar + lastNameVar,
			PersonFirstName: firstNameVar,
			PersonLastName:  lastNameVar,
			IsAdmin:         isadmin,
			LastLogonTime:   loginTimeStr,
		}
		accountList = append(accountList, address)
		userList = append(userList, user)
		total++
		account.Release()
	}

	// 校验是否包含头像
	_ = utils.HasAvatar(accountList)

	return userList, total, nil
}

// DeleteUser 删除用户
func DeleteUser(adminPassword, email string) error {
	// 获取 hMailServer 账号对象
	account, err := utils.GetHmailAccount(adminPassword, email)
	if err != nil {
		return err
	}
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 删除账号
	_, err = oleutil.CallMethod(account, "Delete")
	if err != nil {
		return fmt.Errorf("删除账号失败: %v", err)
	}

	return nil
}

// UpdateUser 更新用户
func UpdateUser(adminPassword, email, firstName, lastName string, isAdmin int64) error {
	// 获取 hMailServer 账号对象
	account, err := utils.GetHmailAccount(adminPassword, email)
	if err != nil {
		return err
	}
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 更新名字和姓氏（可选）
	if firstName != "" {
		_, err = oleutil.PutProperty(account, "PersonFirstName", firstName)
		if err != nil {
			return fmt.Errorf("设置名字失败: %v", err)
		}
	}

	if lastName != "" {
		_, err = oleutil.PutProperty(account, "PersonLastName", lastName)
		if err != nil {
			return fmt.Errorf("设置姓氏失败: %v", err)
		}
	}

	if isAdmin != -1 {
		_, err = oleutil.PutProperty(account, "Adminlevel", isAdmin)
		if err != nil {
			return fmt.Errorf("设置管理员级别失败: %v", err)
		}
	}

	// 保存账号修改
	_, err = oleutil.CallMethod(account, "Save")
	if err != nil {
		return fmt.Errorf("保存账号修改失败: %v", err)
	}

	return nil
}

// UpdatePassword 修改密码
func UpdatePassword(adminPassword, email, oldPwd, newPassword string) error {
	// // 建立IMAP连接
	imapClient, err := utils.DialIMAPClient(email, oldPwd)
	if err != nil {
		return fmt.Errorf("旧密码验证失败: %w", err)
	}
	imapClient.Logout()

	// 获取 hMailServer 账号对象
	account, err := utils.GetHmailAccount(adminPassword, email)
	if err != nil {
		return err
	}
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 设置新密码
	_, err = oleutil.PutProperty(account, "Password", newPassword)
	if err != nil {
		return fmt.Errorf("设置新密码属性失败: %v", err)
	}

	// 保存账号修改
	_, err = oleutil.CallMethod(account, "Save")
	if err != nil {
		return fmt.Errorf("保存账号修改失败: %v", err)
	}

	return nil
}

// ContactList 联系人列表
func ContactList(prefix, email string) ([]*model.Contact, int64, error) {
	contactEmails, err := utils.ListUserContacts(prefix, email)
	if err != nil {
		return nil, 0, fmt.Errorf("读取联系人失败: %w", err)
	}

	var contactList []*model.Contact
	for _, c := range contactEmails {
		item := &model.Contact{
			Email: c.Email,
			Name:  c.Name,
		}
		contactList = append(contactList, item)
	}
	total := int64(len(contactList))

	return contactList, total, nil
}

// SaveContact 保存联系人
func SaveContact(prefix, email, to, name string) error {
	// 不能自己保存自己
	if email == to {
		return nil
	}
	return utils.SaveSentContact(prefix, email, to, name)
}

// DeleteContact 删除单个联系人
func DeleteContact(prefix, email string, to string) error {
	return utils.DelContact(prefix, email, to)
}

// ClearContact 清空所有联系人
func ClearContact(prefix, email string) error {
	return utils.ClearAllContact(prefix, email)
}

// UploadAvatar 上传头像
func UploadAvatar(email string, avatarFile *multipart.FileHeader) error {
	// 打开上传的文件
	src, err := avatarFile.Open()
	if err != nil {
		return fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	// 读取文件内容
	fileContent, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 验证文件大小（限制为 5MB）
	if len(fileContent) > 5*1024*1024 {
		return fmt.Errorf("头像文件大小不能超过 5MB")
	}

	// 验证文件类型（检查是否为图片）
	contentType := http.DetectContentType(fileContent)
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("上传的文件不是有效的图片格式")
	}
	// 生成本地文件路径
	fileName := fmt.Sprintf("%s%s", email, ".webp")

	// 保存到静态资源目录
	staticDir := filepath.Join("static", "avatars")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return fmt.Errorf("创建静态目录失败: %v", err)
	}

	localPath := filepath.Join(staticDir, fileName)

	// 保存文件
	if err := os.WriteFile(localPath, fileContent, 0644); err != nil {
		return fmt.Errorf("保存头像文件失败: %v", err)
	}
	return nil
}
