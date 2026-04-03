# StepUp Backend Scaffold (v0.1)

## Run

From repository root:

```bash
go run ./backend/cmd/server
```

Server default address: `0.0.0.0:8080`

## Quick Start (with Docker Compose)

```bash
cp .env.example .env
docker compose up -d --build
```

`docker-compose` 默认 **`ANALYSIS_ADAPTER=http`**：导入 seed 后优先调用库内激活的 `ai_model`（含 DeepSeek）。若仅想用 **mock-ai** 协议、不配库内密钥，可在 `.env` 里把激活模型关掉或设 `AI_ENDPOINT=http://mock-ai:8090/analyze` 并保证解析不会命中带 `app_secret` 的模型（见环境变量说明）。

Initialize schema and seed:

```bash
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < docs/mysql_schema_v0.1_260403.sql
docker compose exec -T mysql mysql -u"${MYSQL_USER}" -p"${MYSQL_PASSWORD}" "${MYSQL_DATABASE}" < scripts/dev_seed.sql
```

When `DB_DSN` is set, the process opens **one** shared `*sql.DB` connection pool; admin sessions use `admin_session`, student sessions use `student_session`, and papers use `exam_paper` / `paper_analysis` / `improvement_plan`. If you initialized the DB before `student_session` existed, re-apply the schema file or add that table manually.

## Environment Variables

Copy `backend/.env.example` and export values in your shell.

- `APP_ENV` - `dev` by default
- `HTTP_HOST` - `0.0.0.0` by default
- `HTTP_PORT` - `8080` by default
- `DB_DSN` - MySQL DSN (e.g. `user:pass@tcp(127.0.0.1:3306)/stepup?parseTime=true&loc=Local`); when unset, auth and papers stay in-memory
- `CORS_ALLOWED_ORIGINS` - `http://localhost:3000,http://localhost:3001`
- `ANALYSIS_ADAPTER` - `mock` (default) or `http`
- `AI_ENDPOINT` - HTTP adapter fallback URL when `ANALYSIS_ADAPTER=http` (see resolution below)
- `AI_REQUEST_TIMEOUT_SECONDS` - timeout for HTTP adapter calls

**HTTP 分析地址解析（`ANALYSIS_ADAPTER=http`）**：仅当使用 HTTP 适配器时生效。若已配置 `DB_DSN`，优先使用 MySQL **`ai_model` 表中当前激活的一条**（`status=1`、`is_deleted=0`，按 `id` 取最新）的 **`url`**；若没有激活模型或查不到行，则使用 **`AI_ENDPOINT`**；若仍没有可用 URL，则退回 **mock** 行为。写入 `paper_analysis` 的模型快照为 `name` + `url`（不含密钥）。当该行的 **`app_secret` 非空** 时，请求走 **OpenAI 兼容 `chat/completions`**（`Authorization: Bearer <app_secret>`，`app_key` 作为 **model** 名，例如 `deepseek-chat`）；`app_secret` 为空时仍按 **mock-ai** 协议（JSON：`subject` / `stage` / `file_name`）请求 **`AI_ENDPOINT`** 或自建桥接服务。

## Current Scope

- HTTP server bootstrap
- Graceful shutdown
- Health endpoints:
  - `GET /healthz`
  - `GET /readyz` — if `DB_DSN` is set, checks DB ping (`503` + `DATABASE_UNAVAILABLE` / `DATABASE_UNREACHABLE` when not healthy)
- Admin auth minimal flow (in-memory without `DB_DSN`; `admin` + `admin_session` when configured):
  - `POST /api/v1/admin/auth/login`
  - `POST /api/v1/admin/auth/logout`
  - `GET /api/v1/admin/auth/me`
- Student auth minimal flow (in-memory without `DB_DSN`; `student`, `verification_code`, `student_session` when configured):
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
- `GET|POST /api/v1/admin/stages`, `PATCH /api/v1/admin/stages/{stageId}` — stage CRUD
- `GET|POST /api/v1/admin/ai-models`, `PATCH /api/v1/admin/ai-models/{modelId}` — AI 模型（列表不返回 secret；激活一个模型会将其他模型置为非激活）
- `GET|POST /api/v1/admin/prompts`, `PATCH /api/v1/admin/prompts/{promptId}` — Prompt 模板
- `GET /api/v1/admin/audit-logs` — 审计日志只读列表（`?limit=`，默认 100，最大 500）

## Audit log（`audit_log`）

在 **`DB_DSN` 已配置** 时，关键写操作会异步写入 `audit_log`（短超时，失败不影响主流程），包括：管理员与学生登录；学生上传试卷创建分析任务；管理员对学生 / 科目 / 阶段 / AI 模型 / Prompt 的创建与更新。涉及 **`app_secret` 的 PATCH** 记为 **`credential_change`**；学生/管理员密码变更记为 **`password_change`**。`snapshot` 刻意避免密码、`app_secret` 等敏感正文，多为标识字段或布尔标记。

## Notes

- Without `DB_DSN`, admin and student auth (and student papers) use in-memory stores; **audit 写入同样依赖数据库**，无 DB 时不落库。
- With `DB_DSN`, admin uses `admin` + `admin_session`; student uses `student`, `verification_code`, and `student_session`; papers persist to `exam_paper` and related tables.
- Dev seed stores the bootstrap admin password as **bcrypt** (`admin123` by default; regenerate with `go run scripts/gen_admin_bcrypt.go` if you change it).
- Student paper analysis uses a pluggable `AnalysisAdapter` (default mock).
- `ANALYSIS_ADAPTER=http` uses the URL resolution order above; HTTP 调用失败或解析失败时仍可能退回 mock 输出。
