# email-server 邮件服务后端
基于 Gin + IMAP/SMTP 开发，适配 `hMailServer` 自建邮件服务器
完整实现：登录鉴权、发邮件、收件箱/已发送/草稿箱管理、邮件移动删除、附件下载，**全部接口 POST 请求**

## 项目简介
1. JWT 登录认证，Token 内置邮箱账号密码，业务接口无需重复传参
2. 支持发送邮件、抄送、多文件附件上传
3. 自动归档已发送邮件到 Sent 文件夹
4. 支持读取任意邮箱文件夹邮件、分页查询
5. 邮件查看正文、移动文件夹、彻底删除
6. 附件在线流式下载
7. 规范分层架构：配置层、中间件、控制器、业务服务
8. 支持 `air` 热重载开发，无需频繁重启服务

## 项目结构
```
email-server/
├── go.mod               # 依赖管理
├── go.sum
├── .air.toml            # air热重载配置
├── README.md            # 项目文档
├── main.go              # 项目入口 + 路由
├── config/
│   └── app.go           # 服务器配置、JWT配置、文件夹常量
├── middleware/
│   └── jwt.go           # JWT全局鉴权中间件
├── controller/
│   ├── auth.go          # 登录接口
│   └── email.go         # 邮件业务接口
└── service/
    └── email.go         # IMAP/SMTP 核心业务逻辑
```

## 环境依赖
- Go >= 1.21
- 本地搭建 `hMailServer` 邮件服务
- 开启端口：
  - SMTP 25
  - IMAP 143

## 快速部署
### 1. 安装依赖
```bash
go mod tidy


go build -o hst-mail.exe
```

### 2. 安装 air 热重载
```bash
go install github.com/cosmtrek/air@latest
```

### 3. 修改配置
打开 `config/app.go` 修改为你的邮件服务器地址
```go
var (
	HmailHost = "127.0.0.1"
	SmtpPort   = 25
	ImapPort = 993
	SenderName = "邮件服务"
)
```

### 4. 启动项目
```bash
air
```
默认运行端口：`8888`

## 全局规则
1. 所有接口**统一 POST**
2. 除登录接口外，所有接口必须携带 Token
3. 请求头格式
```
Authorization: Bearer 你的登录token
```
4. 传参格式：`form-data`

## 接口文档
### 1. 登录获取Token
**地址**：`POST /api/email/login`
**参数**
| 参数 | 说明 | 必填 |
|----|----|----|
| email | 邮箱账号 | 是 |
| pwd | 邮箱登录密码 | 是 |

**返回示例**
```json
{
    "code": 200,
    "msg": "登录成功",
    "token": "xxxx.xxxx.xxxx"
}
```

---

### 2. 发送邮件
**地址**：`POST /api/email/send`
**无需传邮箱账号密码，自动从Token读取**
| 参数 | 说明 | 必填 |
|----|----|----|
| to | 收件人，多个逗号分隔 | 是 |
| cc | 抄送邮箱 | 否 |
| subject | 邮件标题 | 是 |
| content | 邮件正文 | 否 |
| files | 多附件文件 | 否 |

---

### 3. 获取文件夹邮件列表
**地址**：`POST /api/email/list`
| 参数 | 说明 | 必填 |
|----|----|----|
| folder | 文件夹名称 | 是 |
| page | 页码默认1 | 否 |
| size | 每页条数默认20 | 否 |

**内置标准文件夹**
- INBOX 收件箱
- Sent 已发送
- Drafts 草稿箱
- Deleted 垃圾箱
- Junk 垃圾邮件

---

### 4. 获取邮件详情
**地址**：`POST /api/email/detail`
| 参数 | 说明 | 必填 |
|----|----|----|
| folder | 文件夹名 | 是 |
| uid | 邮件唯一ID | 是 |

---

### 5. 移动邮件
**地址**：`POST /api/email/move`
| 参数 | 说明 | 必填 |
|----|----|----|
| from_folder | 原文件夹 | 是 |
| to_folder | 目标文件夹 | 是 |
| uid | 邮件UID | 是 |

---

### 6. 删除邮件
**地址**：`POST /api/email/delete`
| 参数 | 说明 | 必填 |
|----|----|----|
| folder | 所在文件夹 | 是 |
| uid | 邮件UID | 是 |

---

### 7. 附件下载
**地址**：`POST /api/email/download`
| 参数 | 说明 | 必填 |
|----|----|----|
| folder | 文件夹 | 是 |
| uid | 邮件UID | 是 |
| idx | 附件下标 0开始 | 否 |
