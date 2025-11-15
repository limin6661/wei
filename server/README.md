## Wechat2RSS 服务（后端）

Go + Gin + PostgreSQL 实现的私有化部署服务。当前特性：

- 管理后台登录、首次强制改密。
- 公众号维护（名称、原始 ID、BizID、绑定会话）。
- 微信公众号后台扫码登录，会话状态自动轮询，成功后保存 cookie/token。
- 基于公众号后台 API 的任务调度：入队、状态机、失败重试、执行日志。
- 抓取历史文章（标题、摘要、正文 HTML），写入数据库。
- 提供 REST API + RSS 导出。

### 快速启动

```bash
cd server
cp .env.example .env   # 填写 DATABASE_URL 等
go run ./cmd/server
```

首次运行会自动创建 `ADMIN_USER/ADMIN_PASSWORD` 对应的管理员账号；登录后必须执行 `/api/password` 修改密码，之后才能访问其余接口。

### 主要环境变量

- `DATABASE_URL`：PostgreSQL 连接。
- `SESSION_SECRET`：Session 加密。
- `ADMIN_USER` / `ADMIN_PASSWORD`：默认账号。
- `CHROMIUM_PATH`：保留字段，后续用于 Playwright；当前 HTTP 抓取不依赖。
- `CRAWLER_CONCURRENCY`：任务并发数（默认 1）。
- `TASK_POLL_INTERVAL`：任务轮询间隔，单位秒（默认 5）。
- `WEB_STATIC_DIR`：可选，指向前端构建产物目录（Docker 镜像默认 `/app/static`），配置后由 Go 服务托管 SPA。

### 核心 API

- `POST /api/login`、`POST /api/logout`、`GET /api/me`、`POST /api/password`：账户登录及管理。
- `GET/POST/PUT/DELETE /api/accounts`：公众号维护（支持设置 BizID、绑定会话）。
- `POST /api/accounts/:id/tasks`：创建抓取任务。
- `GET /api/tasks`、`GET /api/tasks/:id/logs`：查看任务与执行日志。
- `GET /api/wechat/sessions`、`POST /api/wechat/sessions`、`GET /api/wechat/sessions/:id`：创建并查看公众号后台扫码登录会话。
- `GET /api/wechat/search?session_id=..&query=..`：使用指定活跃会话搜索公众号，获取 FakeID/BizID。
- `GET /api/accounts/:id/articles`：查看某个账号已抓取文章。
- `GET /feed/:id`：输出指定账号的 RSS（最近 50 篇）。

### 抓取与日志

后台 `crawler.Manager` 会根据 `TASK_POLL_INTERVAL` 轮询 `pending` 任务，并尊重 `CRAWLER_CONCURRENCY` 控制并发。执行流程：

1. 获取任务 → 状态改为 `running`。
2. 读取账号对应的 Session（需为 `active` 状态）和 BizID。
3. 通过公众号后台接口 `searchbiz`/`appmsg` 拉取历史文章，逐条持久化，正文通过公共链接解析 `#js_content`。
4. 成功写入 → 任务标记 `success`；遇到错误记录日志并重试，最多 3 次，之后记为 `failed`。

可通过 `GET /api/tasks/:id/logs` 查看“任务开始”“任务成功”“错误信息”等记录。

### RSS

`GET /feed/:accountID` 返回简单的 RSS 2.0（最近 50 条）。部署到 Zeabur 或其他平台时，请确保外部可访问该路径，以便订阅器读取。

---

前端界面位于 `/web` 目录，使用 Vue3 + Vite + Pinia，对上述 API 做了基础封装：登录、公众号管理、会话二维码展示、任务/日志列表、文章 & RSS 查看等。构建方式：

```bash
cd web
npm install
npm run build
```

默认 `vite.config.ts` 已配置代理到本地 `http://localhost:8080`。
