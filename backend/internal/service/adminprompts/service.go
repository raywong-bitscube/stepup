package adminprompts

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

type Prompt struct {
	ID          uint64    `json:"id"`
	Key         string    `json:"key"`
	Description *string   `json:"description"`
	Content     string    `json:"content"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Service struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]Prompt, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
SELECT id, "key", description, content, status, created_at
FROM prompt_template
WHERE is_deleted = 0
ORDER BY "key" ASC
LIMIT 500`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Prompt, 0, 32)
	for rows.Next() {
		var (
			id          uint64
			key         string
			description sql.NullString
			content     string
			status      int
			createdAt   time.Time
		)
		if err := rows.Scan(&id, &key, &description, &content, &status, &createdAt); err != nil {
			return nil, err
		}
		p := Prompt{ID: id, Key: key, Content: content, Status: status, CreatedAt: createdAt}
		if description.Valid && description.String != "" {
			d := description.String
			p.Description = &d
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type UpdateInput struct {
	Description *string
	Content     *string
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
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if d == "" {
			sets = append(sets, "description = NULL")
		} else {
			sets = append(sets, "description = ?")
			args = append(args, d)
		}
	}
	if in.Content != nil {
		c := strings.TrimSpace(*in.Content)
		if c == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "content = ?")
		args = append(args, c)
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
	now := time.Now()
	args = append(args, now, id)

	q := `UPDATE prompt_template SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, dbutil.Rebind(q), args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
