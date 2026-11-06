# email-web

hMailServer Web 邮件客户端前端。React 19 + rolldown-vite + Arco Design + Tailwind CSS v4。

> 重构原则：**视觉与交互零变化**。className、Arco 组件参数、中文文案、图标资源均与重构前逐字一致，本次只重构代码分层与数据流。

## 1. 技术栈

| 类别 | 选型 |
| --- | --- |
| 框架 | React 19、React Router 7 |
| 构建 | rolldown-vite 7（`vite` 指向 `npm:rolldown-vite@7.2.5`） |
| UI | Arco Design 2.66、Tailwind CSS 4、Sass |
| 请求 | axios（`src/api/request.js` 统一实例） |
| 状态 | React Context（`MailProvider`）+ 自定义 hook |
| 测试 | Vitest 5 + jsdom + Testing Library |
| 图标 | `vite-plugin-svgr`（`*.svg` 作为组件导入） |

## 2. 目录

```
src/
├── api/
│   ├── request.js        # axios 实例：baseURL、token、统一错误提示
│   ├── auth.js           # 登录
│   ├── user.js           # 改密、账号增删改查、头像上传
│   ├── contact.js        # 联系人列表/保存/删除/清空
│   └── mail.js           # 列表/星标/未读/详情/状态/移动/删除/草稿/发送/取消定时/下载
├── constants/mail.jsx    # 文件夹、分页、菜单、移动目标、筛选、标记、图标
├── hooks/
│   ├── useMailbox.jsx    # 邮箱状态机（目录/列表/筛选/详情/写信/联系人/标记/滚动分页）
│   └── useAutoRefresh.js # 自动刷新调度（1min → 5min 退避）
├── utils/
│   ├── index.jsx         # 文件类型、树展开、时间格式化、属性互换、SVG 判断、节流
│   └── mail.js           # 内联图片、详情标准化、引用块、FormData、分组、分页合并
├── pages/
│   ├── Mail/
│   │   ├── index.jsx     # 薄壳：取上下文 → useMailbox → Layout/Menu/Content 或写信页
│   │   ├── MailContext.jsx
│   │   ├── Menu.jsx      # 左侧目录 + 写信按钮
│   │   ├── List.jsx      # 邮件列表（分组、多选、右键操作）
│   │   ├── Content.jsx   # 列表 + 详情的可拖拽分栏
│   │   └── Detail.jsx    # 详情、附件、回复转发、定时提示
│   ├── Personal.jsx      # 个人中心（信息、改密、头像）
│   └── User.jsx          # 用户管理
├── views/
│   ├── Home.jsx          # 布局与搜索框，向下传递 searchSeq
│   ├── Login.jsx
│   └── Preview.jsx       # 附件预览（pdf/图片/office 等）
├── components/
│   ├── WriteMail.jsx     # 写信/回复/转发/草稿编辑
│   ├── Editor/           # 富文本编辑器（Arco Editor 定制 + 颜色/字号选择）
│   └── AvatarImage.jsx   # 头像（失败回退首字母）
├── router.jsx            # basename=/web
├── main.jsx
└── test/setup.js         # jest-dom、cleanup、ResizeObserver/matchMedia 兜底
```

## 3. 数据流

```
Home（搜索框、用户信息）
  ├─ searchSeq 递增 ────────────────┐
  └─ <Outlet context> ─────────────┤
                                    ▼
pages/Mail/index.jsx ── useMailbox(searchSeq) ── mailContext
                                    │
        ┌───────────────┬───────────┴───────────┬──────────────┐
      Menu            List                  Content          Detail
        └───────────────┴───────────┬───────────┴──────────────┘
                                    ▼
                          api/* → axios → 后端 /api
```

要点：

- 搜索采用 `searchSeq` 单向通知，替代旧的 `registerMethod`（父组件反向调用子组件方法）。
- 列表刷新统一走 `mergeMailPage`：`page === 1` 替换，否则追加，避免闭包中的旧列表覆盖新数据。
- 自动刷新：每次成功加载后 `schedule()` 重新计时；前 20 次自动刷新间隔 1 分钟，之后退避到 5 分钟，累计 50 次后停止。页面不可见（`document.hidden`）时跳过本轮请求，恢复可见后按原节奏继续，避免后台标签页持续请求列表与未读数。
- 写信页与目录互斥：打开写信时新增虚拟目录 `compose`，关闭后回到来源目录。

