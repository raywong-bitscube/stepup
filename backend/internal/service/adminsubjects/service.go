package adminsubjects

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	ErrNotFound     = errors.New("not found")
)

type Subject struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]Subject, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
SELECT id, name, description, status, created_at
FROM subject
WHERE is_deleted = 0
ORDER BY id DESC
LIMIT 500`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Subject, 0, 32)
	for rows.Next() {
		var (
			id          uint64
			name        string
			description sql.NullString
			status      int
			createdAt   time.Time
		)
		if err := rows.Scan(&id, &name, &description, &status, &createdAt); err != nil {
			return nil, err
		}
		sub := Subject{ID: id, Name: name, Status: status, CreatedAt: createdAt}
		if description.Valid && description.String != "" {
			d := description.String
			sub.Description = &d
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

type CreateInput struct {
	Name        string
	Description string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return 0, ErrInvalidInput
	}
	desc := sql.NullString{}
	d := strings.TrimSpace(in.Description)
	if d != "" {
		desc = sql.NullString{String: d, Valid: true}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO subject
  (name, description, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, 1, ?, 0, ?, 0, 0)
`, name, desc, now, now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return 0, ErrConflict
		}
		return 0, err
	}
	nid, _ := res.LastInsertId()
	return uint64(nid), nil
}

type UpdateInput struct {
	Name        *string
	Description *string
	Status      *int
}

func (s *Service) Patch(ctx context.Context, id uint64, in UpdateInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var sets []string
	var args []any
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "name = ?")
		args = append(args, name)
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if d == "" {
			sets = append(sets, "description = NULL")
		} else {
			sets = append(sets, "description = ?")
			args = append(args, d)
		}
	}
	if in.Status != nil {
		if *in.Status != 0 && *in.Status != 1 {
			return ErrInvalidInput
		}
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}
	sets = append(sets, "updated_at = ?", "updated_by = 0")
	args = append(args, time.Now(), id)

	q := `UPDATE subject SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return ErrConflict
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
