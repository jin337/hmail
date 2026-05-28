# email-server

## 1. 安装依赖

```bash
go mod tidy
```

## 2. 安装 air 热重载

```bash
go install github.com/cosmtrek/air@latest
```

## 3. 修改配置

打开 `config/config.go` 修改为你的邮件服务器地址

```go
var (
	HmailHost = "127.0.0.1"
	SmtpPort  = 25
	ImapPort  = 143
	AdminPwd  = "a123456"
)
```

## 4. 启动项目

```bash
air
```

默认运行端口：`8058`
