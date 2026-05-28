package service

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"email-server/config"
	"email-server/model"
	"email-server/utils"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/golang-jwt/jwt/v5"
)

// Login 登录
func Login(adminPassword, email, password string) (*model.UserItem, error) {
	// 首先通过 IMAP 验证邮箱和密码是否正确
	imapClient, err := utils.DialIMAPClient(email, password)
	if err != nil {
		return nil, fmt.Errorf("邮箱或密码错误: %w", err)
	}
	defer imapClient.Logout()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err = ole.CoInitialize(0)
	if err != nil {
		return nil, fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return nil, fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return nil, fmt.Errorf("管理员鉴权失败: %v", err)
	}

	// 自动从邮箱地址中提取域名
	domainName := ""
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		domainName = email[idx+1:]
	} else {
		return nil, fmt.Errorf("邮箱地址格式不正确，未找到 '@' 符号")
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return nil, fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	// 遍历 Domains 集合查找匹配的域名
	var domain *ole.IDispatch
	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return nil, fmt.Errorf("获取域名数量失败: %v", err)
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
		return nil, fmt.Errorf("未找到域名 [%s]，请检查是否配置该域名", domainName)
	}
	defer domain.Release()

	// 获取该域名下的 Accounts 集合
	accountsObj, err := oleutil.GetProperty(domain, "Accounts")
	if err != nil {
		return nil, fmt.Errorf("获取 Accounts 属性失败: %v", err)
	}
	accounts := accountsObj.ToIDispatch()
	defer accounts.Release()

	// 通过完整邮箱地址查找对应的 Account 对象
	var account *ole.IDispatch
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err != nil {
		// 如果 ItemByAddress 失败，则遍历所有账号
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			return nil, fmt.Errorf("获取账号数量失败: %v", err)
		}
		accountsCount := int(accountsCountResult.Val)

		accountFound := false
		for j := 0; j < accountsCount; j++ {
			acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
			if err != nil {
				continue
			}
			currentAccount := acctItem.ToIDispatch()

			// 获取账号地址
			addressResult, err := oleutil.GetProperty(currentAccount, "Address")
			if err == nil {
				currentAddress := addressResult.ToString()
				if strings.EqualFold(currentAddress, email) {
					account = currentAccount
					accountFound = true
					break
				}
			}
			currentAccount.Release()
		}

		if !accountFound {
			return nil, fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
		}
	} else {
		account = accountItem.ToIDispatch()
	}
	defer account.Release()

	// 获取用户信息
	idVar, _ := oleutil.GetProperty(account, "Id")
	addressVar, _ := oleutil.GetProperty(account, "Address")
	PersonFirstNameVar, _ := oleutil.GetProperty(account, "PersonFirstName")
	PersonLastNameVar, _ := oleutil.GetProperty(account, "PersonLastName")
	adminVar, _ := oleutil.GetProperty(account, "AdministrationLevel")

	id := idVar.Val
	address := addressVar.ToString()
	name := PersonFirstNameVar.ToString() + PersonLastNameVar.ToString()
	isadmin := adminVar.Val

	// 生成token
	claims := model.UserClaims{
		Email:    email,
		Password: password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * config.JwtExpireHour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.JwtSecretKey))
	if err != nil {
		return nil, fmt.Errorf("生成Token失败: %v", err)
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

// UpdatePassword 修改密码
func UpdatePassword(adminPassword, email, oldPwd, newPassword string) error {
	// // 建立IMAP连接
	imapClient, err := utils.DialIMAPClient(email, oldPwd)
	if err != nil {
		return fmt.Errorf("旧密码验证失败: %w", err)
	}
	imapClient.Logout()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err = ole.CoInitialize(0)
	if err != nil {
		return fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return fmt.Errorf("管理员鉴权失败: %v", err)
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	// 自动从邮箱地址中提取域名
	domainName := ""
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		domainName = email[idx+1:]
	} else {
		return fmt.Errorf("邮箱地址格式不正确，未找到 '@' 符号")
	}

	// 遍历 Domains 集合查找匹配的域名
	var domain *ole.IDispatch
	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return fmt.Errorf("获取域名数量失败: %v", err)
	}
	count := int(countResult.Val)

	found := false

	// hMailServer Domains 集合
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

	// 通过完整邮箱地址查找对应的 Account 对象
	var account *ole.IDispatch
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err != nil {

		// 如果 ItemByAddress 失败，则遍历所有账号
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			return fmt.Errorf("获取账号数量失败: %v", err)
		}
		accountsCount := int(accountsCountResult.Val)

		accountFound := false
		for j := 0; j < accountsCount; j++ {
			// 使用 GetProperty 访问 Item 属性
			acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
			if err != nil {
				continue
			}
			currentAccount := acctItem.ToIDispatch()

			// 获取账号地址
			addressResult, err := oleutil.GetProperty(currentAccount, "Address")
			if err == nil {
				currentAddress := addressResult.ToString()
				if strings.EqualFold(currentAddress, email) {
					account = currentAccount
					accountFound = true
					break
				}
			}
			currentAccount.Release()
		}

		if !accountFound {
			return fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
		}
	} else {
		account = accountItem.ToIDispatch()
	}
	defer account.Release()

	// 设置新密码并保存
	_, err = oleutil.PutProperty(account, "Password", newPassword)
	if err != nil {
		return fmt.Errorf("设置新密码属性失败: %v", err)
	}

	_, err = oleutil.CallMethod(account, "Save")
	if err != nil {
		return fmt.Errorf("保存账号修改失败: %v", err)
	}

	return nil
}

