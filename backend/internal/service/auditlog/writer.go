package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"strings"
	"time"
)

// Writer appends rows to audit_log. A nil Writer or nil db is a no-op.
type Writer struct {
	db *sql.DB
}

func New(db *sql.DB) *Writer {
	if db == nil {
		return nil
	}
	return &Writer{db: db}
}

// Event describes one audit row (no sensitive payloads).
type Event struct {
	UserID     *uint64
	UserType   string // "admin" | "student"
	Action     string
	EntityType string
	EntityID   *uint64
	Snapshot   json.RawMessage
	IP         string
	CreatedBy  uint64
}

// Write inserts asynchronously safe: short timeout, ignores errors.
func (w *Writer) Write(ctx context.Context, e Event) {
	if w == nil || w.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var userID any
	if e.UserID != nil {
		userID = *e.UserID
	}
	var entityID any
	if e.EntityID != nil {
		entityID = *e.EntityID
	}
	var snap any
	if len(e.Snapshot) > 0 {
		snap = string(e.Snapshot)
	}

	ipKey := ClientIP(e.IP)

	_, _ = w.db.ExecContext(ctx, `
INSERT INTO audit_log
  (user_id, user_type, action, entity_type, entity_id, snapshot, ip_address, created_at, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), ?)
`, userID, e.UserType, e.Action, e.EntityType, entityID, snap, nullIfEmpty(ipKey), e.CreatedBy)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ClientIP trims IPv6:port from RemoteAddr when present.
func ClientIP(remoteAddr string) string {
	s := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(s); err == nil {
		return host
	}
	return s
}
