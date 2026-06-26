package utils

import (
	"crypto/rand"
	"email-server/config"
	"email-server/constant"
	"email-server/model"
	"fmt"
	"math/big"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/emersion/go-imap/client"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// ValidateRequiredParams 校验必填参数
func ValidateRequiredParams(fields []string, obj interface{}) error {
	// 使用反射获取对象的字段值
	v := reflect.ValueOf(obj)

	// 如果是指针，获取其指向的值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 只支持结构体
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("参数结构错误: 只支持Object类型")
	}

	for _, field := range fields {
		// 查找对应字段（首字母大写）
		fieldVal := v.FieldByName(field)

		if !fieldVal.IsValid() {
			return fmt.Errorf("字段 %s 不存在", field)
		}

		// 检查字段值是否为空
		if fieldVal.Kind() == reflect.String {
			if strings.TrimSpace(fieldVal.String()) == "" {
				return fmt.Errorf("%s 参数不能为空", field)
			}
		}
	}
	return nil
}

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
	// 注意：不释放 domain 和 app，因为 account 可能依赖它们的生命周期

	// 成功获取 account，由调用者负责 Release
	return account, nil
}

// FormatFileSize 字节 转为 友好单位 B/KB/MB/GB
func FormatFileSize(size uint32) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(size)/1024/1024)
	} else {
		return fmt.Sprintf("%.1fGB", float64(size)/1024/1024/1024)
	}
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

// GetNameInfo 解析邮箱
func GetNameInfo(mailStr string) (*string, []*model.MailInfo, error) {
	if mailStr == "" {
		return nil, nil, nil
	}

	// 标准库直接解析（自动处理中文、编码、格式）
	addresses, err := mail.ParseAddressList(mailStr)
	if err != nil {
		return nil, nil, err
	}

	mailList := make([]string, 0, len(addresses))
	infoList := make([]*model.MailInfo, 0, len(addresses))

	for _, addr := range addresses {
		mailList = append(mailList, addr.Address)

		name := addr.Name
		if name == "" {
			addname := strings.Split(addr.Address, "@")
			name = addname[0]
		}

		infoList = append(infoList, &model.MailInfo{
			Name:  name,
			Email: addr.Address,
		})
	}
	str := strings.Join(mailList, ", ")

	return &str, infoList, nil
}

// parseMailDate 解析时间，格式化为 2006-01-02 15:04:05
func ParseMailDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("时间为空")
	}

	if t, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
		return t.Local(), nil
	}

	if t, err := time.Parse(time.RFC1123, dateStr); err == nil {
		return t.Local(), nil
	}

	return time.Time{}, fmt.Errorf("不支持的日期格式: %s", dateStr)
}

const shortChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const shortLength = 6

// 生成短链接
func GenerateShortCode() (string, error) {
	b := make([]byte, shortLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(shortChars))))
		if err != nil {
			return "", err
		}
		b[i] = shortChars[n.Int64()]
	}
	return string(b), nil
}
