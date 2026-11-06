# hMailServer Web 邮件网关 — 前后端全量功能测试报告

- 测试日期：2026-11-06
- 测试对象：`email-server`（Go 后端）+ `email-web`（React 前端）
- 测试环境：Windows 本机，Go 1.26.7，Node/pnpm 10.34.9，真实 hMailServer（IMAP 993 / SMTP 587）
- 结论：**自动化套件全绿；后端 23 个 HTTP 接口全部具备 HTTP 层用例，其中 18 个已在真实 hMailServer 上跑通；前端 44 个用例、lint、生产构建全部通过。**

---

## 1. 后端

### 1.1 静态检查

| 检查 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `gofmt -l .` | 无输出（格式干净） |
| `go test ./... -count=1` | 全包通过 |

### 1.2 覆盖率（语句）

| 包 | 本轮覆盖率 | 上一轮 |
|---|---|---|
| `config` | 77.7% | 77.7% |
| `controller` | **64.2%** | 42.1% |
| `mailbox` | 18.6% | 18.6% |
| `middleware` | 76.7% | 76.7% |
| `service` | 68.5% | 68.5% |
| `utils` | 57.5% | 57.5% |
| **总体** | **46.8%** | 44.2% |

`apperr`、`contacts`、`directory`、`router`、`main` 默认套件仍无独立单测；`contacts`、`directory` 已由真实环境用例走通真实实现。

### 1.3 新增自动化用例

新增 `controller/functional_extra_test.go`：

| 用例 | 覆盖接口 |
|---|---|
| `TestMailOpsAPI` | `/mail/unread-total`、`/mail/status`、`/mail/star-list`、`/mail/move`、`/mail/delete`，含同文件夹移动、非法状态标志等错误分支 |
| `TestUserAndContactAPI` | `/user/chgpwd` 成功、`/user/create`、`/user/update`、`/user/list`、`/user/delete`、`/user/contact/{save,list,delete,clear}`、`/user/uploadavatar` |
| `TestUnScheduleAPI` | `/mail/save-draft`（带 `X-Schedule-Send`）+ `/mail/un-schedule`，校验头部被移除、草稿被重写 |

新增 `controller/real_full_test.go`（默认 `t.Skip`，需 `MAIL_REAL_TEST=1`）：`TestRealFullHTTPFlow`。

---

## 2. HTTP 接口覆盖矩阵

后端实际注册 **23 个接口**：2 个公开接口（`/login`、`/viewfile`）+ 21 个 JWT 保护接口。

| # | 接口 | 自动化 HTTP 用例 | 真实 hMailServer |
|---|---|---|---|
| 1 | `POST /api/login` | 成功 + 错误密码 | 成功 + 错误密码 |
| 2 | `GET /api/viewfile` | 正常/路径穿越/缺参 | 真实内联图片 |
| 3 | `POST /api/user/chgpwd` | 成功 + 旧密码错误 | 未执行（避免改生产密码） |
| 4 | `POST /api/user/list` | 通过 | 通过（COM 真实目录） |
| 5 | `POST /api/user/create` | 通过 | 未执行（避免改生产账号） |
| 6 | `POST /api/user/delete` | 通过 | 未执行 |
| 7 | `POST /api/user/update` | 通过 | 未执行 |
| 8 | `POST /api/user/uploadavatar` | 通过（临时静态目录） | 通过（写临时目录） |
| 9 | `POST /api/user/contact/list` | 通过 | 通过（真实 yiyidb 联系人库） |
| 10 | `POST /api/user/contact/save` | 通过 | 通过 |
| 11 | `POST /api/user/contact/delete` | 通过 | 通过 |
| 12 | `POST /api/user/contact/clear` | 通过 | 通过 |
| 13 | `POST /api/mail/list` | 通过 | 通过（INBOX/Sent/Drafts/Deleted） |
| 14 | `POST /api/mail/star-list` | 通过 | 通过 |
| 15 | `POST /api/mail/unread-total` | 通过 | 通过 |
| 16 | `POST /api/mail/detail` | 通过 | 通过（最小/大附件邮件） |
| 17 | `POST /api/mail/status` | 通过 | 通过（加/去 Flagged、加 Seen） |
| 18 | `POST /api/mail/download` | 成功 + 无效 PartID | 通过（真实附件，校验文件名头） |
| 19 | `POST /api/mail/move` | 通过 + 同文件夹错误 | 通过（INBOX→Junk） |
| 20 | `POST /api/mail/delete` | 通过 | 通过（Junk 永久删除） |
| 21 | `POST /api/mail/save-draft` | 通过（含定时草稿） | 未单独执行（定时发送走 `send`） |
| 22 | `POST /api/mail/send` | 立即发送 + 附件 | 立即发送 + 定时发送设置 |
| 23 | `POST /api/mail/un-schedule` | 通过 | 通过（定时任务取消 + 草稿重写） |

