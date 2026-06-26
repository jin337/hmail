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
	Port      = 8058
	HmailHost = "127.0.0.1"
	SmtpPort  = 25
	ImapPort  = 993
	AdminPwd  = "hMailServer密码"
)
```

## 4. 启动项目

```bash
air
```

默认运行端口：`8058`

## 5. 打包项目

```bash
# 编译
go build -trimpath -ldflags="-s -w" -o email-server.exe main.go

# UPX压缩
upx --lzma -9 email-server.exe

# 校验
upx -t email-server.exe
```
