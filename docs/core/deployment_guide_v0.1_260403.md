# StepUp 部署指南 v0.1

**日期**: 2026-04-03  
**适用范围**: 测试环境、内网联调、小规模预览；生产首发前请结合安全清单加固。

**相关文档**:

- [**文档索引与阅读顺序**](../README.md)
- [**v0.1 增量：部署与升级说明**](../releases/DEPLOY_AND_UPGRADE_v0.1_260404.md)（含 AI 调用日志、SQL 目录调整等）
- [架构与部署设计](./architecture_deployment_v0.1_260403.md)
- [MySQL 建表脚本](../../db/schema/mysql_schema_v0.1_260403.sql)（**所有 SQL 见仓库 [`db/`](../../db/README.md)**）
- [API 文档](./api_v0.1_260403.md)
- 后端运行说明：仓库内 [`backend/README.md`](../../backend/README.md)

---

## 1. 组件与端口

| 组件 | 说明 | 默认端口（Compose） |
|------|------|---------------------|
| `mysql` | MySQL 8.4 | `3306` |
| `backend` | Go API | `8080` |
| `frontend-student` | 学生端静态站点 | `3000` |
| `frontend-admin` | 管理端静态站点 | `3001` |
| `mock-ai` | 本地占位分析服务（可选） | `8090` |

对外暴露时建议用 **Nginx / 云负载均衡** 做 HTTPS 与路由，不直接裸奔公网 8080。

---

## 2. 前置条件

- 已安装 **Docker** 与 **Docker Compose**，或自备 **MySQL 8** + 可运行 Linux/amd64 二进制（或自编译）的环境。
- 仓库 **clone** 到目标机，并检出需要发布的 **commit / tag**。

---

## 3. 方式 A：Docker Compose（推荐测试 / 预览）

### 3.1 配置环境变量

在仓库根目录：

```bash
cp .env.example .env
```

按环境编辑 `.env`（至少检查 **`MYSQL_*`**、**`ADMIN_BOOTSTRAP_*`**、**`CORS_ALLOWED_ORIGINS`**，见 §5）。

**测试 / QA**：可使用仓库内模板 **`.env.qa`**（占位密码需自行替换），启动时：

```bash
docker compose --env-file .env.qa up -d --build
```

若在测试机 **直接运行 Go 二进制**（`go run` / 编译后的 `server`），请用 **`backend/.env.qa`** 配置 `DB_DSN` 等（与根目录 `.env.qa` 中数据库账号可保持一致），详见 [`backend/README.md`](../../backend/README.md)。

### 3.2 启动

```bash
docker compose up -d --build
```

Compose 中后端默认 **`ANALYSIS_ADAPTER=http`**（见根目录 `docker-compose.yml`）。未导入数据库前，部分接口会异常；完成 §3.3 后再验全链路。

### 3.3 初始化数据库（首次或空库）

在 **MySQL 已就绪** 后执行（变量与 `.env` 一致时可直接复制）：

```bash
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < db/schema/mysql_schema_v0.1_260403.sql
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < db/seed/dev_seed.sql
```

**若库在引入 `ai_call_log` 之前已建成**：补执行增量脚本（仅新建表，`IF NOT EXISTS` 可重复执行）：

```bash
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < db/migrations/20260404_ai_call_log.sql
```

说明见 [`ai_model_log_v0.1_260403.md`](./ai_model_log_v0.1_260403.md) 与 [部署与升级说明](../releases/DEPLOY_AND_UPGRADE_v0.1_260404.md) §3。

**种子数据说明**（`db/seed/dev_seed.sql`）：

- 默认管理员（bcrypt）、阶段、科目、`ai_model` 中的 DeepSeek 占位行等。
- **`app_secret` 为占位符** `__REPLACE_WITH_DEEPSEEK_API_KEY__`：执行前在本地替换为真实 Key，或在导入后用管理端 **AI 模型** 界面更新；**不要把真实密钥提交到 Git**。

若库中 **已存在** 同名 DeepSeek 种子行，`INSERT ... WHERE NOT EXISTS` 不会重复插入；需改密钥时用 SQL `UPDATE` 或管理端。

### 3.4 健康检查

```bash
curl -sS "http://localhost:${BACKEND_PORT:-8080}/healthz"
curl -sS "http://localhost:${BACKEND_PORT:-8080}/readyz"
```

