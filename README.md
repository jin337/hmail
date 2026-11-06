# hmail · hMailServer Web 邮件客户端

基于 hMailServer 的自建 Web 邮箱系统，由 Go 后端（IMAP/SMTP 代理 + hMailServer COM 账号管理）与 React 前端组成。

- 后端：`email-server/`，Go + Gin，通过 IMAP/SMTP 收发邮件，通过 COM 调用 hMailServer 管理域账号。
- 前端：`email-web/`，React 19 + rolldown-vite + Arco Design + Tailwind CSS v4，构建产物输出到 `email-web/Web`。

> 重构约定：前端**视觉样式与交互保持与重构前完全一致**（className、Arco 组件参数、文案、图标资源均逐字保留），本次重构只调整代码分层、状态流转与数据访问方式。

---

## 1. 目录结构

```
hmail/
├── email-server/                 # Go 后端
│   ├── main.go                   # 入口：配置 → 联系人库 → 服务组装 → 路由 → 优雅关闭
│   ├── apperr/                   # 统一业务错误码
│   ├── config/                   # 配置加载（默认值 / config.env / 环境变量）
│   │   ├── config.go
│   │   ├── config.env            # 可提交的示例配置
│   │   └── config_test.go
│   ├── mailbox/                  # IMAP / SMTP 客户端抽象（Client、Factory、Sender）
│   ├── directory/                # hMailServer COM 目录服务（Windows / 非 Windows 构建标签）
│   ├── contacts/                 # 联系人 KV 存储（yiyidb）
│   ├── model/                    # 请求/响应数据结构
│   ├── utils/                    # 格式化、日期、校验、头像、邮件头工具
│   ├── service/                  # 业务层：列表 / 详情 / 操作 / 组装 MIME / 发送 / 用户
│   ├── controller/               # HTTP 层：参数绑定、统一响应、附件下载、路径安全
│   ├── middleware/               # JWT 鉴权
│   ├── router/                   # 路由注册（与旧版路径完全一致）
│   ├── internal/testutil/        # 测试替身与邮件构造工具（不对外导出）
│   ├── static/                   # 头像与邮件内联图片（运行期生成）
│   └── test/test1.eml            # 真实邮件样本（解析/渲染回归用）
└── email-web/                    # React 前端
    ├── src/
    │   ├── api/                  # 接口函数（auth / user / contact / mail / request）
    │   ├── constants/            # 文件夹、分页、菜单、筛选等常量
    │   ├── hooks/                # useMailbox（邮箱状态机）、useAutoRefresh（自动刷新）
    │   ├── utils/                # 通用工具、邮件内容工具
    │   ├── pages/Mail/           # 邮箱页面：index（薄壳）+ Menu / List / Content / Detail
    │   ├── pages/                # Personal（个人中心）、User（用户管理）
    │   ├── views/                # Home / Login / Preview
    │   ├── components/           # WriteMail、Editor、AvatarImage
    │   ├── test/                 # Vitest 全局 setup
    │   ├── router.jsx
    │   └── *.test.*              # 单元/组件测试
    └── Web/                      # 生产构建产物
```

---

## 2. 架构说明

### 2.1 后端分层

| 层 | 包 | 职责 |
| --- | --- | --- |
| 入口 | `main` | 加载配置、打开联系人库、组装依赖、注册路由、监听 `:8058`、优雅关闭 |
| 配置 | `config` | 三层合并（内置默认 → `config/config.env` → 环境变量），校验端口/模式/JWT/上传体积 |
| 接口抽象 | `mailbox`、`directory`、`contacts` | 分别封装 IMAP/SMTP、hMailServer COM、联系人 KV；均以接口形式注入，便于替换与测试 |
| 业务 | `service` | 邮件列表/星标/未读数、详情与内联图片、标记/移动/删除、草稿与发送、MIME 重写、定时邮件、用户与联系人 |
| 协议层 | `controller` | 绑定参数、统一响应 `{code,msg,data}`、按业务码区分 HTTP 200 内的失败 |
| 鉴权 | `middleware` | JWT（`Authorization: Bearer <token>`），载荷携带邮箱账号与密码用于 IMAP/SMTP 会话 |
| 路由 | `router` | 统一前缀 `/api`（debug）或 `/`（release），每条受保护路由单独挂中间件 |

