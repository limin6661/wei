# Wechat2RSS 私有化部署

这是一个基于 Go + Vue + PostgreSQL 的微信公众号历史文章抓取与 RSS 输出服务。它通过微信公众号后台的扫码登录方式获取授权，自动调度抓取任务、落库文章正文，并提供 Web 控制台与 RSS 导出能力。

## 目录结构

- `server/`：Go 后端（Gin + GORM），负责账号/会话管理、任务调度、文章存储、RSS 输出。
- `web/`：Vue3 + Vite 控制台，提供二维码扫码、公众号绑定、任务/文章查看等页面。
- `system-design.md`：架构与迭代规划文档。
- `Dockerfile` / `.dockerignore`：多阶段构建镜像，内置前后端。

旧版的 VitePress 文档、静态资源以及脚本已移除，以免混淆当前实现。

## 本地开发

1. **准备依赖**
   - Go 1.20+、Node.js 18+、PostgreSQL。
   - PostgreSQL 中新建数据库，并拿到连接串。

2. **启动后端**
   ```bash
   cd server
   cp .env.example .env   # 设置 DATABASE_URL、SESSION_SECRET 等
   go run ./cmd/server
   ```
   首次启动会创建 `ADMIN_USER/ADMIN_PASSWORD` 对应的管理员账号；登录后需调用 `/api/password` 修改密码后再使用其他接口。

3. **启动前端**
   ```bash
   cd web
   npm install
   npm run dev
   ```
   默认通过 Vite 代理访问 `http://localhost:8080/api`。

4. **测试流程**
   - 登录 Web 控制台 → 修改密码。
   - 在「微信会话」页生成二维码，用手机微信扫描并确认登录，等待状态变为 `active`。
   - 在「公众号」页选择该会话，搜索公众号拿到 BizID 并保存。
   - 在「任务」页触发一次抓取，查看任务/日志状态。
   - 在「文章」页确认抓取结果，并在 RSS 阅读器中订阅 `http://<服务器>/feed/{accountID}`。

更多 API 说明见 `server/README.md`。

## Docker 部署

已提供多阶段 Dockerfile，会自动：
1. 构建 `web` 前端并生成静态资源；
2. 构建 Go 后端二进制；
3. 在精简运行镜像中托管二者（默认静态目录 `/app/static`）。

```bash
docker build -t wechat2rss .
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/wechat2rss?sslmode=disable" \
  -e SESSION_SECRET="your-secret" \
  wechat2rss
```

常用环境变量：
- `APP_PORT`（默认 8080）
- `DATABASE_URL`
- `SESSION_SECRET`
- `ADMIN_USER` / `ADMIN_PASSWORD`
- `CRAWLER_CONCURRENCY`
- `TASK_POLL_INTERVAL`
- `WEB_STATIC_DIR`（Docker 镜像默认 `/app/static`）

部署到 Zeabur、Railway 等平台时，只需提供 PostgreSQL 实例和上述环境变量即可。若要分离前后端，也可以单独部署 `web/dist` 为静态站点，通过反向代理将 `/api` 指向 `server`。

## 注意事项

- 仅通过公众号后台提供的接口抓取文章，若账号登录失效需重新扫码。
- 抓取频率建议适当限制，避免触发风控。
- 项目默认不内置代理，请根据自身网络情况选择是否配置。

欢迎按照自己的需求二次开发或提交 PR。***
