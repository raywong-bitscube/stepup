package dbutil

import "github.com/jmoiron/sqlx"

// Rebind converts MySQL-style "?" placeholders to PostgreSQL "$1", "$2", ...
func Rebind(q string) string {
	return sqlx.Rebind(sqlx.DOLLAR, q)
}