依赖注入：`service.Deps{Config, IMAP, SMTP, Directory, Contacts}`，生产用真实实现，测试用 `internal/testutil` 的内存替身；异步与休眠也可注入（`SetAsyncRunner` / `SetSleeper`），保证定时邮件相关测试可确定地执行。

### 2.2 前端分层

```
页面薄壳 (pages/Mail/index.jsx)
      │  useOutletContext 取 Home 下发的 searchSeq / 用户信息
      ▼
useMailbox（唯一状态机：目录、列表、筛选、详情、写信、联系人、标记）
      │  mailContext 通过 MailProvider 下发
      ▼
视图组件 Menu / List / Content / Detail（只读 context，className 与结构不变）
      │
      ▼
api/*（命名函数） → axios 实例（统一 token、错误提示、租户头）
```

关键点：

- `useMailbox` 承担原 `pages/Mail/index.jsx` 的全部业务，页面组件退化为布局薄壳。
- 搜索不再使用 `registerMethod` 反向注册子组件方法，而是 `Home` 递增 `searchSeq` 单向通知，`useMailbox` 比对序号后触发请求。
- 列表分页统一走 `mergeMailPage`：第 1 页替换、后续页追加，避免并发刷新时闭包旧列表覆盖新数据。
- 自动刷新由 `useAutoRefresh` 调度：前 20 次自动刷新间隔 1 分钟，之后退避到 5 分钟，累计 50 次后停止；每次手动加载会重新计时。

---

## 3. 环境要求

| 组件 | 版本/说明 |
| --- | --- |
| 操作系统 | Windows（账号管理依赖 hMailServer COM；Linux 可编译运行但用户管理接口返回“不支持”） |
| hMailServer | 已安装并启动，开启 IMAP（默认 993）与 SMTP（默认 587），已知管理员密码 |
| Go | 1.26+（`go.mod` 声明 `go 1.26.2`） |
| Node.js | 20.19+ 或 22.12+（rolldown-vite 要求） |
| pnpm | 10.x（本仓库锁文件与 store 均为 pnpm 10 格式，请勿用 pnpm 11/12） |

---

## 4. 后端：配置与启动

### 4.1 配置文件

编辑 `email-server/config/config.env`（也可用环境变量覆盖，键名点转下划线，如 `mail.server.host` → `MAIL_SERVER_HOST`）：

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| `mail.server.host` | `127.0.0.1` | hMailServer 主机（IMAP/SMTP 同机） |
| `mail.server.port` | `8058` | 后端 HTTP 监听端口 |
| `smtp.port` | `587` | SMTP 端口 |
| `imap.port` | `993` | IMAP 端口 |
| `admin.password` | `a123456` | hMailServer 管理员密码 |
| `jwt.secret` | `email-server-hmail-20260525` | JWT 签名密钥 |
| `jwt.life.web.span` | `24` | 令牌有效期（小时） |
| `gin.mode` | `debug` | `debug` / `release` / `test` |
| `api.prefix` | 跟随模式 | 显式指定路由前缀；debug 默认 `/api`，release 默认 `/` |
| `static.dir` | `static` | 静态目录（头像、内联图片） |
| `contacts.db.path` | `cache/contact_db` | 联系人 KV 数据库目录 |
| `server.max.upload` | `20` | 单次请求最大上传体积（MB） |
| `cors.origins` | `*` | 允许跨域来源，逗号分隔 |

> 生产环境把 `gin.mode` 改为 `release` 时，务必同时显式设置 `api.prefix = "/api"`：前端固定请求 `/api/...`，而 release 模式下前缀默认会变成 `/`。

### 4.2 启动

```bash
cd email-server
go mod download

# 开发（热重载，读取 .air.toml）
air

# 或直接运行
go run .
```

启动成功会打印实际生效的配置：

```
配置加载完成: mode=debug prefix=/api listen=:8058 imap=127.0.0.1:993 smtp=127.0.0.1:587
```

### 4.3 构建

