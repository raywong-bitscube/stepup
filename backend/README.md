# StepUp Backend Scaffold (v0.1)

## Run

From repository root:

```bash
go run ./backend/cmd/server
```

Server default address: `0.0.0.0:8080`

## Environment Variables

Copy `backend/.env.example` and export values in your shell.

- `APP_ENV` - `dev` by default
- `HTTP_HOST` - `0.0.0.0` by default
- `HTTP_PORT` - `8080` by default
- `DB_DSN` - MySQL DSN placeholder for next phase

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
- API route skeleton for v0.1 endpoints (returns `501 NOT_IMPLEMENTED`)

## Notes

- Current admin auth/session is scaffold-only and uses in-memory sessions.
- Replace with `admin` and `admin_session` table implementation in next step.
