# StepUp Backend Scaffold (v0.1)

发版与数据库升级见仓库 [**docs/README.md**](../docs/README.md)、[**docs/releases/20260404#01_DEPLOY_AND_UPGRADE.md**](../docs/releases/20260404%2301_DEPLOY_AND_UPGRADE.md)。

## Run

From repository root:

```bash
go run ./backend/cmd/server
```

Server default address: `0.0.0.0:8080`

**不用 Docker、前端由本机 Nginx 提供时**：Go 只负责 API（例如本机 `8080`），Nginx 开 `7010`/`7011` 静态 + `7012` 反代到 `127.0.0.1:8080`，须让 **OPTIONS 预检** 进到 Go（或由 Nginx 完整返回 CORS 头）。示例配置见 **[`docs/deploy/nginx_go_static_split_ports.conf.example`](../docs/deploy/nginx_go_static_split_ports.conf.example)**，说明见 [**部署指南** §3.7](../docs/core/deployment_guide_v0.1_260403.md)。

仓库内 **`frontend-admin`**、**`frontend-student`** 为静态站点（HTML/CSS/JS）。**Docker 构建的后端镜像**会将两套页面装入容器 **`STATIC_DIR`（默认 `/srv/static`）**，并由 Go 进程挂载到 **`/admin/`**、**`/student/`**（与 API 同端口，免跨域）。Compose 仍可选启动独立 Nginx 容器映射 `:3001` / `:3000`。环境变量 **`CORS_ALLOWED_ORIGINS`**：默认（**未设置**时）在代码与 Compose 中均以 **`*`** 开头，对浏览器 **`http://`/`https://` Origin 回显允许**，用 **局域网/公网 IP + 分端口** 时可不显式写每个 IP。**公网生产**请覆盖该变量并**去掉 `*`**，只保留可信 Origin（实现见 `middleware/cors.go`）。页面支持 **`?api=`** 与 `localStorage` 覆盖 API 根地址。

## Quick Start (with Docker Compose)

```bash
cp .env.example .env
docker compose up -d --build
```

`docker-compose` 默认 **`ANALYSIS_ADAPTER=http`**：导入 seed 后优先调用库内激活的 `ai_provider_model`（含 DeepSeek/Kimi 等）。若仅想用 **mock-ai** 协议、不配库内密钥，可在 `.env` 里把激活模型关掉或设 `AI_ENDPOINT=http://mock-ai:8090/analyze` 并保证解析不会命中带 `app_secret` 的模型（见环境变量说明）。

Initialize schema (PostgreSQL baseline):

```bash
docker compose exec -T postgres psql -U "${POSTGRES_USER:-stepup_user}" -d "${POSTGRES_DB:-stepup}" < db/schema/postgresql_schema_v0.1_260403.sql
```

`db/seed/*.sql` 仍以 MySQL 语法为主；迁到 Postgres 后请用手写 `psql`/`COPY` 或待补充的 PG 种子，勿直接执行原 MySQL seed。

When `DB_DSN` is set, the process opens **one** shared `*sqlx.DB` pool (PostgreSQL via pgx); login sessions use **`sys_session`** with **`user_type`** `admin` or `student`; papers use **`student_exam_paper`** / **`student_paper_analysis`** / **`student_improvement_plan`**.

**AI 调用日志**：学生上传触发同步分析时写入 **`ai_call_log`**（适配器、`outcome`、耗时、错误、`endpoint`、`chat_model`、是否回退 mock、可选 **`sys_user_id`**、**`ref_table`/`ref_id`**、**`request_body`/`response_body`** 等；不落 API Key，**图片 inline base64 在 `request_body` 中脱敏**。表定义见 `db/schema/postgresql_schema_v0.1_260403.sql`；只读 `GET /api/v1/admin/ai-call-logs`；说明见 `docs/core/api_v0.1_260403.md` §3.12。

## Environment Variables

Copy `backend/.env.example` and export values in your shell.

**QA / 测试（仅 `go run`，不经 Compose）**：可使用 `backend/.env.qa` 为模板，改 `DB_DSN`、密码与 `CORS_ALLOWED_ORIGINS` 后加载再启动。Docker 编排请用仓库根目录的 **`.env.qa`** + `docker compose --env-file .env.qa`。

- `APP_ENV` - `dev` by default
- `HTTP_HOST` - `0.0.0.0` by default
- `HTTP_PORT` - `8080` by default
- `DB_DSN` - PostgreSQL URL for pgx (e.g. `postgres://user:pass@127.0.0.1:5432/stepup?sslmode=disable`); when unset, auth and papers stay in-memory
- `CORS_ALLOWED_ORIGINS` - 逗号分隔 Origin；可含 `*`（回显 Origin，见上文，勿滥用）
- `ANALYSIS_ADAPTER` - `mock` (default) or `http`
- `AI_ENDPOINT` - HTTP adapter fallback URL when `ANALYSIS_ADAPTER=http` (see resolution below)
- `AI_REQUEST_TIMEOUT_SECONDS` - timeout for HTTP adapter calls (default **180**; vision / slow networks may need more)

**HTTP 分析地址解析（`ANALYSIS_ADAPTER=http`）**：仅当使用 HTTP 适配器时生效。若已配置 `DB_DSN`，优先使用数据库 **`ai_provider_model` 表中当前激活的一条**（`status=1`、`is_deleted=0`，按 `id` 取最新）的 **`url`**；若没有激活模型或查不到行，则使用 **`AI_ENDPOINT`**；若仍没有可用 URL，则退回 **mock** 行为。写入 **`student_paper_analysis`** 的模型快照为 `name` + `url`（不含密钥）。当该行的 **`app_secret` 非空** 时，请求走 **OpenAI 兼容 `chat/completions`**（`Authorization: Bearer <app_secret>`，表字段 **`model`** 作为 JSON **`model`** 名，例如 `deepseek-chat`）；`app_secret` 为空时仍按 **mock-ai** 协议（JSON：`subject` / `stage` / `file_name`）请求 **`AI_ENDPOINT`** 或自建桥接服务。

## Current Scope

- HTTP server bootstrap
- Graceful shutdown
- Health endpoints:
  - `GET /healthz`
  - `GET /readyz` — if `DB_DSN` is set, checks DB ping (`503` + `DATABASE_UNAVAILABLE` / `DATABASE_UNREACHABLE` when not healthy)
- Admin auth minimal flow (in-memory without `DB_DSN`; `sys_admin_user` + `sys_session` when configured):
  - `POST /api/v1/admin/auth/login`
  - `POST /api/v1/admin/auth/logout`
  - `GET /api/v1/admin/auth/me`
- Student auth minimal flow (in-memory without `DB_DSN`; `sys_user`, `sys_verification_code`, `sys_session` when configured):
  - `POST /api/v1/student/auth/send-code`
  - `POST /api/v1/student/auth/verify-code`
  - `POST /api/v1/student/auth/set-password`
  - `POST /api/v1/student/auth/login`
  - `POST /api/v1/student/auth/logout`
  - `GET /api/v1/student/auth/me`
- Student paper flow (in-memory without `DB_DSN`; persisted to MySQL when `DB_DSN` is set):
  - `POST /api/v1/student/papers` (multipart: `subject`, `stage`, `file`)
  - `GET /api/v1/student/papers`
  - `GET /api/v1/student/papers/{paperId}/analysis`
  - `GET /api/v1/student/papers/{paperId}/plan`
  - All paper endpoints require `Authorization: Bearer <student_token>`
- `GET /api/v1/admin/students` — list students (requires admin Bearer token and `DB_DSN`)
- `POST /api/v1/admin/students` — create student (identifier/password/name/stage)
- `PATCH /api/v1/admin/students/{studentId}` — update student (name/stage/status/password)
- `GET|POST /api/v1/admin/subjects`, `PATCH /api/v1/admin/subjects/{subjectId}` — subject CRUD
- 教材目录（仅更新、无增删）：`GET /api/v1/admin/subjects/{subjectId}/textbooks`，`PATCH /api/v1/admin/textbooks/{textbookId}`，`GET /api/v1/admin/textbooks/{textbookId}/chapters`，`PATCH /api/v1/admin/chapters/{chapterId}`，`GET /api/v1/admin/chapters/{chapterId}/sections`，`PATCH /api/v1/admin/sections/{sectionId}`
- `GET|POST /api/v1/admin/stages`, `PATCH /api/v1/admin/stages/{stageId}` — stage CRUD
- `GET|POST /api/v1/admin/ai-models`, `PATCH /api/v1/admin/ai-models/{modelId}` — AI 模型（列表不返回 secret；激活一个模型会将其他模型置为非激活）
- `GET /api/v1/admin/prompts`, `PATCH /api/v1/admin/prompts/{promptId}` — Prompt 模板（预置行，仅更新）
- `GET /api/v1/admin/audit-logs` — 审计日志只读列表（`?limit=`，默认 100，最大 500）

## Audit log（`audit_log`）

在 **`DB_DSN` 已配置** 时，关键写操作会异步写入 `audit_log`（短超时，失败不影响主流程），包括：管理员与学生登录；学生上传试卷创建分析任务；管理员对学生 / 科目 / 阶段 / AI 模型 / Prompt 的创建与更新。涉及 **`app_secret` 的 PATCH** 记为 **`credential_change`**；学生/管理员密码变更记为 **`password_change`**。`snapshot` 刻意避免密码、`app_secret` 等敏感正文，多为标识字段或布尔标记。

## Notes

- Without `DB_DSN`, admin and student auth (and student papers) use in-memory stores; **audit 写入同样依赖数据库**，无 DB 时不落库。
- With `DB_DSN`, admin uses `sys_admin_user` + `sys_session` (`user_type=admin`); end users use `sys_user`, `sys_verification_code`, and `sys_session` (`user_type=student`); papers persist to **`student_exam_paper`** and related tables.
- Dev seed stores the bootstrap admin password as **bcrypt** (`admin123` by default; regenerate with `go run scripts/gen_admin_bcrypt.go` if you change it).
- Student paper analysis uses a pluggable `AnalysisAdapter` (default mock).
- `ANALYSIS_ADAPTER=http` uses the URL resolution order above; HTTP 调用失败或解析失败时仍可能退回 mock 输出。
