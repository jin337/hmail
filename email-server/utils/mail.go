package utils

import (
	"bytes"
	"email-server/config"
	"email-server/constant"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"runtime"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// DialIMAPClient 连接IMAP服务器
func DialIMAPClient(email, password string) (*client.Client, error) {
	host := config.GetConfig(constant.MailServerHost)
	port := config.GetConfig(constant.ImapPort)
	// 使用 net.JoinHostPort 处理 IPv4/IPv6 地址
	address := net.JoinHostPort(host, port)
	imapClient, err := client.Dial(address)
	if err != nil {
		return nil, fmt.Errorf("连接IMAP失败: %v", err)
	}

	if err := imapClient.Login(email, password); err != nil {
		imapClient.Logout()
		return nil, fmt.Errorf("用户名或密码错误")
	}

	return imapClient, nil
}

// DialSMTPClient 连接SMTP服务器
func DialSMTPClient(email, password string) (*smtp.Client, error) {
	host := config.GetConfig(constant.MailServerHost)
	port := config.GetConfig(constant.SmtpPort)
	// 建立TCP连接 - 使用 net.JoinHostPort 处理 IPv4/IPv6 地址
	address := net.JoinHostPort(host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("连接SMTP服务器失败: %w", err)
	}

	// 创建SMTP客户端
	smtpClient, err := smtp.NewClient(conn, config.GetConfig(constant.MailServerHost))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建SMTP客户端失败: %w", err)
	}

	// SMTP认证
	auth := smtp.PlainAuth("", email, password, config.GetConfig(constant.MailServerHost))
	if err := smtpClient.Auth(auth); err != nil {
		smtpClient.Quit()
		conn.Close()
		return nil, fmt.Errorf("SMTP认证失败: %w", err)
	}

	return smtpClient, nil
}

// InitHmailApp 初始化 hMailServer Application 并进行管理员鉴权
func InitHmailApp(adminPassword string) (*ole.IDispatch, error) {
	runtime.LockOSThread()

	// 初始化 COM 库
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_DISABLE_OLE1DDE)
	if err != nil {
		if oleErr, ok := err.(*ole.OleError); !ok || oleErr.Code() != 0x00000001 {
			runtime.UnlockOSThread()
			return nil, fmt.Errorf("初始化 COM 失败: %v", err)
		}
	}

	// 创建 hMailServer.Application 对象
	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		ole.CoUninitialize()
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("创建对象失败: %v", err)
	}

	// 获取 IDispatch 接口
	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		unknown.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("获取接口失败: %v", err)
	}

	// 使用管理员账号鉴权
	_, err = oleutil.CallMethod(app, "Authenticate", "Administrator", adminPassword)
	if err != nil {
		app.Release()
		unknown.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("管理员鉴权失败: %v", err)
	}

	// 注意：unknown 已经通过 QueryInterface 转移到 app，不需要单独 Release
	return app, nil
}

// GetHmailAccount 获取 hMailServer 中的账号对象
func GetHmailAccount(adminPassword, email string) (*ole.IDispatch, error) {
	app, err := InitHmailApp(adminPassword)
	if err != nil {
		return nil, err
	}
	// 注意：不能在这里 defer Release app，因为 account 对象依赖 app 的生命周期
	// app 的生命周期由调用者管理

	// 自动从邮箱地址中提取域名
	domainName := ""
	if idx := strings.LastIndex(email, "@"); idx != -1 {
		domainName = email[idx+1:]
	} else {
		app.Release() // 错误时手动释放
		return nil, fmt.Errorf("邮箱地址格式不正确，未找到 '@' 符号")
	}

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		app.Release()
		return nil, fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	// 注意：不释放 domains，因为后续需要查找 domain

	// 遍历 Domains 集合查找匹配的域名
	var domain *ole.IDispatch
	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		domains.Release()
		app.Release()
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
		domains.Release()
		app.Release()
		return nil, fmt.Errorf("未找到域名 [%s]，请检查是否配置该域名", domainName)
	}
	// 找到 domain 后，释放 domains 集合（domain 对象是独立的）
	domains.Release()

	// 获取该域名下的 Accounts 集合
	accountsObj, err := oleutil.GetProperty(domain, "Accounts")
	if err != nil {
		domain.Release()
		app.Release()
		return nil, fmt.Errorf("获取 Accounts 属性失败: %v", err)
	}
	accounts := accountsObj.ToIDispatch()

	// 通过完整邮箱地址查找对应的 Account 对象
	var account *ole.IDispatch
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", email)
	if err != nil {
		// 如果 ItemByAddress 失败，则遍历所有账号
		accountsCountResult, err := oleutil.GetProperty(accounts, "Count")
		if err != nil {
			accounts.Release()
			domain.Release()
			app.Release()
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
			accounts.Release()
			domain.Release()
			app.Release()
			return nil, fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
		}
	} else {
		account = accountItem.ToIDispatch()
	}

	// 找到 account 后，释放 accounts 集合（account 对象是独立的）
	accounts.Release()

	// 成功获取 account，由调用者负责 Release
	return account, nil
}