```bash
cd email-server

# 当前平台
go build -trimpath -ldflags="-s -w" -o email-server.exe .

# 交叉编译（Linux 产物不含 hMailServer 用户管理能力）
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o email-server-linux .
```

`Makefile` 提供 `make build / win / linux / mac / test`，均为多包整包编译（`go build ... .`）。

---

## 5. 前端：配置与启动

### 5.1 环境变量（`email-web/.env`）

```ini
VITE_BASE_URL=http://localhost:8058   # 后端地址
VITE_SUFFIX=@jin337.top               # 登录账号域名后缀
VITE_PAGE_TITLE=华盛邮件
```

### 5.2 启动与构建

```bash
cd email-web
corepack pnpm@10 install     # 首次
corepack pnpm@10 dev         # 开发服务器，端口 4055，base /web

corepack pnpm@10 build       # 生产构建，产物输出到 Web/
corepack pnpm@10 preview     # 本地预览构建产物
```

> 必须使用 `pnpm@10`。本机 `node_modules` 由 `D:\.pnpm-store\v10` 链接，使用 pnpm 11/12 会报 `ERR_PNPM_UNEXPECTED_STORE`。

### 5.3 部署形态

前端 `base` 为 `/web`、路由 basename 为 `/web`，构建产物 `email-web/Web` 交给 Nginx 等静态服务器；`/api` 反向代理到后端。生产建议：

```nginx
location /web/ {
    root /srv/hmail;          # 对应 email-web/Web 的上级目录
    try_files $uri $uri/ /web/index.html;
}

location /api/ {
    proxy_pass http://127.0.0.1:8058;
    proxy_set_header Host $host;
    client_max_body_size 25m;
}
```

后端 `static` 目录（头像、邮件内联图片）通过 `/static/...` 直接访问，`viewfile` 接口负责带鉴权地读取。

---

## 6. 接口一览

统一响应：`{"code":200,"msg":"success","data":...}`；业务失败仍返回 HTTP 200，`code` 为 `400`/`401`/`500`。除 `viewfile` 为 GET 外，其余均为 POST + JSON（`send` / `save-draft` 为 `multipart/form-data`）。

| 路径 | 鉴权 | 主要参数 | 说明 |
| --- | --- | --- | --- |
| `POST /api/login` | 否 | `email, password` | 登录，返回 `token` 与用户信息 |
| `GET /api/viewfile?url=` | 否 | `url=static/...` | 读取静态文件，带路径穿越防护 |
| `POST /api/user/chgpwd` | 是 | `old_password, new_password` | 修改密码 |
| `POST /api/user/list` | 是 | — | 用户列表（COM） |
| `POST /api/user/create` | 是 | `email, password, person_first_name, person_last_name, is_admin` | 新建账号 |
| `POST /api/user/update` | 是 | `email, person_first_name, person_last_name, is_admin` | 修改账号 |
| `POST /api/user/delete` | 是 | `email` | 删除账号 |
| `POST /api/user/uploadavatar` | 是 | `file`（≤5MB，`image/*`） | 上传头像 |
| `POST /api/user/contact/list` | 是 | `prefix` | 联系人列表（`user_contact` / `user_sent`） |
| `POST /api/user/contact/save` | 是 | `prefix, email, name` | 保存联系人 |
| `POST /api/user/contact/delete` | 是 | `prefix, email` | 删除联系人 |
| `POST /api/user/contact/clear` | 是 | `prefix` | 清空联系人 |
| `POST /api/mail/list` | 是 | `folder, page, size, keyword, filter[]` | 邮件列表（分页） |
| `POST /api/mail/star-list` | 是 | `keyword, filter[]` | 星标邮件（遍历 5 个标准文件夹，不分页） |
| `POST /api/mail/unread-total` | 是 | — | 未读数（含虚拟 Star） |
| `POST /api/mail/detail` | 是 | `folder, uid` | 邮件详情（HTML、内联图片、附件） |
| `POST /api/mail/status` | 是 | `folder, uids[], status, type` | 标记已读/未读/星标 |
| `POST /api/mail/download` | 是 | `folder, uid, part_id` | 下载附件（返回 Blob） |
| `POST /api/mail/move` | 是 | `from_folder, to_folder, uids[]` | 移动邮件 |
| `POST /api/mail/delete` | 是 | `folder, uids[]` | 删除 / 彻底删除 |
| `POST /api/mail/save-draft` | 是 | multipart | 保存草稿 |
| `POST /api/mail/send` | 是 | multipart | 发送邮件（支持定时） |
| `POST /api/mail/un-schedule` | 是 | `folder, uid, status, type` | 取消定时发送 |