## 4. 环境变量（`.env`）

```ini
VITE_BASE_URL=http://localhost:8058
VITE_SUFFIX=@jin337.top
VITE_PAGE_TITLE=华盛邮件
```

`vite.config.js`：`base: '/web'`、开发端口 `4055`、`outDir: 'Web'`。

## 5. 开发与构建

```bash
corepack pnpm@10 install
corepack pnpm@10 dev        # http://localhost:4055/web
corepack pnpm@10 build      # 产物 Web/
corepack pnpm@10 preview
```

> 必须使用 `pnpm@10`：仓库 store 为 `D:\.pnpm-store\v10`，pnpm 11/12 会报 `ERR_PNPM_UNEXPECTED_STORE`。

## 6. 测试与静态检查

```bash
corepack pnpm@10 test       # vitest run（7 文件 / 44 用例）
corepack pnpm@10 test:watch
corepack pnpm@10 lint       # eslint .
```

覆盖范围：

| 文件 | 关注点 |
| --- | --- |
| `utils/index.test.jsx` | 文件类型、树展开、时间格式、属性互换、SVG、节流（含尾调用参数） |
| `utils/mail.test.js` | 内联图 id、详情标准化、时间分组、分页合并、引用块、FormData |
| `hooks/useMailbox.test.jsx` | 初始化、搜索序号、标记已读、星标取消、写信开关、发送、必填校验 |
| `pages/Mail/Menu.test.jsx` | 目录与未读数、写信入口、切换目录清理搜索/选中 |
| `pages/Mail/List.test.jsx` | 列表渲染、未读加粗与选中样式（`selelct-mail`）、空态 |
| `pages/Mail/Detail.test.jsx` | 正文/附件、定时提示、未选中空态 |

组件测试对关键 className 做断言，用于防止重构时样式漂移。

## 7. 样式约束（修改前必读）

以下均为**刻意保留**的现状，不得“修正”：

- 拼写：`selelct-mail`（选中态类名）、资源 `file_aduio.svg`。
- 关键类名：`mail-item`、`mail-list`、`mail-detail`、`mail-menu`、`mail-contacts`、`editor-wrap`、`editor-contact`、`no-footer`、`resizebox-custom-trigger-line`、`focus-box`、`top-card`、`total`、`simpleModal`。
- Tailwind 任意值/重要修饰：`pr-4!`、`bg-transparent!`、`shadow-none!`、`text-(--color-text-1)!`、`hover:bg-(--color-fill-2)`、`h-[calc(100vh-116px)]`、`h-[calc(100vh-117px)]`、`w-[calc(100%-436px)]`、`w-[calc(100%-72px)]`、`w-[calc(100%-30px)]`、`text-[140px]!`、`leading-7!`、`min-w-10!`。
- Arco 参数：`Layout.Sider width={220}`；`Content` 的 `resizeBoxProps`（`directions:['right']`、`onMoving` 740 阈值切换 `isMove/isTable`、宽度三态 `100% / 0 / 410px`、`minWidth 350px/auto`）；`InputTag maxTagCount={5} saveOnBlur labelInValue`；`Upload autoUpload={false} multiple tip="（上传附件不得超过20M）"`；`Editor height='auto'`；`Tree blockNode autoExpandParent`；`Checkbox.useCheckbox`。
- 中文文案与图标文件保持原样（如“暂无邮件待处理”“请选择非星标邮件”“已移入垃圾箱”等）。

## 8. 排障

| 现象 | 处理 |
| --- | --- |
| `ERR_PNPM_UNEXPECTED_STORE` | 使用 `corepack pnpm@10 ...` |
| `eslint .` OOM | 确认 `eslint.config.js` 忽略 `Web`、`node_modules`、`coverage` |
| 开发环境接口 404 | 检查 `.env` 的 `VITE_BASE_URL`；后端 debug 模式前缀为 `/api` |
| 生产环境白屏 | 静态服务器需把 `/web/*` 回退到 `/web/index.html`（SPA 路由） |
| 附件预览空白 | 预览依赖 `jit-viewer` 与浏览器能力（如 3D/Office 预览），先确认文件类型是否在支持范围内 |
