package adminstudents

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNoDatabase = errors.New("database not configured")

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
	db *sql.DB
}

func New(db *sql.DB) *Service {
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
FROM student s
JOIN stage stg ON stg.id = s.stage_id
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
