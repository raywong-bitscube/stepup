package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// OpenPostgres opens a pooled *sqlx.DB for PostgreSQL (pgx stdlib driver).
// DB_DSN example: postgres://user:pass@localhost:5432/stepup?sslmode=disable
func OpenPostgres(dsn string) (*sqlx.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("database: empty DB_DSN")
	}
	// go-sql-driver/mysql DSNs use "@tcp(host:port)/dbname". pgx expects a URL or libpq keyword string.
	if strings.Contains(dsn, "@tcp(") {
		return nil, fmt.Errorf(`database: DB_DSN looks like MySQL (contains "@tcp("); use a PostgreSQL URL, e.g. postgres://USER:PASS@HOST:5432/DBNAME?sslmode=disable`)
	}
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