`readyz` 在配置了 `DB_DSN` 且数据库不可达时返回 `503`，编排系统可据此摘流或重启。

### 3.5 访问前端

- **整合入口（推荐）**：后端镜像内已打包静态页，与 API 同端口：
  - 管理端：`http://<主机>:${BACKEND_PORT:-8080}/admin/`
  - 学生端：`http://<主机>:${BACKEND_PORT:-8080}/student/`
  - `GET /` 返回 JSON，并可查看其中 `ui` 字段的快捷路径。
- **独立 nginx 容器**（Compose 默认仍启动）：`http://<主机>:${STUDENT_PORT:-3000}`、`http://<主机>:${ADMIN_PORT:-3001}`。
- API：`http://<主机>:${BACKEND_PORT:-8080}/api/v1/...`

同域访问 `/admin/`、`/student/` 时一般无 CORS 问题；若仍使用独立端口前端，后端的 **`CORS_ALLOWED_ORIGINS`** 必须包含对应 **Origin**（见根目录 `.env.example` 与 §5）。

### 3.6 同一台机：学生 / 管理 / API 分端口（示例 7010 / 7011 / 7012）

典型拓扑：**学生静态** `7010`，**管理静态** `7011`，**Nginx 对外 API** `7012` → `proxy_pass http://127.0.0.1:8080`（Go 监听本机 8080）。浏览器里学生页、管理页的 **Origin** 是 `:7010`、`:7011`，请求 API 须打到 `:7012`，不能再用页面 `location.origin`。

1. **`CORS_ALLOWED_ORIGINS`**（逗号分隔）必须包含学生页与管理页的完整 Origin，例如：  
   `http://<主机名或 IP>:7010,http://<主机名或 IP>:7011`  
   （一般**不必**写 `:7012`，除非有页面也挂在 API 同一端口。）
2. **前端**：`app.js` 内约定 **页面端口 `7010` / `7011` 时自动指向同主机 API 端口 `7012`**，**无需改 `index.html`**。若你使用其它端口组合，可选用任选其一覆盖：`?api=`、`localStorage`、`meta name="stepup-api-base"` / `stepup-api-port`，或管理端登录框填 API 端口。
3. **Nginx**（片段示意，路径需完整转发到后端，含 `/api/`、`/healthz` 等按需）：

