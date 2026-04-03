package adminaudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
)

type Entry struct {
	ID         uint64          `json:"id"`
	UserID     *uint64         `json:"user_id"`
	UserType   string          `json:"user_type"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   *uint64         `json:"entity_id"`
	Snapshot   json.RawMessage `json:"snapshot"`
	IPAddress  *string         `json:"ip_address"`
	CreatedAt  time.Time       `json:"created_at"`
	CreatedBy  uint64          `json:"created_by"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, limit int) ([]Entry, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if limit <= 0 || limit > 500 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
SELECT id, user_id, user_type, action, entity_type, entity_id, snapshot, ip_address, created_at, created_by
FROM audit_log
ORDER BY id DESC
LIMIT ?`
	rows, err := s.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Entry, 0, limit)
	for rows.Next() {
		var (
			id         uint64
			userID     sql.NullInt64
			userType   string
			action     string
			entityType string
			entityID   sql.NullInt64
			snapshot   sql.NullString
			ip         sql.NullString
			createdAt  time.Time
			createdBy  uint64
		)
		if err := rows.Scan(&id, &userID, &userType, &action, &entityType, &entityID, &snapshot, &ip, &createdAt, &createdBy); err != nil {
			return nil, err
		}
		e := Entry{
			ID:         id,
			UserType:   userType,
			Action:     action,
			EntityType: entityType,
			CreatedAt:  createdAt,
			CreatedBy:  createdBy,
		}
		if userID.Valid {
			u := uint64(userID.Int64)
			e.UserID = &u
		}
		if entityID.Valid {
			x := uint64(entityID.Int64)
			e.EntityID = &x
		}
		if snapshot.Valid && snapshot.String != "" {
			e.Snapshot = json.RawMessage(snapshot.String)
		} else {
			e.Snapshot = json.RawMessage("null")
		}
		if ip.Valid && ip.String != "" {
			s := ip.String
			e.IPAddress = &s
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
