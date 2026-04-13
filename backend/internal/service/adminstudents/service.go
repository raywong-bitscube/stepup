package adminstudents

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	ErrNotFound     = errors.New("not found")
)

type Student struct {
	ID        uint64    `json:"id"`
	Phone     *string   `json:"phone"`
	Email     *string   `json:"email"`
	Name      string    `json:"name"`
	StageName string    `json:"stage"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]Student, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
SELECT s.id, s.phone, s.email, s.name, stg.name, s.status, s.created_at
FROM sys_user s
JOIN k12_grade stg ON stg.id = s.k12_grade_id
WHERE s.is_deleted = 0
ORDER BY s.id DESC
LIMIT 500`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Student, 0, 32)
	for rows.Next() {
		var (
			id        uint64
			phone     sql.NullString
			email     sql.NullString
			name      string
			stageName string
			status    int
			createdAt time.Time
		)
		if err := rows.Scan(&id, &phone, &email, &name, &stageName, &status, &createdAt); err != nil {
			return nil, err
		}
		st := Student{
			ID:        id,
			Name:      name,
			StageName: stageName,
			Status:    status,
			CreatedAt: createdAt,
		}
		if phone.Valid && phone.String != "" {
			p := phone.String
			st.Phone = &p
		}
		if email.Valid && email.String != "" {
			e := email.String
			st.Email = &e
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

type CreateInput struct {
	Identifier string
	Password   string
	Name       string
	Stage      string
}

type UpdateInput struct {
	Name     *string
	Stage    *string
	Status   *int
	Password *string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	identifier := strings.TrimSpace(strings.ToLower(in.Identifier))
	name := strings.TrimSpace(in.Name)
	stage := strings.TrimSpace(in.Stage)
	password := strings.TrimSpace(in.Password)
	if identifier == "" || password == "" || name == "" || stage == "" {
		return 0, ErrInvalidInput
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var stageID uint64
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT id FROM k12_grade
WHERE name = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`), stage).Scan(&stageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidInput
		}
		return 0, err
	}

	phone := sql.NullString{}
	email := sql.NullString{}
	if strings.Contains(identifier, "@") {
		email = sql.NullString{String: identifier, Valid: true}
	} else {
		phone = sql.NullString{String: identifier, Valid: true}
	}
	now := time.Now()
	var nid uint64
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO sys_user
  (phone, email, password, name, k12_grade_id, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, 1, ?, 0, ?, 0, 0)
RETURNING id
`), phone, email, string(hashed), name, stageID, now, now).Scan(&nid)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return 0, ErrConflict
		}
		return 0, err
	}
	return nid, nil
}

func (s *Service) Patch(ctx context.Context, studentID uint64, in UpdateInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if studentID == 0 {
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
	if in.Status != nil {
		if *in.Status != 0 && *in.Status != 1 {
			return ErrInvalidInput
		}
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if in.Stage != nil {
		stage := strings.TrimSpace(*in.Stage)
		if stage == "" {
			return ErrInvalidInput
		}
		var stageID uint64
		if err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT id FROM k12_grade
WHERE name = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`), stage).Scan(&stageID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidInput
			}
			return err
		}
		sets = append(sets, "k12_grade_id = ?")
		args = append(args, stageID)
	}
	if in.Password != nil {
		password := strings.TrimSpace(*in.Password)
		if password == "" {
			return ErrInvalidInput
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		sets = append(sets, "password = ?")
		args = append(args, string(hashed))
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}

	sets = append(sets, "updated_at = ?", "updated_by = 0")
	args = append(args, time.Now())
	args = append(args, studentID)

	query := `UPDATE sys_user SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, dbutil.Rebind(query), args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