```nginx
server {
    listen 7012;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 4. 方式 B：仅后端 + 自建 MySQL（无 Compose）

适用于已有 MySQL、单独部署 API 的场景。

1. 在 MySQL 中创建库与用户，执行 `db/schema/mysql_schema_v0.1_260403.sql` 与（可选）`db/seed/dev_seed.sql`。
2. 设置环境变量 **`DB_DSN`**，例如：  
   `user:pass@tcp(数据库主机:3306)/stepup?charset=utf8mb4&parseTime=True&loc=Local`
3. 本地构建并运行：

```bash
cd /path/to/stepup
go build -o stepup-server ./backend/cmd/server
export DB_DSN='...'
export ANALYSIS_ADAPTER=http   # 若需走 HTTP / 库内模型
export CORS_ALLOWED_ORIGINS='https://app.example.com,https://admin.example.com'
./stepup-server
```

或使用根目录 / `backend/Dockerfile` 构建镜像，在编排中注入相同环境变量。

---

## 5. 环境变量清单（部署必查）

根目录 `.env.example` 与 `docker-compose.yml` 中后端服务可见变量如下（节选）。

| 变量 | 说明 |
|------|------|
| `APP_ENV` | `dev` / `staging` / `prod` 等，按规范自取 |
| `MYSQL_*` | 数据库库名、用户、密码、映射端口（Compose） |
| `BACKEND_PORT` / `STUDENT_PORT` / `ADMIN_PORT` | 宿主机映射端口 |
| `DB_DSN` | 非 Compose 时必填；Compose 内由 compose 拼装注入容器 |
| `ANALYSIS_ADAPTER` | `http`：走 HTTP 适配器；`mock`：不调用外部分析 URL |
| `AI_ENDPOINT` | `ANALYSIS_ADAPTER=http` 时，**无可用库内激活模型 URL** 时的回退地址；可与 `mock-ai` 对接 |
| `AI_REQUEST_TIMEOUT_SECONDS` | 分析 HTTP 超时 |
| `ADMIN_BOOTSTRAP_USERNAME` / `ADMIN_BOOTSTRAP_PASSWORD` | 首次 bootstrap 管理员；**测试环境务必改为强密码** |
| `ADMIN_SESSION_TTL_HOURS` | 管理端会话时长 |
| `CORS_ALLOWED_ORIGINS` | **逗号分隔**的前端 Origin 白名单；缺省常为 localhost，上线 **必须** 改为真实域名 |

**分析行为**（`ANALYSIS_ADAPTER=http`）：优先使用 **`ai_model` 中 `status=1` 且未删除** 的最新一条的 **`url`**；若该行 **`app_secret` 非空**，则按 **OpenAI 兼容 `chat/completions`** 调用；`app_secret` 为空则按项目 **mock-ai** JSON 协议请求（适合 `AI_ENDPOINT` 指向 `mock-ai`）。详见 [`backend/README.md`](../../backend/README.md)。

Compose 未集中列出 `CORS_ALLOWED_ORIGINS` 时，后端使用代码中的默认值；**跨域名部署** 请在 `docker-compose.yml` 的 `backend.environment` 中增加该变量，或通过扩展 `docker-compose.override.yml` 注入。

---

## 6. 测试环境与生产差异（建议）

| 项目 | 测试 / 预览 | 生产建议 |
|------|-------------|-----------|
| HTTPS | 可 HTTP 内网 | 全站 HTTPS，证书自动续期 |
| 密钥 | `.env` 或本地改 seed 后导入 | 密钥管理系统 / 编排 Secret，**禁止** 写进仓库 |
| 默认口令 | 可弱口令仅限内网 | 强密码 + 禁用默认账号或 MFA |
| CORS | 可放宽容错 | 严格白名单 |
| 限流 | 可选 | 登录、验证码等接口限流 |
| 备份 | 可选 | MySQL 周期备份与恢复演练 |

更完整的安全与观测要求见 [架构与部署设计](./architecture_deployment_v0.1_260403.md) §5、§10。

---

## 7. 升级与回滚（简要）

1. 拉取新镜像或新二进制，**先备份数据库**。
2. 若有新 DDL，在维护窗口执行对应迁移脚本（当前仓库以 `db/schema/mysql_schema_v0.1_260403.sql` 为基线；增量见 `db/migrations/`；后续若引入迁移工具，以工具版本为准）。
3. 滚动重启 `backend` → 验证 `readyz` 与核心业务路径。
4. 异常时回滚上一个镜像/二进制版本，必要时恢复数据库备份。

---

## 8. 常见问题

**Q：`readyz` 503 `DATABASE_UNAVAILABLE`？**  
A：后端未连上 MySQL，检查 `DB_DSN`、网络、账号权限与防火墙。

**Q：前端报 CORS 错误（`No 'Access-Control-Allow-Origin' header`）？**  
A：将 **页面** 的 Origin（地址栏「协议 + 主机 + 端口」，无路径、无末尾 `/`）逐字加入 **`CORS_ALLOWED_ORIGINS`**（逗号分隔），**重启 backend** 使环境变量生效。学生静态 `:7010`、管理 `:7011`、API 经 Nginx `:7012` 时，白名单要写 **`http://…:7010` 与 `http://…:7011`**，一般不必写 `:7012`。若仍失败，用浏览器开发工具看请求是否到达 Go（Nginx 对 4xx/5xx 的响应有时不带 CORS 头，需先排除上游错误或 OPTIONS 未转发）。

**Q：试卷分析总是 mock 结果？**  
A：确认 **`ANALYSIS_ADAPTER=http`**、库内有 **激活** `ai_model` 且 **URL 可解析**；若用真实 LLM，确认 **`app_secret` 已配置** 且能访问公网 API；失败时实现会回退 mock，可查后端日志与网络。

**Q：种子里的 DeepSeek 不生效？**  
A：确认已执行 `db/seed/dev_seed.sql` 且 `app_secret` 不是占位符；或 `WHERE NOT EXISTS` 已跳过插入，需 `UPDATE ai_model ...`。

---

## 9. 文档版本

- v0.1：与当前仓库 Compose、后端行为对齐；后续迭代请更新本文日期与章节。
