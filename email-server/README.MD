# email-server

hMailServer 的 Web 邮件网关后端：对前端暴露 HTTP 接口，对下通过 IMAP/SMTP 收发邮件、通过 hMailServer COM 管理账号。

## 1. 技术栈

- Go 1.26 + Gin
- `emersion/go-imap` v1、`net/smtp`（SMTP）
- `jhillyerd/enmime`、`emersion/go-message`（MIME 解析与重写）
- `go-ole`（hMailServer Application COM，仅 Windows）
- `jacoblai/yiyidb`（联系人 KV 存储）
- `golang-jwt/jwt/v5`（登录令牌）

## 2. 目录与分层

```
email-server/
├── main.go                  # 入口：配置、联系人库、依赖组装、路由、优雅关闭
├── apperr/                  # 业务错误码（200/400/401/500）与错误包装
├── config/                  # 配置加载与校验（config.env + 环境变量）
├── mailbox/                 # IMAP/SMTP 抽象：Client / Factory / Sender
├── directory/               # hMailServer 目录服务
│   ├── directory.go         # 接口、Account、ErrUnsupported
│   ├── hmail_windows.go     # COM 实现（//go:build windows）
│   └── hmail_other.go       # 非 Windows 占位实现
├── contacts/                # 联系人存储接口 + yiyidb 实现 + 内存实现
├── model/                   # 请求/响应结构体
├── utils/                   # 格式化、日期、校验、头像、邮件头
├── service/                 # 业务实现（按职责拆文件）
│   ├── service.go           # 依赖、构造、可注入的异步/休眠
│   ├── mail_list.go         # 列表、星标、未读数
│   ├── mail_detail.go       # 详情、内联图片、附件
│   ├── mail_ops.go          # 标记、移动、删除、取消定时
│   ├── mail_build.go        # 撰写模型与 MIME 组装
│   ├── mail_send.go         # 草稿、发送、重试、已发送联系人
│   ├── user.go              # 登录、改密、账号、联系人、头像
│   └── domain.go            # 文件夹/标记常量
├── controller/              # HTTP 层
│   ├── controller.go        # Handler、统一响应、参数绑定
│   ├── user.go              # 用户/联系人/头像
│   ├── email.go             # 邮件列表、详情、状态、移动、删除、下载
│   ├── compose.go           # multipart 解析、草稿、发送
│   └── viewfile.go          # 静态文件读取（含路径穿越防护）
├── middleware/jwt.go         # JWT 鉴权
├── router/router.go          # 路由注册
├── internal/testutil/        # 内存替身与测试邮件构造
├── static/                   # 头像、邮件内联图片（运行期生成）
├── test/test1.eml            # 真实邮件样本
└── config/config.env         # 默认配置
```

依赖注入：`service.New(service.Deps{Config, IMAP, SMTP, Directory, Contacts})`。`IMAP` 为 `mailbox.Factory`，`SMTP` 为 `mailbox.Sender`，`Directory`/`Contacts` 均为接口，测试时替换为 `internal/testutil` 的内存实现。

## 3. 配置

配置优先级：内置默认值 → `<cwd>/config/config.env` → 环境变量。

- 配置文件路径可用 `EMAIL_SERVER_CONFIG` 指定；文件不存在时仅使用默认值 + 环境变量（便于容器部署）。
- 环境变量名由配置键自动映射：`mail.server.host` → `MAIL_SERVER_HOST`，`jwt.life.web.span` → `JWT_LIFE_WEB_SPAN`。

| 键 | 默认 | 说明 |
| --- | --- | --- |
| `mail.server.host` | `127.0.0.1` | hMailServer 主机，IMAP/SMTP 共用 |
| `mail.server.port` | `8058` | HTTP 监听端口 |
| `imap.port` | `993` | IMAP 端口 |
| `smtp.port` | `587` | SMTP 端口 |
| `admin.password` | `a123456` | hMailServer 管理员密码 |
| `jwt.secret` | `email-server-hmail-20260525` | JWT 签名密钥 |
| `jwt.life.web.span` | `24` | 令牌有效期（小时） |
| `gin.mode` | `debug` | `debug` / `release` / `test` |
| `api.prefix` | 跟随模式 | debug 为 `/api`，release 为 `/`；建议生产显式设为 `/api` |
| `static.dir` | `static` | 静态目录 |
| `contacts.db.path` | `cache/contact_db` | 联系人 KV 目录 |
| `server.max.upload` | `20` | 单次上传上限（MB），同时设置 Gin `MaxMultipartMemory` |
| `cors.origins` | `*` | 跨域白名单，逗号分隔 |

