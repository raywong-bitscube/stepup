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

To enable HTTP analysis adapter with local mock-ai:

```bash
export ANALYSIS_ADAPTER=http
export AI_ENDPOINT=http://mock-ai:8090/analyze
docker compose up -d --build
```

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
- `AI_ENDPOINT` - HTTP adapter target endpoint (used when `ANALYSIS_ADAPTER=http`)
- `AI_REQUEST_TIMEOUT_SECONDS` - timeout for HTTP adapter calls

## Current Scope

- HTTP server bootstrap
- Graceful shutdown
- Health endpoints:
  - `GET /healthz`
  - `GET /readyz`
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
- API route skeleton for v0.1 endpoints (returns `501 NOT_IMPLEMENTED`)

## Notes

- Without `DB_DSN`, admin and student auth (and student papers) use in-memory stores.
- With `DB_DSN`, admin uses `admin` + `admin_session`; student uses `student`, `verification_code`, and `student_session`; papers persist to `exam_paper` and related tables.
- Dev seed stores the bootstrap admin password as **bcrypt** (`admin123` by default; regenerate with `go run scripts/gen_admin_bcrypt.go` if you change it).
- Student paper analysis currently uses a pluggable `AnalysisAdapter` (default mock).
- HTTP adapter scaffold is available via `ANALYSIS_ADAPTER=http`; it falls back to mock output on request/parse failure.