`send` / `save-draft` 的 multipart 字段：`from, to, cc, subject, content, folder, uid, part_ids, in_reply_to, references, x-schedule-send, files[]`。

文件夹常量：`INBOX`、`Sent`、`Drafts`、`Deleted`、`Junk`；前端另有虚拟目录 `Star`（星标聚合）与 `compose`（写信页）。

---

## 7. 测试

### 7.1 后端

```bash
cd email-server

$env:GOPROXY="https://goproxy.cn,direct"   # 国内网络建议
go build ./...
go vet ./...
go test ./...
```

覆盖情况：

- `config`：默认值、文件覆盖、环境变量优先级、非法值校验。
- `utils`：邮件名/大小/时间格式化、HTML 摘要、排序、地址解析。
- `service`：列表分页与筛选、星标聚合、详情渲染与内联图片 URL、详情分段抓取（分段规划、正文选择、base64 尺寸推断）、标记/移动/删除校验、MIME 组装、草稿与发送（含定时、重试）、用户与联系人、真实 `.eml` 解析回归。
- `mailbox`：`BODYSTRUCTURE` 转换、分段抓取语法与响应键、附件判断、定时检索条件。
- `controller`：登录、鉴权失败、参数校验、列表/详情、viewfile 路径穿越；本轮补齐星标列表、未读数、标记、移动、删除、取消定时、用户增删改、改密、联系人删除/清空、头像上传及对应错误分支。
- `middleware`：JWT 签发/校验/Bearer 解析/算法校验。

当前结果：`go build`、`go vet`、`gofmt -l .`、`go test ./...` 全部通过。覆盖率（语句）：

| 包 | 覆盖率 |
|---|---|
| `config` | 77.7% |
| `controller` | 64.2% |
| `mailbox` | 18.6% |
| `middleware` | 76.7% |
| `service` | 68.5% |
| `utils` | 57.5% |
| 总体 | 46.8% |

`apperr`、`contacts`、`directory`、`router`、`main` 仍无独立单测；`contacts`/`directory` 在下面的真实环境测试中通过真实实现走通，`mailbox` 的真实 IMAP 会话方法由真实环境测试覆盖但默认套件不计入覆盖率。

### 7.2 前端

```bash
cd email-web
corepack pnpm@10 test     # vitest run
corepack pnpm@10 lint
corepack pnpm@10 build
```

覆盖情况（7 个测试文件 / 44 个用例）：

- `utils`：文件类型、树展开、时间格式化、属性互换、SVG 判断、节流（含尾调用参数与 `cancel`）。
- 邮件工具：内联图片 id、详情内容标准化、按时间分组、分页合并、引用块、FormData 组装。
- `useAutoRefresh`：到点刷新、页面不可见时跳过、`clear` 后停止。
- `useMailbox`：初始化、搜索序号、标记已读、星标目录取消星标、写信开关、发送成功、缺少收件人/主题。
- 视图：Menu（目录与未读数、写信、切换目录清理搜索与选中）、List（列表渲染、未读加粗/选中样式 `selelct-mail`、空态）、Detail（正文/附件、定时提示、空态）。

样式回归通过组件测试中的 className 断言保底（如 `mail-item`、`selelct-mail`、`font-bold`、`py-2`），其余样式以源码字符串为准不做改动。

当前结果：`test` 7 文件 / 44 用例全部通过；`lint` 0 error、1 warning（`src/hooks/useMailbox.jsx:681` 首次加载 effect 内同步 `setCurrentFolder`，属既有逻辑写法，未改动）；`build` 成功（1114 模块），有两条体积告警：`chunk-preview` 11,075 kB（gzip 2,972 kB）与 `chunk-vendor` 947 kB 超过 800 kB 阈值，`jit-viewer` 的 `fs` 被外部化。