前端 API 模块共 21 个 `request` 调用（`auth.js` 1、`mail.js` 11、`user.js` 5、`contact.js` 4），另有 `Personal.jsx` 的头像上传和 `viewfile` 直连；路径与后端一致。

---

## 3. 真实 hMailServer 联调结果

命令：

```powershell
cd email-server
$env:MAIL_REAL_TEST="1"
go test ./controller -run TestRealFullHTTPFlow -count=1 -v
```

结果：`PASS`，3.61s。通过进程内 `httptest` 启动真实 `router`，使用真实 `mailbox.NewIMAPFactory`、`mailbox.NewSMTPSender`、`directory.New` 与真实 yiyidb 联系人库。

实测通过项：

- 登录成功；错误密码登录失败；无 Token 访问返回 401。
- 真实账号目录列表包含 `admin@jin337.top`。
- INBOX / Sent / Drafts / Deleted 列表、星标列表、未读总数均返回 200。
- 现有带附件邮件：详情正文非空、附件下载成功且带 `Content-Disposition`、正文内联图片经 `viewfile` 可读取。
- 向本账号发送一封带附件测试邮件：INBOX 收到、详情正文包含标记、附件解析成功。
- 标记 Flagged 后星标列表可见；标记 Seen、取消 Flagged 均成功。
- 同文件夹移动返回 400；移动到 Junk 成功；按 Junk 中的新 UID 永久删除成功。
- 定时发送：`send` 带 `x-schedule-send` 返回“定时发送已设置”，草稿箱出现草稿；`un-schedule` 返回 200，草稿被重写、定时时间清空。
- 联系人保存/列表/删除/清空全部 200。
- 头像上传写入临时静态目录成功（未覆盖生产头像）。
- 缺少 `uid` 的详情、无效 `PartID` 下载均返回失败。
- 测试邮件与定时草稿在用例结束时清理；启动时还会先清理历史 `FULLTEST-` 残留。

未在真实环境执行的高风险操作：改密、创建/删除/更新真实账号。这些接口已由内存目录替身覆盖，避免对生产域账号产生不可逆影响。

联调注意：hMailServer 的 `MOVE` 之后目标文件夹中的 UID 可能变化，后续删除/清理必须使用目标文件夹重新列出的 UID。首轮脚本按原 UID 删除曾留下 1 封测试邮件，修正为“移动后按主题在 Junk 重新取 UID”并清理完毕；这是测试脚本问题，不是产品接口缺陷。

---

## 4. 前端

| 检查 | 结果 |
|---|---|
| `corepack pnpm@10 test` | 7 个文件 / **44 用例全部通过**，20.30s |
| `corepack pnpm@10 lint` | **0 error / 1 warning** |
| `corepack pnpm@10 build` | 成功，1114 模块，13.80s |

用例分布：`utils/mail.test.js` 14、`utils/index.test.jsx` 11、`useAutoRefresh.test.js` 3、`useMailbox.test.jsx` 7、`List.test.jsx` 3、`Detail.test.jsx` 3、`Menu.test.jsx` 3。

