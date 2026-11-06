//go:build windows

package directory

import (
	"fmt"
	"mime"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// hmailDirectory 通过 COM 访问本机 hMailServer。
type hmailDirectory struct {
	adminPassword string
}

// New 返回 Windows 下的 hMailServer 账号目录实现。
func New(adminPassword string) Directory {
	return &hmailDirectory{adminPassword: adminPassword}
}

// withApp 初始化 COM 与 Application 对象，执行 fn 后统一释放资源。
// COM 要求调用协程固定在线程上，因此这里 LockOSThread。
func (d *hmailDirectory) withApp(fn func(app *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_DISABLE_OLE1DDE); err != nil {
		if oleErr, ok := err.(*ole.OleError); !ok || oleErr.Code() != 0x00000001 {
			return fmt.Errorf("初始化 COM 失败: %v", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("hMailServer.Application")
	if err != nil {
		return fmt.Errorf("创建对象失败: %v", err)
	}
	defer unknown.Release()

	app, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("获取接口失败: %v", err)
	}
	defer app.Release()

	if _, err := oleutil.CallMethod(app, "Authenticate", "Administrator", d.adminPassword); err != nil {
		return fmt.Errorf("管理员鉴权失败: %v", err)
	}

	return fn(app)
}

// withAccount 打开 Application 并定位账号对象。
func (d *hmailDirectory) withAccount(email string, fn func(account *ole.IDispatch) error) error {
	return d.withApp(func(app *ole.IDispatch) error {
		account, err := d.account(app, email)
		if err != nil {
			return err
		}
		defer account.Release()
		return fn(account)
	})
}

// domainOf 从邮箱地址提取域名。
func domainOf(email string) (string, error) {
	idx := strings.LastIndex(email, "@")
	if idx < 0 || idx == len(email)-1 {
		return "", fmt.Errorf("邮箱地址格式不正确，未找到 '@' 符号")
	}
	return email[idx+1:], nil
}

// findDomain 在 Domains 集合中按名称（忽略大小写）查找域名。
// 返回的对象由调用方释放。
func findDomain(app *ole.IDispatch, domainName string) (*ole.IDispatch, error) {
	domainsObj, err := oleutil.GetProperty(app, "Domains")
	if err != nil {
		return nil, fmt.Errorf("获取 Domains 属性失败: %v", err)
	}
	domains := domainsObj.ToIDispatch()
	defer domains.Release()

	countResult, err := oleutil.GetProperty(domains, "Count")
	if err != nil {
		return nil, fmt.Errorf("获取域名数量失败: %v", err)
	}
	count := int(countResult.Val)

	for i := 0; i < count; i++ {
		itemResult, err := oleutil.GetProperty(domains, "Item", int32(i))
		if err != nil {
			continue
		}
		current := itemResult.ToIDispatch()

		nameResult, err := oleutil.GetProperty(current, "Name")
		if err == nil && strings.EqualFold(nameResult.ToString(), domainName) {
			return current, nil
		}
		current.Release()
	}

	return nil, fmt.Errorf("未找到域名 [%s]，请检查是否配置该域名", domainName)
}

// accountsOf 取得域名下的 Accounts 集合，由调用方释放。
func accountsOf(domain *ole.IDispatch) (*ole.IDispatch, error) {
	accountsObj, err := oleutil.GetProperty(domain, "Accounts")
	if err != nil {
		return nil, fmt.Errorf("获取 Accounts 属性失败: %v", err)
	}
	return accountsObj.ToIDispatch(), nil
}

// account 定位账号对象，返回的对象由调用方释放。
func (d *hmailDirectory) account(app *ole.IDispatch, email string) (*ole.IDispatch, error) {
	domainName, err := domainOf(email)
	if err != nil {
		return nil, err
	}

	domain, err := findDomain(app, domainName)
	if err != nil {
		return nil, err
	}
	defer domain.Release()

	accounts, err := accountsOf(domain)
	if err != nil {
		return nil, err
	}
	defer accounts.Release()

	if item, err := oleutil.GetProperty(accounts, "ItemByAddress", email); err == nil {
		return item.ToIDispatch(), nil
	}

	countResult, err := oleutil.GetProperty(accounts, "Count")
	if err != nil {
		return nil, fmt.Errorf("获取账号数量失败: %v", err)
	}
	count := int(countResult.Val)

	for i := 0; i < count; i++ {
		item, err := oleutil.GetProperty(accounts, "Item", int32(i))
		if err != nil {
			continue
		}
		acc := item.ToIDispatch()
		addrResult, err := oleutil.GetProperty(acc, "Address")
		if err == nil && strings.EqualFold(addrResult.ToString(), email) {
			return acc, nil
		}
		acc.Release()
	}

	return nil, fmt.Errorf("在域名 [%s] 下未找到邮箱账号 [%s]，请检查该账号是否存在", domainName, email)
}

// readAccount 读取账号属性，缺失的属性按零值处理（与原实现一致）。
func readAccount(acc *ole.IDispatch) Account {
	var a Account
	if v, err := oleutil.GetProperty(acc, "Id"); err == nil && v != nil {
		a.ID = v.Val
	}
	if v, err := oleutil.GetProperty(acc, "Address"); err == nil && v != nil {
		a.Address = v.ToString()
	}
	if v, err := oleutil.GetProperty(acc, "PersonFirstName"); err == nil && v != nil {
		a.FirstName = v.ToString()
	}
	if v, err := oleutil.GetProperty(acc, "PersonLastName"); err == nil && v != nil {
		a.LastName = v.ToString()
	}
	if v, err := oleutil.GetProperty(acc, "Adminlevel"); err == nil && v != nil {
		a.AdminLevel = v.Val
	}
	if v, err := oleutil.GetProperty(acc, "LastLogonTime"); err == nil && v != nil {
		if t, convErr := ole.GetVariantDate(uint64(v.Val)); convErr == nil {
			a.LastLogon = t
		}
	}
	return a
}

// Get 实现 Directory。
func (d *hmailDirectory) Get(email string) (*Account, error) {
	var account Account
	err := d.withAccount(email, func(acc *ole.IDispatch) error {
		account = readAccount(acc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// List 实现 Directory。
func (d *hmailDirectory) List(domainName string) ([]Account, error) {
	var accounts []Account
	err := d.withApp(func(app *ole.IDispatch) error {
		domain, err := findDomain(app, domainName)
		if err != nil {
			return err
		}
		defer domain.Release()

		coll, err := accountsOf(domain)
		if err != nil {
			return err
		}
		defer coll.Release()

		countResult, err := oleutil.GetProperty(coll, "Count")
		if err != nil {
			return fmt.Errorf("获取账号数量失败: %v", err)
		}
		count := int(countResult.Val)

		for i := 0; i < count; i++ {
			item, err := oleutil.GetProperty(coll, "Item", int32(i))
			if err != nil {
				continue
			}
			acc := item.ToIDispatch()
			accounts = append(accounts, readAccount(acc))
			acc.Release()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// Create 实现 Directory。
func (d *hmailDirectory) Create(email, password, firstName, lastName string, adminLevel int64) error {
	domainName, err := domainOf(email)
	if err != nil {
		return err
	}

	return d.withApp(func(app *ole.IDispatch) error {
		domain, err := findDomain(app, domainName)
		if err != nil {
			return err
		}
		defer domain.Release()

		coll, err := accountsOf(domain)
		if err != nil {
			return err
		}
		defer coll.Release()

		if _, err := oleutil.GetProperty(coll, "ItemByAddress", email); err == nil {
			return fmt.Errorf("邮箱地址 [%s] 已存在", email)
		}

		obj, err := oleutil.CallMethod(coll, "Add")
		if err != nil {
			return fmt.Errorf("创建账号失败: %v", err)
		}
		acc := obj.ToIDispatch()
		defer acc.Release()

		if _, err := oleutil.PutProperty(acc, "Address", email); err != nil {
			return fmt.Errorf("设置邮箱地址失败: %v", err)
		}
		if _, err := oleutil.PutProperty(acc, "Password", password); err != nil {
			return fmt.Errorf("设置密码失败: %v", err)
		}
		if firstName != "" {
			if _, err := oleutil.PutProperty(acc, "PersonFirstName", firstName); err != nil {
				return fmt.Errorf("设置名字失败: %v", err)
			}
		}
		if lastName != "" {
			if _, err := oleutil.PutProperty(acc, "PersonLastName", lastName); err != nil {
				return fmt.Errorf("设置姓氏失败: %v", err)
			}
		}
		if _, err := oleutil.PutProperty(acc, "Active", true); err != nil {
			return fmt.Errorf("设置激活状态失败: %v", err)
		}
		if _, err := oleutil.PutProperty(acc, "Adminlevel", adminLevel); err != nil {
			return fmt.Errorf("设置管理员级别失败: %v", err)
		}
		if _, err := oleutil.CallMethod(acc, "Save"); err != nil {
			return fmt.Errorf("保存账号失败: %v", err)
		}
		return nil
	})
}

// Update 实现 Directory。
func (d *hmailDirectory) Update(email, firstName, lastName string, adminLevel int64) error {
	return d.withAccount(email, func(acc *ole.IDispatch) error {
		if firstName != "" {
			if _, err := oleutil.PutProperty(acc, "PersonFirstName", firstName); err != nil {
				return fmt.Errorf("设置名字失败: %v", err)
			}
		}
		if lastName != "" {
			if _, err := oleutil.PutProperty(acc, "PersonLastName", lastName); err != nil {
				return fmt.Errorf("设置姓氏失败: %v", err)
			}
		}
		if adminLevel >= 0 {
			if _, err := oleutil.PutProperty(acc, "Adminlevel", adminLevel); err != nil {
				return fmt.Errorf("设置管理员级别失败: %v", err)
			}
		}
		if _, err := oleutil.CallMethod(acc, "Save"); err != nil {
			return fmt.Errorf("保存账号修改失败: %v", err)
		}
		return nil
	})
}

// Delete 实现 Directory。
func (d *hmailDirectory) Delete(email string) error {
	return d.withAccount(email, func(acc *ole.IDispatch) error {
		if _, err := oleutil.CallMethod(acc, "Delete"); err != nil {
			return fmt.Errorf("删除账号失败: %v", err)
		}
		return nil
	})
}

// SetPassword 实现 Directory。
func (d *hmailDirectory) SetPassword(email, password string) error {
	return d.withAccount(email, func(acc *ole.IDispatch) error {
		if _, err := oleutil.PutProperty(acc, "Password", password); err != nil {
			return fmt.Errorf("设置新密码属性失败: %v", err)
		}
		if _, err := oleutil.CallMethod(acc, "Save"); err != nil {
			return fmt.Errorf("保存账号修改失败: %v", err)
		}
		return nil
	})
}

// DisplayName 实现 Directory；账号不存在时回退为原始地址。
func (d *hmailDirectory) DisplayName(email string) (string, error) {
	if email == "" {
		return "", nil
	}
	account, err := d.Get(email)
	if err != nil {
		return email, err
	}
	name := strings.TrimSpace(account.FirstName + account.LastName)
	if name == "" {
		return email, nil
	}
	return fmt.Sprintf(`%s <%s>`, mime.BEncoding.Encode("utf-8", name), email), nil
}