### 7.3 真实环境联调（已验证）

在具备 hMailServer（IMAP 993 / SMTP 587）与有效域账号的机器上，已按以下顺序做过一次真实链路验证，全部返回 `code=200`：

1. `POST /api/login` → 签发 JWT，返回账号 ID、显示名、管理员级别。
2. `POST /api/mail/list`（带 JWT）→ 读取真实收件箱，返回 uid、发件人、主题、时间、大小、flags。
3. `POST /api/mail/detail` → 返回真实正文 HTML 与附件列表。
4. `POST /api/mail/unread-total` → 返回各文件夹未读数与虚拟 Star。
5. `POST /api/user/list`、`POST /api/user/contact/list` → 走 hMailServer COM 与联系人库。
6. `GET /api/viewfile` → 正常读取头像；`url=../config/config.env` 被拦截为 `非法文件路径`。
7. 定时发送：`POST /api/mail/send` 带 `x-schedule-send` 后，草稿箱出现带 `Draft` 标记与定时时间的邮件；到点后真实投递成功，草稿移入已发送，收件箱也收到该封自测邮件。
8. 撤销定时：`POST /api/mail/un-schedule` 返回 200；草稿被重写（UID 变化、`X-Schedule-Send` 清除、`Draft` 标记移除），越过原定时间后未产生任何投递，草稿仍保留。
9. 重启恢复：先用一个进程设置定时邮件后**立即退出**（模拟重启、内存任务丢失），再用全新进程启动并发出一次带 JWT 的请求；草稿被自动识别并重新挂上任务，未到点的按原定时间投递，已过点的立刻补发，两种情况均真实投递成功并归档到已发送。测试邮件已全部清理。

登录会真实建立 IMAP 会话并补齐标准文件夹，同时更新 hMailServer 的最后登录时间，属于预期副作用。

本轮（2026-11-06）补充了 HTTP 层的真实全接口回归，可用以下命令复跑（默认跳过，需显式开启）：

```powershell
cd email-server
$env:MAIL_REAL_TEST="1"
go test ./controller -run TestRealFullHTTPFlow -count=1 -v
```

该用例通过进程内 `httptest` 启动真实路由，连接真实 hMailServer，覆盖登录/鉴权、全部邮件列表、详情、附件下载、`viewfile`、发信、标记、移动、删除、定时发送与撤销、联系人增删清空、头像上传和错误分支；测试邮件与定时草稿在用例结束时自动清理，管理员账号的密码、姓名和头像不做修改。

---

## 8. 本次重构要点

### 8.1 后端

- 由单文件 `service/email.go`（1300+ 行）拆为 `service` 下按职责划分的多个文件：`mail_list` / `mail_detail` / `mail_ops` / `mail_build` / `mail_send` / `user`。
- 抽出 `mailbox`（IMAP/SMTP）、`directory`（COM）、`contacts`（KV）三层接口，业务层只依赖接口，测试可替换为内存实现。
- 配置集中到 `config`，支持文件 + 环境变量覆盖与合法性校验；`APIPrefix` 随模式推导并允许显式覆盖。
- 统一响应与错误码（`apperr`），业务失败不再混用 HTTP 状态与业务码。
- 修复：`viewfile` 路径穿越（`path.Clean` 后必须落在 `static.dir` 内）；受保护路由逐条挂载 JWT（旧版 `user.Use()` 未覆盖 contact 子组）；内联图片 URL 在 Windows 下拼接出反斜杠绝对路径；列表分页边界与空结果处理；`mailbox.UIDSearchByHeader` 往 `nil` 的 `SearchCriteria.Header` 写值导致 panic（表现为设置定时发送/按 Message-ID 定位草稿时请求 500，已补 `mailbox/imap_test.go` 回归）。
- 保留旧版兼容语义：邮件列表“从末尾取第 page 页”、星标邮件遍历标准文件夹、非 INBOX 列表强制 `Seen`、`cid:` 图片加 `data-href` 供前端预览。

### 8.2 前端