### 告警与未处理项

1. `src/hooks/useMailbox.jsx:681`：首次加载 `useEffect` 内同步 `setCurrentFolder(folderList[0])`，触发 `react-hooks/set-state-in-effect` warning。属既有初始化逻辑，本轮未改动。
2. `chunk-preview` 11,075 kB（gzip 2,972 kB）、`chunk-vendor` 947 kB，超过 800 kB 阈值。
3. `jit-viewer` 的 `fs` 被浏览器构建外部化（该依赖的 Node 分支不会在浏览器执行）。
4. 本轮未做浏览器级 UI E2E；前端为组件/单元测试 + 生产构建 + 后端真实接口联调。

---

## 5. 未覆盖 / 剩余缺口

| 缺口 | 说明 | 建议 |
|---|---|---|
| `mailbox/imap.go` 真实会话方法 | 默认单测只覆盖纯函数；`Connect/Select/Search/Fetch*` 等依赖真实 IMAP，默认套件为 0% | 保留真实环境 opt-in 测试；如需 CI 覆盖，引入可编程 IMAP 测试服务器 |
| `service/mail_detail_parts.go` 的 `mailDetailParts` | 默认 2.8%：内存替身的 `FetchStructure` 返回 `ErrUnsupported`，只能测规划/解码辅助函数 | 真实详情用例已覆盖主路径；可给 `FetchStructure` 增加可编程 fake 以提升单测 |
| `contacts` / `directory` | 无独立单测；真实实现由 opt-in 用例走通 | 给 contacts 增加 CRUD/排序/前缀隔离单测；directory 受 COM 限制，保持真实环境验证 |
| `apperr` / `router` / `main` | 无独立单测 | `apperr` 可低成本补映射测试；`router` 可断言路由表 |
| `controller` 错误辅助 | `failError` / `failWithDetail` 仍 0%，只有 MIME 构建失败等少数路径触发 | 补一条非法定时时间/构建失败的 save-draft 用例 |
| 真实账号增删改/改密 | 有意未在生产域执行 | 在测试域或快照环境执行 |
| 定时到点真实投递 | 本轮只验证“设置 + 取消”；此前轮次已验证到点投递与重启恢复 | 需要时用 1–2 分钟后的定时任务复测 |
| 并发/压力 | 未做 | 列表/详情已有性能基线，功能测试不含并发 |

---

## 6. 复跑命令

```powershell
# 后端
$env:PATH = "$env:USERPROFILE\go-toolchain\go\bin;" + $env:PATH
$env:GOPROXY = "https://goproxy.cn,direct"
cd D:\Desktop\Project\hmail\email-server
go build ./...
go vet ./...
gofmt -l .
go test ./... -count=1

# 覆盖率（Go 会把 profile 写成无扩展名文件）
go test ./... -count=1 -coverprofile=coverage-final
go tool cover "-func=D:\Desktop\Project\hmail\email-server\coverage-final"

# 真实 HTTP 全链路（会向本账号发一封测试邮件并自动清理）
$env:MAIL_REAL_TEST="1"
go test ./controller -run TestRealFullHTTPFlow -count=1 -v

# 前端
cd D:\Desktop\Project\hmail\email-web
corepack pnpm@10 test
corepack pnpm@10 lint
corepack pnpm@10 build
```

---

## 7. 测试影响与清理

- 真实测试向 `admin@jin337.top` 自发一封带附件邮件，随后移动至 Junk 并永久删除。
- 定时发送测试创建一封 2 小时后的定时草稿，随后取消定时并删除。
- 头像写入 `t.TempDir()`，不覆盖 `email-server/static/avatars`。
- 联系人使用临时 yiyidb 库，不写生产 `cache/contact_db`。
- 管理员密码、姓名、管理员级别未修改。
- 本轮产生的临时覆盖率文件 `coverage-final`、`coverage-final2` 为测试产物，可按需删除；源码与构建产物未做业务改动。
