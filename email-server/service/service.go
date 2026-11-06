// Package service 承载全部业务逻辑。
//
// 依赖（IMAP/SMTP/账号目录/联系人存储）都通过接口注入，
// 因此可以在没有任何外部邮件服务器的情况下用假实现做完整测试。
package service

import (
	"sync"
	"time"

	"email-server/config"
	"email-server/contacts"
	"email-server/directory"
	"email-server/mailbox"
)

// Service 是业务服务，所有方法都以它作为接收者。
type Service struct {
	cfg      config.Config
	imap     mailbox.Factory
	smtp     mailbox.Sender
	dir      directory.Directory
	contacts contacts.Store

	// async 决定后台任务如何执行（默认起 goroutine，测试中可同步执行）。
	async func(func())
	// sleep 决定定时任务如何等待（测试中可替换为记账函数）。
	sleep func(time.Duration)

	// schedMu/sched 记录已注册的定时任务，避免同一封草稿被重复调度。
	schedMu sync.Mutex
	sched   map[string]struct{}
	// restoreMu/restored 记录已扫描过草稿箱的用户，避免每次请求都扫一遍。
	restoreMu sync.Mutex
	restored  map[string]struct{}
}

// Deps 汇总构造 Service 所需的依赖。
type Deps struct {
	Config    config.Config
	IMAP      mailbox.Factory
	SMTP      mailbox.Sender
	Directory directory.Directory
	Contacts  contacts.Store
}

// New 构造业务服务。
func New(deps Deps) *Service {
	return &Service{
		cfg:      deps.Config,
		imap:     deps.IMAP,
		smtp:     deps.SMTP,
		dir:      deps.Directory,
		contacts: deps.Contacts,
		async: func(fn func()) {
			go fn()
		},
		sleep:    time.Sleep,
		sched:    map[string]struct{}{},
		restored: map[string]struct{}{},
	}
}

// SetAsyncRunner 替换后台任务执行器（用于测试同步执行）。
func (s *Service) SetAsyncRunner(fn func(func())) {
	if fn != nil {
		s.async = fn
	}
}

// SetSleeper 替换等待函数（用于测试跳过真实等待）。
func (s *Service) SetSleeper(fn func(time.Duration)) {
	if fn != nil {
		s.sleep = fn
	}
}

// Config 返回服务使用的配置副本。
func (s *Service) Config() config.Config { return s.cfg }