- 新增 `api/`：把散落的 `request.post('/api/...')` 收敛为命名函数，统一错误处理。
- 新增 `constants/`、`utils/mail.js`、`hooks/useMailbox.jsx`、`hooks/useAutoRefresh.js`，页面组件改为只消费 context。
- `Home.jsx` 的 `registerMethod` 反模式（父组件反向调用子组件方法）改为 `searchSeq` 单向通知。
- `WriteMail` 修复：附件列表 `key` 由 `itemkey` 改为 `email`；抄送回填字段由 `cc_email` 改为 `cc_info`。
- `throttle` 修复：尾调用只记住第一次参数，改为始终执行最后一次调用的参数。
- 清理 `onRead` 重复的星标判断、`onSearch` 中 `DRAFTS` 死代码。
- `vite.config.js`：`src` 别名改为基于 `import.meta.url` 的绝对路径，新增 Vitest（jsdom）配置；`eslint.config.js` 忽略构建产物，避免 lint 时 OOM。

### 8.3 前端零样式变化的边界

以下内容在本次重构中**逐字保留**，后续修改请勿“顺手修正”：

- Tailwind 类名，包括原有拼写 `selelct-mail`、`file_aduio.svg`。
- Arco 组件参数（`Layout.Sider width={220}`、`resizeBoxProps` 的 740/410/350 阈值、`InputTag maxTagCount={5}`、`Upload tip`、`Editor height='auto'` 等）。
- 中文文案与图标资源文件。
- 页面结构与组件层级（`Layout` → `Sider` → `Menu` / `Content` → `List` + `Detail`）。

### 8.4 定时任务重启恢复

定时任务原本只是进程内 goroutine，服务重启会丢失（草稿还在，但不会到点发送）。现在的恢复机制：

- **触发时机**：进程启动后，用户第一次发出带 JWT 的请求时，鉴权中间件回调 `Service.EnsureScheduledRestored`，异步扫描该用户的草稿箱。不需要服务端保存账号密码——密码本来就在 JWT 载荷里。
- **识别方式**：用 IMAP 的 `1:*` 序列区间抓取草稿箱全部报文（不依赖 SEARCH 条件），解析仍带 `X-Schedule-Send` 且有收件人的草稿。
- **去重**：以「邮箱 + Message-ID」为键的内存注册表 + 每用户只扫描一次，避免重复登录或多个并发请求把同一封草稿挂成多个任务。
- **过期策略**：未到点按剩余时间等待；服务停机期间已过点的任务**立即补发**，日志会打印错过了多久。
- **安全边界**：任务执行前仍会重新读取草稿并校验 `X-Schedule-Send` 是否存在，因此撤销过的草稿不会被恢复出来误发。
- **已知限制**：用户完全不访问系统时不会触发恢复；注册表是进程内存（这一次重启后由草稿箱重建），因此不需要持久化任务表。

### 8.5 列表性能优化（实测）

排查方式：真实账号 + 进程内服务逐接口计时、IMAP 阶段拆解、同连接交替配对抓取，并用 4 个文件夹 54 封邮件做字段级快照比对。

问题定位：

- `MailList` / `StarMailList` 原来用 `BODY.PEEK[]` 抓**整封**报文，只为了取几行正文预览；INBOX 里一封 8.5MB 附件的邮件让列表接口稳定在 120–157ms。
- 单封对比：`BODY.PEEK[HEADER]` 0.6ms，`BODY.PEEK[]`（同样一封）33ms——附件字节和 MIME 解码是主要成本。

改动：

- `mailbox.Client` 新增 `FetchPreview(seqNums, maxBytes)`，用 `BODY.PEEK[]<0.N>` 只取报文前 64KB，并在同一次 FETCH 里带 `BODYSTRUCTURE`。
- 附件标记不再依赖下载附件内容：由 `BODYSTRUCTURE` 递归判断 `disposition=attachment`，`parseMailItem` 优先采用该结果；结构缺失时退回头部/分段扫描。
- 列表解析遇到截断导致的 MIME 解析失败，退回只解析邮件头，保证不丢邮件、元信息完整，预览正文置空。
- 详情、附件下载、定时任务恢复等需要完整报文的路径**仍然抓全量**，未受影响。
- `parseFlags` 结果改为排序输出，同一封邮件多次返回的 `flags` 顺序稳定（此前来自 map 遍历，顺序随机）。

