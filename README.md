# hmail 邮件服务系统

基于 Go + Gin + IMAP/SMTP 开发的邮件服务系统，适配 `hMailServer` 自建邮件服务器。包含后端服务和前端界面，提供完整的邮件管理功能。

## 项目简介

### 后端 (email-server)

1. JWT 登录认证，Token 内置邮箱账号密码，业务接口无需重复传参
2. 支持发送邮件、抄送、多文件附件上传(20M 限制)
3. 自动归档已发送邮件到 Sent 文件夹
4. 支持读取任意邮箱文件夹邮件、分页查询
5. 邮件查看正文、移动文件夹、彻底删除
6. 附件在线流式下载
7. 规范分层架构：配置层、中间件、控制器、业务服务
8. 支持 `air` 热重载开发，无需频繁重启服务
9. 用户管理功能：创建、删除、更新用户信息

### 前端 (email-web)

1. 基于 React + Vite 构建的现代化邮件客户端
2. 响应式设计，多模式查看
3. 直观的邮件列表、详情和撰写界面
4. 与后端 API 无缝集成

## 项目结构

```
hmail/
├── email-server/          # 后端服务
│   ├── config/           # 配置管理
│   ├── controller/       # 控制器层
│   ├── middleware/       # 中间件
│   ├── model/            # 数据模型
│   ├── router/           # 路由配置
│   ├── service/          # 业务逻辑层
│   ├── utils/            # 工具函数
│   ├── .air.toml         # air热重载配置
│   ├── .env              # 环境变量
│   ├── go.mod            # 依赖管理
│   ├── go.sum
│   ├── main.go           # 项目入口
│   └── README.MD         # 后端文档
├── email-web/            # 前端应用
│   ├── public/           # 静态资源
│   ├── src/              # 源代码
│   │   ├── api/          # API 请求
│   │   ├── assets/       # 静态资源
│   │   ├── views/        # 页面组件
│   │   ├── App.jsx       # 主应用组件
│   │   ├── index.css     # 全局样式
│   │   ├── main.jsx      # 入口文件
│   │   └── router.jsx    # 路由配置
│   ├── .env              # 环境变量
│   ├── package.json      # 依赖配置
│   ├── vite.config.js    # Vite 配置
│   └── README.md         # 前端文档
├── .gitignore
└── README.md             # 项目总览文档
```

## 环境依赖

- Go >= 1.21
- Node.js >= 20
- 本地搭建 `hMailServer` 邮件服务
- 开启端口：
  - SMTP 25
  - IMAP 993

## 快速部署

### 后端部署

#### 1. 安装依赖

```bash
cd email-server
go mod tidy
```

#### 2. 安装 air 热重载

```bash
go install github.com/cosmtrek/air@latest
```

#### 3. 修改配置

打开 `config/config.go` 修改为你的邮件服务器地址

```go
var (
	HmailHost = "127.0.0.1"
	SmtpPort  = 25
	ImapPort  = 143
	AdminPwd  = "hMailServer密码"
)
```

#### 4. 启动项目

```bash
air
```

默认运行端口：`8058`

### 前端部署

#### 1. 安装依赖

```bash
cd email-web
npm install
# 或使用 pnpm
pnpm install
```

#### 2. 启动开发服务器

```bash
npm run dev
# 或
pnpm dev
```

#### 3. 构建生产版本

```bash
npm run build
# 或
pnpm build
```

## 全局规则

1. 所有接口**统一 POST**
2. 除登录接口外，所有接口必须携带 Token
3. 请求头格式`Token: 你的登录token`
4. 传参格式：`JSON` 或 `form-data`（文件上传时）

## 技术栈

### 后端

- **框架**: Gin Web Framework
- **认证**: JWT (golang-jwt/jwt)
- **邮件协议**: IMAP, SMTP
- **热重载**: air
- **跨域**: gin-contrib/cors

### 前端

- **框架**: React 19
- **构建工具**: Vite
- **路由**: React Router DOM
- **HTTP客户端**: Axios
- **UI库**: Ant Design
- **代码规范**: ESLint + Prettier

## 开发说明

### 后端开发

- 使用 `air` 进行热重载开发
- 遵循 MVC 架构模式
- 所有业务逻辑在 service 层实现
- 控制器负责参数验证和响应格式化

### 前端开发

- 组件化开发，复用性强
- 统一的 API 请求封装
- 响应式设计，多模式查看
- 使用现代 CSS 特性提升用户体验

## 注意事项

1. 确保 hMailServer 服务正常运行
2. 检查防火墙设置，允许相应端口通信
3. 生产环境请修改默认密钥和管理员密码
4. 定期备份重要邮件数据
5. 注意邮件附件大小限制（默认20MB）
