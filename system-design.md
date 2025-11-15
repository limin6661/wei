# Wechat2RSS 自建方案架构设计

> 约束：单实例（Zeabur 1CPU/1GB）、单用户、支持 500 个公众号、抓取成功率优先、允许慢速串行抓取、内容只需图文。

## 1. 技术栈

- **后端**：Go 1.22（Gin/Fiber + GORM + Playwright Go bindings），同进程内包含任务调度与抓取 worker。
- **前端**：Vue 3 + Vite + TypeScript + Pinia + Element Plus，SPA + 登录态管理。
- **数据库**：PostgreSQL（账号、任务、文章、日志、配置等）；初期不引入 Redis，后续可扩展。
- **浏览器引擎**：Playwright (WebKit 或精简 Chromium)，单实例 + 低并发复用。
- **部署**：Zeabur 单容器运行，提供 HTTP(S) 服务并暴露管理界面；静态资源由同一 Go 服务托管。

## 2. 总体架构

```
[前端 SPA] ←→ [Go API 层]
                     ├─ 鉴权模块（账号密码、微信状态）
                     ├─ 管理模块（公众号、任务、文章、配置）
                     ├─ 调度模块（任务入队、状态机）
                     ├─ 抓取 worker（控制 Playwright 浏览器）
                     ├─ RSS/API 输出
                     └─ 日志 & 告警
                          ↓
                    PostgreSQL（持久化）
```

说明：
1. **单进程内调度**：API 调度模块维护一个任务队列（内存 + DB 状态），串行执行抓取，确保内存使用可控。
2. **浏览器复用**：启动时创建 1 个 Playwright browser 实例，任务执行时复用上下文；每抓取 N 个任务或内存>阈值时重启实例。
3. **触发模型**：用户在前端“激活”某公众号后，Go 服务写入 `tasks`，调度器按 FIFO（或优先级）顺序执行；执行完成更新状态。

## 3. 核心模块

| 模块 | 功能点 |
| --- | --- |
| 鉴权模块 | 默认管理员账号密码，首次登录强制修改；维护 session/token；CSRF 防护。 |
| 微信登录模块 | 控制 Playwright 获取二维码、展示给前端；扫码后保存 cookie/凭证；监控失效并通知。 |
| 公众号管理 | 添加/编辑公众号（名称、原始 ID、备注、启用状态）；查看最近抓取结果与错误。 |
| 调度与队列 | 任务入队、状态机（待执行 → 执行中 → 成功/失败）；重试 3 次；串行或极低并发。 |
| 抓取 worker | 使用会话访问公众号文章列表，增量解析文章、提取正文/图片，存储为文章记录。 |
| RSS/API | 为每个公众号生成 RSS/JSON Feed；提供文章搜索接口；支持导出 OPML。 |
| 日志与报警 | 结构化日志（任务级别），供前端查看；异常（登录失效、连续失败）触发邮件/Webhook。 |

## 4. 数据流程

1. **登录阶段**：管理员访问管理后台 → 输入默认账号 → 强制修改密码 → 查看控制面板。
2. **绑定公众号**：
   - （首次）触发“登录微信” → 后端返回二维码 → 用户扫码 → 登录成功后保存凭证。
   - 前端填写公众号信息（原始 ID、昵称等）→ API 校验/保存。
3. **抓取**：
   - 用户点击“激活抓取”或定时任务触发 → `tasks` 表写入一条任务 → 调度器拾取。
   - Worker 控制浏览器请求公众号最近文章列表 → 对比数据库 → 新文章解析详情 → 入库 `articles`、`media`。
   - 抓取完成更新 `tasks` 状态并记录日志；失败则重试，每次记录错误详情。
4. **输出**：
   - 前端查看文章列表；RSS 访问 `/feed/{id}.xml`，由 API 实时生成或缓存。
5. **监控**：
   - 登录失效或任务连续失败 → 写入 `alerts` 表并发送通知；前端仪表盘展示状态。

## 5. 数据库草案

