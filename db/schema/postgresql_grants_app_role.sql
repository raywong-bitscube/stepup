-- Grant privileges for the application DB user (default name: stepup_user, see docker-compose POSTGRES_USER).
--
-- When: after `postgresql_schema_v0.1_260403.sql` and migrations, if DDL was run as `postgres` (or another
-- superuser) while the backend connects as `stepup_user`. Otherwise you get:
--   permission denied for table ... (SQLSTATE 42501)
--
-- Run as a role that can grant on these objects (typically the table owner / superuser), e.g.:
--   psql "postgres://postgres:...@host:5432/stepup?sslmode=disable" -v ON_ERROR_STOP=1 -f db/schema/postgresql_grants_app_role.sql
--
-- Replace `stepup_user` in this file if your app role name differs (or use sed).

GRANT USAGE ON SCHEMA public TO stepup_user;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO stepup_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO stepup_user;

-- Objects created later by the same session user (e.g. postgres) in public will get these grants automatically:
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO stepup_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO stepup_user;