// FormatMailAddr 格式化邮箱地址，显示为 "姓名 <email>" 的形式
func FormatMailAddr(adminPwd, email string) string {
	if email == "" {
		return ""
	}
	account, err := GetHmailAccount(adminPwd, email)
	if err != nil {
		return email // 获取账号失败，返回原始邮箱地址
	}
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	PersonFirstNameVar, err := oleutil.GetProperty(account, "PersonFirstName")
	PersonLastNameVar, err := oleutil.GetProperty(account, "PersonLastName")
	name := PersonFirstNameVar.ToString() + PersonLastNameVar.ToString()

	encodedName := mime.BEncoding.Encode("utf-8", name)
	return fmt.Sprintf(`%s <%s>`, encodedName, email)
}

// GetMailName 获取邮箱对应的用户显示名称与邮箱
func GetMailName(adminPwd, email string) (string, string, error) {
	if email == "" {
		return "", "", fmt.Errorf("邮箱不能为空")
	}

	// 获取账号COM对象
	account, err := GetHmailAccount(adminPwd, email)
	if err != nil {
		return "", email, fmt.Errorf("当前域名下此账号不存在")
	}

	// COM资源统一释放
	defer func() {
		account.Release()
		ole.CoUninitialize()
		runtime.UnlockOSThread()
	}()

	// 分别获取名、姓，分开捕获错误，不覆盖
	firstNameVar, err1 := oleutil.GetProperty(account, "PersonFirstName")
	lastNameVar, err2 := oleutil.GetProperty(account, "PersonLastName")
	if err1 != nil {
		return "", email, fmt.Errorf("读取FirstName失败: %w", err1)
	}
	if err2 != nil {
		return "", email, fmt.Errorf("读取LastName失败: %w", err2)
	}

	firstName := firstNameVar.ToString()
	lastName := lastNameVar.ToString()
	fullName := strings.TrimSpace(firstName + lastName)

	// 姓名为空直接返回原邮箱，不做编码
	if fullName == "" {
		return "", email, nil
	}

	return fullName, email, nil
}

// GetExtractHeader 获取邮件头信息
func GetExtractHeader(raw []byte, headerKey string) string {
	reader := bytes.NewReader(raw)
	mailReader, err := mail.CreateReader(reader)
	if err != nil {
		return ""
	}
	defer mailReader.Close()
	return mailReader.Header.Get(headerKey)
}

// GetUid 定时发送后将草稿移动到已发送
func GetUid(email, pwd, messageID, folder string) (int64, error) {
	imapClient, err := DialIMAPClient(email, pwd)
	if err != nil {
		return 0, fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer imapClient.Logout()

	// 选择草稿箱
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return 0, fmt.Errorf("选择文件夹失败: %w", err)
	}

	// 搜索包含该 Message-ID 的邮件
	searchCrit := &imap.SearchCriteria{}
	searchCrit.Header = make(map[string][]string)
	searchCrit.Header.Add("Message-Id", messageID)

	ids, err := imapClient.UidSearch(searchCrit)
	if err != nil {
		return 0, fmt.Errorf("搜索邮件失败: %w", err)
	}

	if len(ids) == 0 {
		return 0, fmt.Errorf("未找到对应的邮件")
	}

	return int64(ids[0]), nil
}

// GetMailRawByUID 获取邮件原始数据
func GetMailRawByUID(email, pwd, folder string, uid int64) ([]byte, error) {
	imapClient, err := DialIMAPClient(email, pwd)
	if err != nil {
		return nil, fmt.Errorf("连接IMAP服务器失败: %w", err)
	}
	defer imapClient.Logout()

	// 选择文件夹
	_, err = imapClient.Select(folder, false)
	if err != nil {
		return nil, fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)
	}

	uidSet := new(imap.SeqSet)
	uidSet.AddNum(uint32(uid))

	bodyMail := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- imapClient.UidFetch(uidSet, []imap.FetchItem{
			imap.FetchRFC822,
		}, bodyMail)
	}()

	// 从channel中获取邮件消息
	msg, ok := <-bodyMail
	if !ok {
		if err := <-done; err != nil {
			return nil, fmt.Errorf("获取邮件失败: %w", err)
		}
		return nil, fmt.Errorf("未找到邮件 UID: %d", uid)
	}

	section := &imap.BodySectionName{}
	r := msg.GetBody(section)
	if r == nil {
		return nil, fmt.Errorf("无法获取邮件内容")
	}

	// 读取完整的原始邮件数据
	rawData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取邮件内容失败: %w", err)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("获取邮件完成状态失败: %w", err)
	}

	return rawData, nil
}

