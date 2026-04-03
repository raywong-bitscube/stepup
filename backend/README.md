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

## Environment Variables

Copy `backend/.env.example` and export values in your shell.

- `APP_ENV` - `dev` by default
- `HTTP_HOST` - `0.0.0.0` by default
- `HTTP_PORT` - `8080` by default
- `DB_DSN` - MySQL DSN placeholder for next phase
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
- Admin auth minimal flow (in-memory session store):
  - `POST /api/v1/admin/auth/login`
  - `POST /api/v1/admin/auth/logout`
  - `GET /api/v1/admin/auth/me`
- Student auth minimal flow (in-memory):
  - `POST /api/v1/student/auth/send-code`
  - `POST /api/v1/student/auth/verify-code`
  - `POST /api/v1/student/auth/set-password`
  - `POST /api/v1/student/auth/login`
  - `POST /api/v1/student/auth/logout`
  - `GET /api/v1/student/auth/me`
- Student paper flow (in-memory):
  - `POST /api/v1/student/papers` (multipart: `subject`, `stage`, `file`)
  - `GET /api/v1/student/papers`
  - `GET /api/v1/student/papers/{paperId}/analysis`
  - `GET /api/v1/student/papers/{paperId}/plan`
  - All paper endpoints require `Authorization: Bearer <student_token>`
- API route skeleton for v0.1 endpoints (returns `501 NOT_IMPLEMENTED`)

## Notes

- Current admin auth/session is scaffold-only and uses in-memory sessions.
- Current student auth flow is scaffold-only and uses in-memory stores.
- Replace with DB implementations (`admin`, `admin_session`, `student`, `verification_code`) in next step.
- Current seed admin password uses plain text to match scaffold behavior. Replace with bcrypt before production.
- Student paper analysis currently uses a pluggable `AnalysisAdapter` (default mock).
- HTTP adapter scaffold is available via `ANALYSIS_ADAPTER=http`; it falls back to mock output on request/parse failure.
