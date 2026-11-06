//go:build !windows

package directory

// New 在非 Windows 平台返回不支持账号目录管理的实现。
//
// 这样可以让服务在 Linux/macOS 上编译并运行单元测试（邮件收发走 IMAP/SMTP，
// 只有账号管理依赖 hMailServer COM）。
func New(adminPassword string) Directory {
	return unsupportedDirectory{}
}

type unsupportedDirectory struct{}

func (unsupportedDirectory) Get(string) (*Account, error) { return nil, ErrUnsupported }

func (unsupportedDirectory) List(string) ([]Account, error) { return nil, ErrUnsupported }

func (unsupportedDirectory) Create(string, string, string, string, int64) error {
	return ErrUnsupported
}

func (unsupportedDirectory) Update(string, string, string, int64) error { return ErrUnsupported }

func (unsupportedDirectory) Delete(string) error { return ErrUnsupported }

func (unsupportedDirectory) SetPassword(string, string) error { return ErrUnsupported }

func (unsupportedDirectory) DisplayName(email string) (string, error) { return email, ErrUnsupported }