// UserList 获取用户列表
func UserList(adminPassword string) ([]*model.UserItem, int, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err := ole.CoInitialize(0)
	if err != nil {
		return nil, 0, fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return nil, 0, fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, 0, fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return nil, 0, fmt.Errorf("管理员鉴权失败: %v", err)
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return nil, 0, fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	// 获取域名数量
	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return nil, 0, fmt.Errorf("获取域名数量失败: %v", err)
	}
	domainCount := int(countResult.Val)

	var userList []*model.UserItem
	total := 0

	// 遍历所有域名
	for i := 0; i < domainCount; i++ {
		domainItem, err := oleutil.GetProperty(domains, "Item", int32(i))
		if err != nil {
			continue
		}
		domain := domainItem.ToIDispatch()

		// 获取该域名下的 Accounts 集合
		accountsObj, err := oleutil.GetProperty(domain, "Accounts")
		if err != nil {
			domain.Release()
			continue
		}
		accounts := accountsObj.ToIDispatch()

		// 获取账号数量
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			accounts.Release()
			domain.Release()
			continue
		}
		accountsCount := int(accountsCountResult.Val)

		// 遍历该域名下的所有账号
		for j := 0; j < accountsCount; j++ {
			acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
			if err != nil {
				continue
			}
			account := acctItem.ToIDispatch()

			// 提取账号信息
			idVar, _ := oleutil.GetProperty(account, "Id")
			addressVar, _ := oleutil.GetProperty(account, "Address")
			PersonFirstNameVar, _ := oleutil.GetProperty(account, "PersonFirstName")
			PersonLastNameVar, _ := oleutil.GetProperty(account, "PersonLastName")
			adminVar, _ := oleutil.GetProperty(account, "AdministrationLevel")

			id := idVar.Val
			address := addressVar.ToString()
			name := PersonFirstNameVar.ToString() + PersonLastNameVar.ToString()
			isadmin := adminVar.Val

			user := &model.UserItem{
				ID:       id,
				Email:    address,
				FullName: name,
				IsAdmin:  isadmin,
			}

			// 添加到用户列表
			userList = append(userList, user)
			total++

			account.Release()
		}

		accounts.Release()
		domain.Release()
	}

	return userList, total, nil
}

// CreateUser 创建用户
func CreateUser(Folders []string, adminPassword, email, password, firstName, lastName string, isAdmin int64) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err := ole.CoInitialize(0)
	if err != nil {
		return fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return fmt.Errorf("管理员鉴权失败: %v", err)
	}

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

	_, err = oleutil.PutProperty(newAccount, "Active", true)
	if err != nil {
		return fmt.Errorf("设置激活状态失败: %v", err)
	}

	_, err = oleutil.PutProperty(newAccount, "AdministrationLevel", isAdmin)
	if err != nil {
		return fmt.Errorf("设置管理员级别失败: %v", err)
	}

	// 保存账号
	_, err = oleutil.CallMethod(newAccount, "Save")
	if err != nil {
		return fmt.Errorf("保存账号失败: %v", err)
	}

	// 或者在此处建立 IMAP 连接并创建文件夹
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

// DeleteUser 删除用户
func DeleteUser(adminPassword, email string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err := ole.CoInitialize(0)
	if err != nil {
		return fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return fmt.Errorf("管理员鉴权失败: %v", err)
	}

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

	// 通过完整邮箱地址查找对应的 Account 对象
	var account *ole.IDispatch
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err != nil {
		// 如果 ItemByAddress 失败，则遍历所有账号
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			return fmt.Errorf("获取账号数量失败: %v", err)
		}
		accountsCount := int(accountsCountResult.Val)

		accountFound := false
		for j := 0; j < accountsCount; j++ {
			acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
			if err != nil {
				continue
			}
			currentAccount := acctItem.ToIDispatch()

			// 获取账号地址
			addressResult, err := oleutil.GetProperty(currentAccount, "Address")
			if err == nil {
				currentAddress := addressResult.ToString()
				if strings.EqualFold(currentAddress, email) {
					account = currentAccount
					accountFound = true
					break
				}
			}
			currentAccount.Release()
		}

		if !accountFound {
			return fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
		}
	} else {
		account = accountItem.ToIDispatch()
	}
	defer account.Release()

	// 删除账号
	_, err = oleutil.CallMethod(account, "Delete")
	if err != nil {
		return fmt.Errorf("删除账号失败: %v", err)
	}

	return nil
}

// UpdateUser 更新用户
func UpdateUser(adminPassword, email, firstName, lastName string, isAdmin int64) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 初始化 COM 库
	err := ole.CoInitialize(0)
	if err != nil {
		return fmt.Errorf("初始化 COM 失败: %v", err)
	}
	defer ole.CoUninitialize()

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		return fmt.Errorf("管理员鉴权失败: %v", err)
	}

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

	// 通过完整邮箱地址查找对应的 Account 对象
	var account *ole.IDispatch
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err != nil {
		// 如果 ItemByAddress 失败，则遍历所有账号
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			return fmt.Errorf("获取账号数量失败: %v", err)
		}
		accountsCount := int(accountsCountResult.Val)

		accountFound := false
		for j := 0; j < accountsCount; j++ {
			acctItem, err := oleutil.GetProperty(accounts, "Item", int32(j))
			if err != nil {
				continue
			}
			currentAccount := acctItem.ToIDispatch()

			// 获取账号地址
			addressResult, err := oleutil.GetProperty(currentAccount, "Address")
			if err == nil {
				currentAddress := addressResult.ToString()
				if strings.EqualFold(currentAddress, email) {
					account = currentAccount
					accountFound = true
					break
				}
			}
			currentAccount.Release()
		}

		if !accountFound {
			return fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
		}
	} else {
		account = accountItem.ToIDispatch()
	}
	defer account.Release()

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
		_, err = oleutil.PutProperty(account, "AdministrationLevel", isAdmin)
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
