package utils

import (
	"email-server/config"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"runtime"
	"strings"

	"github.com/emersion/go-imap/client"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// DialIMAPClient 连接IMAP服务器
func DialIMAPClient(email, password string) (*client.Client, error) {
	imapClient, err := client.Dial(fmt.Sprintf("%s:%d", config.HmailHost, config.ImapPort))
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
	// 建立TCP连接
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.HmailHost, config.SmtpPort))
	if err != nil {
		return nil, fmt.Errorf("连接SMTP服务器失败: %w", err)
	}

	// 创建SMTP客户端
	smtpClient, err := smtp.NewClient(conn, config.HmailHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建SMTP客户端失败: %w", err)
	}

	// SMTP认证
	auth := smtp.PlainAuth("", email, password, config.HmailHost)
	if err := smtpClient.Auth(auth); err != nil {
		smtpClient.Quit()
		conn.Close()
		return nil, fmt.Errorf("SMTP认证失败: %w", err)
	}

	return smtpClient, nil
}

// ValidFolder 验证文件夹存在，不存在则创建
func ValidFolder(client *client.Client, folder string) error {
	// 尝试选择文件夹（只读模式）
	_, err := client.Select(folder, true)
	if err == nil {
		return nil
	}
	// 文件夹不存在则创建
	if err := client.Create(folder); err != nil {
		return fmt.Errorf("创建文件夹失败: %v", err)
	}
	return nil
}

// containsFlag 检查标志是否在标志列表中
func ContainsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

// CleanText 清理文本中的特殊字符
func CleanText(text string) string {
	// 去除首尾空白
	text = strings.TrimSpace(text)

	// 将多个连续空白字符（包括换行符、制表符等）替换为单个空格
	reg := regexp.MustCompile(`\s+`)
	text = reg.ReplaceAllString(text, "")

	// 去除 * 号
	text = strings.ReplaceAll(text, "*", "")

	return text
}

// ChangePassword 修改用户密码
func ChangePassword(adminPassword, emailAddress, newPassword string) error {
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

	// 获取 Domains 集合
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	// 自动从邮箱地址中提取域名
	domainName := ""
	if idx := strings.LastIndex(emailAddress, "@"); idx != -1 {
		domainName = emailAddress[idx+1:]
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
	accountItem, err := oleutil.GetProperty(accounts, "ItemByAddress", emailAddress)
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
				if strings.EqualFold(currentAddress, emailAddress) {
					account = currentAccount
					accountFound = true
					break
				}
			}
			currentAccount.Release()
		}

		if !accountFound {
			return fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, emailAddress)
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
