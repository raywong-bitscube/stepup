package adminaimodels

import (
	"context"
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

// PublicModel is safe to return from APIs (no app_secret).
type PublicModel struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Model     string    `json:"model"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]PublicModel, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	const q = `
SELECT id, name, url, model, status, created_at
FROM ai_provider_model
WHERE is_deleted = 0
ORDER BY id DESC
LIMIT 500`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PublicModel, 0, 32)
	for rows.Next() {
		var m PublicModel
		if err := rows.Scan(&m.ID, &m.Name, &m.URL, &m.Model, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type CreateInput struct {
	Name      string
	URL       string
	Model     string
	AppSecret string
	Status    *int
}

func (s *Service) Create(ctx context.Context, in CreateInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	name := strings.TrimSpace(in.Name)
	url := strings.TrimSpace(in.URL)
	modelID := strings.TrimSpace(in.Model)
	secret := strings.TrimSpace(in.AppSecret)
	if name == "" || url == "" || modelID == "" || secret == "" {
		return 0, ErrInvalidInput
	}
	status := 0
	if in.Status != nil {
		if *in.Status != 0 && *in.Status != 1 {
			return 0, ErrInvalidInput
		}
		status = *in.Status
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	now := time.Now()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if status == 1 {
		if _, err := tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE ai_provider_model
SET status = 0, updated_at = ?, updated_by = 0
WHERE is_deleted = 0
`), now); err != nil {
			return 0, err
		}
	}
	var nid uint64
	err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO ai_provider_model
  (name, url, model, app_secret, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, 0, 0)
RETURNING id
`), name, url, modelID, secret, status, now, now).Scan(&nid)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return nid, nil
}

type UpdateInput struct {
	Name      *string
	URL       *string
	Model     *string
	AppSecret *string
	Status    *int
}

func (s *Service) Patch(ctx context.Context, id uint64, in UpdateInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var sets []string
	var args []any
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "name = ?")
		args = append(args, v)
	}
	if in.URL != nil {
		v := strings.TrimSpace(*in.URL)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "url = ?")
		args = append(args, v)
	}
	if in.Model != nil {
		v := strings.TrimSpace(*in.Model)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "model = ?")
		args = append(args, v)
	}
	if in.AppSecret != nil {
		sets = append(sets, "app_secret = ?")
		args = append(args, strings.TrimSpace(*in.AppSecret))
	}
	activating := false
	if in.Status != nil {
		if *in.Status != 0 && *in.Status != 1 {
			return ErrInvalidInput
		}
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
		if *in.Status == 1 {
			activating = true
		}
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}
	sets = append(sets, "updated_at = ?", "updated_by = 0")
	now := time.Now()
	args = append(args, now, id)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if activating {
		if _, err := tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE ai_provider_model
SET status = 0, updated_at = ?, updated_by = 0
WHERE is_deleted = 0 AND id != ?
`), now, id); err != nil {
			return err
		}
	}

	q := `UPDATE ai_provider_model SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := tx.ExecContext(ctx, dbutil.Rebind(q), args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