启动日志会打印实际生效配置。校验规则：端口 1–65535、`gin.mode` 枚举合法、JWT 小时 > 0、上传 MB > 0。

## 4. 运行与构建

```bash
# 依赖
go mod download

# 开发（热重载，读取根目录 .air.toml）
air

# 直接运行
go run .

# 当前平台构建
go build -trimpath -ldflags="-s -w" -o email-server.exe .

# 交叉编译
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o email-server-linux .
```

`Makefile`：`make build`、`make win`、`make linux`、`make mac`、`make test`。所有目标均对当前包（`.`）整体编译，适配多包结构。

> 非 Windows 编译不包含 hMailServer COM；用户管理与账号创建相关接口会返回“当前平台不支持该操作”，邮件收发与联系人功能不受影响。

## 5. 接口

统一响应体：`{"code":200,"msg":"...","data":...}`。业务失败也返回 HTTP 200，以 `code`（400/401/500）区分；鉴权失败返回 401。

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/login` | 否 | 登录，返回 token 与用户信息 |
| GET | `/api/viewfile?url=static/...` | 否 | 读取头像/内联图片，越界路径返回 400 |
| POST | `/api/user/chgpwd` | 是 | 修改密码 |
| POST | `/api/user/list` | 是 | 账号列表 |
| POST | `/api/user/create` | 是 | 新建账号 |
| POST | `/api/user/update` | 是 | 修改账号 |
| POST | `/api/user/delete` | 是 | 删除账号 |
| POST | `/api/user/uploadavatar` | 是 | 上传头像（≤5MB，`image/*`） |
| POST | `/api/user/contact/list` | 是 | 联系人列表 |
| POST | `/api/user/contact/save` | 是 | 保存联系人 |
| POST | `/api/user/contact/delete` | 是 | 删除联系人 |
| POST | `/api/user/contact/clear` | 是 | 清空联系人 |
| POST | `/api/mail/list` | 是 | 邮件列表（分页） |
| POST | `/api/mail/star-list` | 是 | 星标邮件 |
| POST | `/api/mail/unread-total` | 是 | 未读数量 |
| POST | `/api/mail/detail` | 是 | 邮件详情 |
| POST | `/api/mail/status` | 是 | 标记已读/未读/星标 |
| POST | `/api/mail/download` | 是 | 下载附件 |
| POST | `/api/mail/move` | 是 | 移动邮件 |
| POST | `/api/mail/delete` | 是 | 删除邮件 |
| POST | `/api/mail/save-draft` | 是 | 保存草稿（multipart） |
| POST | `/api/mail/send` | 是 | 发送邮件（multipart，支持定时） |
| POST | `/api/mail/un-schedule` | 是 | 取消定时发送 |

鉴权头：`Authorization: Bearer <token>`。令牌载荷包含邮箱账号与密码，服务端据此建立 IMAP/SMTP 会话（因此账号密码变更后需重新登录）。

## 6. 关键实现说明

- **列表分页**：沿用旧版语义，从文件夹末尾向前取第 `page` 页；`filter` 支持 `unread`、`date_asc`、`date_desc`、`size_asc`、`size_desc`，不传即全部（前端传入的 `all` 会被忽略）。
- **星标邮件**：遍历 `INBOX/Sent/Drafts/Deleted/Junk` 收集 `Flagged` 邮件，结果不分页；非 INBOX 文件夹为保证列表摘要可读，会以只读方式强制 `Seen`（不写回服务器）。
- **详情与内联图片**：`cid:` 图片落地到 `static/images/{email}/{uid}/image_{partID}{ext}`，并把原 `cid` 保留在 `data-href` 供前端预览；URL 统一用 `filepath.ToSlash` 生成，避免 Windows 反斜杠进入链接。
- **MIME 组装**：重写为 `multipart/mixed` 套 `multipart/alternative`；正文内 data-uri 图片转为 `cid`；Base64 按 76 字符换行；非 ASCII 头用 RFC 2047 编码；定时时间写入 `X-Schedule-Send`（UTC）。
- **发送重试**：SMTP 失败重试两次（间隔 5s、15s）；成功后写入 Sent 并异步记录最近联系人。
- **定时发送**：报文带 `X-Schedule-Send` 时先存草稿并打 `Draft` 标记，后台 goroutine 等到点后按 `Message-ID` 找回草稿、校验定时头仍在，再投递并移入 Sent；取消定时通过删除并重写草稿（去掉 `X-Schedule-Send`、移除 `Draft`）实现，到点任务发现头部缺失即跳过。
- **定时任务重启恢复**：任务只在内存中，重启会丢。鉴权中间件在用户首个带 JWT 的请求时回调 `EnsureScheduledRestored`，异步用 `FetchAll`（IMAP `1:*`）扫描草稿箱，把仍带 `X-Schedule-Send` 的草稿重新注册（按「邮箱 + Message-ID」去重，每用户只扫一次）；未到点按剩余时间等待，已过点立即补发；执行前仍校验定时头，撤销过的草稿不会被误发。
- **安全**：`viewfile` 对 `url` 做 `path.Clean` 并校验必须位于 `static.dir` 内；上传头像校验 MIME 前缀与体积；删除/移动校验 UID 与文件夹合法性。

## 7. 测试

```bash
go build ./...
go vet ./...
go test ./...

# 单个用例
go test ./service -run TestMailDetail -v
```

测试组织：

- `internal/testutil/fake.go`：内存 IMAP/SMTP/Directory/Store，支持失败注入、调用计数。
- `internal/testutil/mail.go`：生成 enmime 可解析的 multipart 邮件、构造附件与地址列表。
- `config`：配置合并优先级、非法值。
- `utils`：格式化、摘要、排序、地址解析。
- `service`：列表/星标/未读、详情与内联图、标记/移动/删除、MIME 组装、草稿/发送/定时/重试、用户与联系人、真实 `test/test1.eml` 解析。
- `controller`：登录、鉴权、参数校验、viewfile 越界。
- `middleware`：JWT 签发、解析、算法校验。

当前状态：以上命令全部通过（`mailbox`/`directory`/`contacts`/`router`/`apperr`/`model` 暂无独立测试文件，其行为由 service/controller 层覆盖）。

真实环境补充验证（hMailServer IMAP 993 / SMTP 587 在线、使用有效域账号）：HTTP 登录签发 JWT、收件箱列表、邮件详情、未读数、COM 用户列表、联系人列表、viewfile 正常读取与越界拦截均返回成功；登录会真实建立 IMAP 会话、补齐标准文件夹并更新最后登录时间。

定时发送与撤销也已实测：定时设定后草稿带 `Draft` 标记与定时时间，到点真实投递并移入已发送；撤销后草稿重写（`X-Schedule-Send` 与 `Draft` 均被清除），越过原定时间不产生投递。过程中发现并修复了 `SearchCriteria.Header` 为 nil 时 `Add` 触发 panic 的缺陷，`mailbox/imap_test.go` 已覆盖。

## 8. 排障

| 现象 | 排查 |
| --- | --- |
| 启动即退出并提示配置错误 | 检查 `config/config.env` 的引号与整数项；`gin.mode` 只能是 debug/release/test |
| 登录返回“用户名或密码错误” | IMAP 端口/防火墙、账号是否存在、密码是否正确 |
| 用户列表返回“不支持” | 非 Windows 构建，或 hMailServer COM 未注册、管理员密码错误、服务账号权限不足 |
| 内联图片 404 | `static.dir` 不存在/不可写，或 `viewfile` 收到越界 url |
| 生产环境所有接口 404 | release 模式前缀默认 `/`，需设 `api.prefix=/api` |
| 定时邮件未发送 | 进程需常驻；待发任务在内存中，重启丢失 |