| 表 | 关键字段 | 说明 |
| --- | --- | --- |
| `users` | id, username, password_hash, force_reset, created_at | 管理员账户；初期仅一条。 |
| `wechat_sessions` | id, session_key, cookies, expires_at, status, last_ping | 微信登录会话。 |
| `accounts` | id, name, wechat_id, alias, status, last_task_id | 公众号管理。 |
| `tasks` | id, account_id, status, retry_count, started_at, finished_at, error | 抓取任务记录。 |
| `articles` | id, account_id, wechat_article_id, title, summary, content_html, published_at, raw_url | 抓取结果。 |
| `media` | id, article_id, type, url, local_path, hash | 图像等资源，若走直链则仅保存 URL。 |
| `feeds` | account_id, rss_token, last_generated_at, cached_path | RSS 访问控制。 |
| `logs` | id, level, module, message, context, created_at | 系统日志。 |
| `alerts` | id, type, status, payload, notified_at | 告警记录。 |

初期可以合并部分表，例如 `feeds` 可作为 `accounts` 的字段；`media` 如果采用直链亦可省略。

## 6. 抓取策略

- **队列与并发**：默认并发 1，队列长度不限，前端展示预计等待时间。后续可加入优先级（手动触发优先于定时）。
- **重试**：失败后等待 3 秒再重试，最多 3 次；记录每次错误信息。
- **成功率保障**：
  - 会话检测：任务开始前校验 cookie 是否有效，不足则提前提示重新扫码。
  - 频率控制：同账号抓取间隔>60 秒，避免触发限流。
  - 资源监控：检测浏览器内存，超阈值自动重启实例。
- **媒体处理**：默认直链微信 CDN，减少存储需求；如需缓存，可按配置将图片下载到本地并通过 `/media/*` 提供。

## 6.1 Chromium 集成计划

- **运行方式**：使用 Playwright Go 绑定驱动 **Chromium Headless**，单实例复用 `browserContext`；每执行 N 个任务或内存超阈值即重启浏览器。
- **配置项**：
  - `CHROMIUM_PATH`：Chromium 可执行文件路径，Zeabur 部署时通过镜像内置或运行时下载。
  - `CRAWLER_CONCURRENCY`：并发量（默认 1）。
  - `TASK_POLL_INTERVAL`：轮询任务间隔（秒）。
- **生命周期**：
  1. 服务启动 → `crawler.Manager` 创建单例浏览器，预热上下文。
  2. 轮询 `tasks` 表取 `pending` 任务 → 设置 `status=running` → 派发到 worker。
  3. Worker 复用 Chromium 实例加载公众号页面（注入必要 UA/Cookie），提取文章数据。
  4. 成功后写入 `articles` 并将 `tasks` 标记为 `success`；失败时写 `error_msg`，并按策略重试。
- **资源控制**：为节省 1GB 内存，Chromium 启动参数将启用 `--disable-dev-shm-usage --single-process --memory-pressure-off` 等，必要时对任务执行时间设置超时并在超时后销毁上下文。
- **后续工作**：在迭代 2 中引入 Playwright 依赖、实现二维码登录流程、编写 worker 逻辑。

## 7. 安全与权限

- 管理后台登录保护（默认用户名/密码 + 首次修改），后续可增加双因素。
- HTTPS（Zeabur 证书）+ CSRF Token + JWT/Session。
- 抓取流程中对外接口仅暴露必要 API，RSS 可设置私有 token。

## 8. 日志与告警

- 任务日志：入库并通过 Web 控制台查看，字段包含任务 ID、账号、耗时、结果、错误。
- 系统日志：Go 服务输出 JSON 格式，便于在 Zeabur 控制台过滤。
- 告警：登录失效、连续失败、浏览器异常等事件写入 `alerts`，并触发邮件/Webhook（SMTP/第三方服务配置项）。

## 9. 迭代计划

1. **迭代 0：基础搭建**
   - 初始化 Go + Vue 项目结构，搭建 PostgreSQL schema，完成账号登录、基础 API。
2. **迭代 1：微信登录 & 公众号管理**
   - 集成 Playwright，完成扫码登录流程；实现公众号 CRUD、任务入队。
3. **迭代 2：抓取 worker & RSS 输出**
   - 完成抓取任务执行、增量存储、RSS/JSON Feed 生成；前端展示文章与日志。
4. **迭代 3：稳定性与告警**
   - 加入重试、监控、告警、日志查看；优化浏览器资源管理。
5. **迭代 4：调优与扩展**
   - 性能调优、界面完善、预留付费控制字段、考虑多实例/worker 方案。

以上为第一版设计，可在实现过程中根据实际测试结果进一步调整。