实测结果（同连接交替配对）：

| 场景 | 优化前传输量 | 优化后传输量 | 优化前耗时 | 优化后耗时 |
| --- | --- | --- | --- | --- |
| INBOX（2 封，含 8.5MB 附件） | 8.98MB | 0.13MB | 36–44ms | 20–26ms |
| Deleted（50 封，22.9MB） | 22.9MB | 0.52MB | 98–112ms | 82–94ms |
| `POST /api/mail/list`（INBOX，n=5 平均） | — | — | 147ms | 39ms |

语义校验：优化前后对 `INBOX`/`Sent`/`Drafts`/`Deleted` 的 `uid/from/to/cc/subject/text/size/flags/has_attach/message_id` 逐行比对，除 `flags` 排序外完全一致；含附件的邮件 `has_attach` 保持为 true。

边界与剩余候选：

- 正文超过 64KB 的邮件，列表预览只覆盖前 64KB（列表本来就只展示摘要）。
- 详情接口的整封解析成本已单独优化（见 8.6），列表与详情都不再为预览/正文下载附件字节。
- 前端 `useAutoRefresh` 在页面不可见时跳过刷新，恢复可见后按原节奏继续，避免后台标签页每分钟请求列表与未读数（新增 3 个单测覆盖）。

### 8.6 详情接口性能优化（大附件，实测）

问题定位（真实账号，INBOX `uid=89`，报文 8,886,239 字节）：

- 原 `MailDetail` 先用 `BODY.PEEK[]` 抓**整封**报文，再 `enmime.ReadEnvelope` 解码全部部件：IMAP 抓取约 58ms、`enmime.ReadEnvelope` 80–95ms，单次分配约 14.1MB。
- 整封解析把 10 个附件（约 8.5MB）全部解码，只为算附件大小与列表；偶发的 878–1172ms 毛刺已定位到一次 GC（一次回收 40MB+ 堆，STW 约 1.0–1.2s），不是业务逻辑本身。
- 对照实验：单独取 `BODYSTRUCTURE` 仅 14–21ms，就能拿到全部 12 个部件的类型、编码、尺寸、`disposition`、文件名、`Content-ID`；只抓 `1.1`（纯文本）+ `1.2`（HTML）+ `2`（内联图）三个分段仅 16–18ms。

改动：

- `mailbox.Client` 新增 `FetchStructure(uid)` 与 `FetchSections(uid, sections)`：一次 FETCH 可组合多个 `BODY.PEEK[path]<offset.length>`，按 `path@offset` 归键（`go-imap` 为带偏移的分段只填 `Partial[0]`，不能用长度做键）。
- 新增 `service/mail_detail_parts.go`：`FetchStructure` → `buildDetailPlan` → 一次 `FetchSections` → 只解码正文与内联图 → 复用原有 `cid` 改写与落盘逻辑。
- 附件尺寸：≤4KB 的 base64 附件整段抓取；更大的只取头部 200B + 尾部 120B，按 base64 折行规则推算解码后字节数（10 个真实附件与逐字节解码结果**完全一致**）；推断失败才补抓整段。
- 结构不安全时自动回退原整封路径：目前只有**空 multipart**（服务端 `BODYSTRUCTURE` 本身解析失败）仍走整封解析；原先的 `message/rfc822` 内嵌邮件、未知传输编码、非 UTF-8/ASCII 正文、quoted-printable 附件、单段非文本根部件均已纳入分段路径。内存假实现返回 `ErrUnsupported`，因此原有单测仍覆盖回退路径。
- 特殊部件的对齐方式：非 UTF-8 正文和 quoted-printable 附件把「分段原始字节 + BODYSTRUCTURE 头信息」重建成单部件报文交回 `enmime` 解码，复刻其字符集探测/转换与 QP 软换行规则；`message/rfc822` 与 enmime 一致地按叶子部件处理（不展开、按附件尺寸计）；未知传输编码按服务端报告的编码尺寸计，不做解码。
- 单段邮件此前用空路径请求分段，会生成非法的 `BODY.PEEK[]<0.0>`；现已把单段根的 section 归一为 `1`（`BODY[1]`），并支持根即附件/内联内容（`PartID=0`，与 enmime 一致）。
- 内联图片已存在且大小一致时跳过重复写盘，反复打开同一封邮件不再反复写文件。

实测结果（同一封 8.89MB 邮件，连续 5 次）：

| 指标 | 优化前 | 优化后 |
| --- | --- | --- |
| 详情耗时 | 129–150ms（偶发 878–1172ms） | 33–49ms |
| IMAP 传输量 | 8.89MB | 约 0.10MB（正文 + 1 张内联图 + 附件首尾样本） |
| 解析方式 | 整封 enmime（分配约 14.1MB/次） | BODYSTRUCTURE + 分段，仅解码约 93KB |

分段内部耗时：`FetchStructure` 14–21ms、`FetchSections`（22 段）17–35ms，其余为解码、落盘与 URL 改写；复测未再出现 800ms 以上毛刺。

语义校验：对 `INBOX`/`Sent`/`Drafts`/`Deleted` 共 **57 封**邮件，用分段路径与整封路径逐字段比对（正文、附件列表每个字段、附件总大小），**差异 0 处、回退 0 封**。另构造 7 类特殊邮件覆盖原回退清单：GBK 正文、QP 正文、QP 附件（4056B 编码 / 1.5KB 解码）、`message/rfc822` 附件、未知传输编码附件、空 multipart、单段 PNG 根附件；其中 6 类走分段且与整封输出**完全一致**，空 multipart 仍回退整封。

边界与剩余候选：

- 空 multipart 仍走整封解析：hMailServer 对这类邮件返回的 `BODYSTRUCTURE` 无法解析（`Non-multipart body part doesn't have 7 fields`），分段路径拿不到结构，只能保留整封兜底。
- quoted-printable 附件必须整段下载才能精确算出解码后尺寸（QP 的软换行与 `=XX` 无法从首尾样本推断），因此这类邮件的传输量下降取决于同封是否还有其他大附件。
- 分段路径每次都重新取 `BODYSTRUCTURE`（14–21ms）；如需进一步压缩，可按「邮箱 + uid」缓存结构，但会引入失效判断。
- 附件尺寸依赖 base64 折行规则；极端非标准折行（如行宽 >512 或缺失换行）会判定失败并补抓整段，功能正确但退化到慢路径。

---

## 9. 常见问题

| 现象 | 原因与处理 |
| --- | --- |
| `ERR_PNPM_UNEXPECTED_STORE` | 用了 pnpm 11/12，改用 `corepack pnpm@10 ...` |
| `eslint .` 报 JS heap out of memory | 构建产物 `Web/` 被纳入 lint；确认 `eslint.config.js` 的 `globalIgnores` 含 `Web`、`node_modules`、`coverage` |
| 登录提示用户名或密码错误 | IMAP 993 未开放、账号不在 hMailServer 域内、或密码错误 |
| 用户管理接口返回“不支持” | 非 Windows 构建（`directory/hmail_other.go`），或 hMailServer COM 未注册/管理员密码错误 |
| 头像/内联图片无法显示 | `static.dir` 目录不存在或不可写；确认 `/static` 路由可达 |
| 生产环境接口 404 | `gin.mode=release` 时前缀默认 `/`，需显式设置 `api.prefix=/api` |
| 定时邮件未发出 | 服务进程需持续运行；重启后需用户发起一次带 JWT 的请求才会触发草稿箱恢复（见 8.4） |

---

## 10. 开发者约定

- 后端：新增能力先落到 `service`，`controller` 只做参数与响应；外部依赖一律经接口注入，便于 `internal/testutil` 替换。
- 前端：新接口先加到 `api/`，业务状态只放在 `useMailbox`，视图组件不直接发请求。
- 提交前跑齐：`go build ./... && go vet ./... && go test ./...`、`corepack pnpm@10 lint && corepack pnpm@10 test && corepack pnpm@10 build`。
- 样式相关改动必须逐字对照重构前快照（备份目录 `hmail-src-backup`），避免无意改变 UI。
