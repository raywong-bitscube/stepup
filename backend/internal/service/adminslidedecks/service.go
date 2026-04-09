package adminslidedecks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

const (
	StatusDraft    = "draft"
	StatusActive   = "active"
	StatusArchived = "archived"
)

type Summary struct {
	ID            uint64    `json:"id"`
	SectionID     uint64    `json:"section_id"`
	Title         string    `json:"title"`
	DeckStatus    string    `json:"deck_status"`
	SchemaVersion int       `json:"schema_version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Full struct {
	Summary
	Content json.RawMessage `json:"content"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func normalizeStatus(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateSlideJSON checks schemaVersion == 1 and top-level shape.
func ValidateSlideJSON(raw json.RawMessage) error {
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return ErrInvalidInput
	}
	sv, ok := root["schemaVersion"]
	if !ok {
		return ErrInvalidInput
	}
	var v int
	switch x := sv.(type) {
	case float64:
		v = int(x)
	case int:
		v = x
	default:
		return ErrInvalidInput
	}
	if v != 1 {
		return ErrInvalidInput
	}
	slides, ok := root["slides"].([]interface{})
	if !ok || slides == nil {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) sectionExists(ctx context.Context, sectionID uint64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
SELECT 1 FROM section WHERE id = ? AND is_deleted = 0 LIMIT 1`, sectionID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ListSummaries(ctx context.Context, sectionID uint64) ([]Summary, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
SELECT id, section_id, title, deck_status, schema_version, updated_at
FROM slide_deck
WHERE section_id = ? AND is_deleted = 0
ORDER BY id DESC
LIMIT 200`, sectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Summary, 0, 8)
	for rows.Next() {
		var x Summary
		if err := rows.Scan(&x.ID, &x.SectionID, &x.Title, &x.DeckStatus, &x.SchemaVersion, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, deckID uint64) (*Full, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var f Full
	var content []byte
	err := s.db.QueryRowContext(ctx, `
SELECT id, section_id, title, deck_status, schema_version, updated_at, content
FROM slide_deck
WHERE id = ? AND is_deleted = 0`, deckID).Scan(
		&f.ID, &f.SectionID, &f.Title, &f.DeckStatus, &f.SchemaVersion, &f.UpdatedAt, &content,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	f.Content = content
	return &f, nil
}

type CreateInput struct {
	Title         string
	Content       json.RawMessage
	DeckStatus    string
	SchemaVersion int
}

func (s *Service) Create(ctx context.Context, sectionID uint64, adminID uint64, in CreateInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	if sectionID == 0 || adminID == 0 {
		return 0, ErrInvalidInput
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return 0, ErrInvalidInput
	}
	if len(in.Content) == 0 {
		return 0, ErrInvalidInput
	}
	if err := ValidateSlideJSON(in.Content); err != nil {
		return 0, err
	}
	st := normalizeStatus(in.DeckStatus)
	if st == "" {
		st = StatusDraft
	}
	if st != StatusDraft && st != StatusActive && st != StatusArchived {
		return 0, ErrInvalidInput
	}
	sv := in.SchemaVersion
	if sv == 0 {
		sv = 1
	}
	if sv != 1 {
		return 0, ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	ok, err := s.sectionExists(ctx, sectionID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if st == StatusActive {
		_, err = tx.ExecContext(ctx, `
UPDATE slide_deck SET deck_status = ?, updated_at = NOW(), updated_by = ?
WHERE section_id = ? AND deck_status = ? AND is_deleted = 0`,
			StatusArchived, adminID, sectionID, StatusActive)
		if err != nil {
			return 0, err
		}
	}

	res, err := tx.ExecContext(ctx, `
INSERT INTO slide_deck
  (section_id, title, deck_status, schema_version, content, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, CAST(? AS JSON), NOW(), ?, NOW(), ?, 0)`,
		sectionID, title, st, sv, string(in.Content), adminID, adminID)
	if err != nil {
		return 0, err
	}
	id64, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id64), nil
}

type PatchInput struct {
	Title      *string
	Content    *json.RawMessage
	DeckStatus *string
}

func (s *Service) Patch(ctx context.Context, deckID uint64, adminID uint64, in PatchInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if deckID == 0 || adminID == 0 {
		return ErrInvalidInput
	}
	if in.Title == nil && in.Content == nil && in.DeckStatus == nil {
		return ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	var sectionID uint64
	var curStatus string
	err := s.db.QueryRowContext(ctx, `
SELECT section_id, deck_status FROM slide_deck WHERE id = ? AND is_deleted = 0`, deckID).Scan(&sectionID, &curStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	newStatus := curStatus
	if in.DeckStatus != nil {
		st := normalizeStatus(*in.DeckStatus)
		if st != StatusDraft && st != StatusActive && st != StatusArchived {
			return ErrInvalidInput
		}
		newStatus = st
	}

	var contentBytes []byte
	if in.Content != nil {
		if err := ValidateSlideJSON(*in.Content); err != nil {
			return err
		}
		contentBytes = *in.Content
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if newStatus == StatusActive && curStatus != StatusActive {
		_, err = tx.ExecContext(ctx, `
UPDATE slide_deck SET deck_status = ?, updated_at = NOW(), updated_by = ?
WHERE section_id = ? AND id <> ? AND deck_status = ? AND is_deleted = 0`,
			StatusArchived, adminID, sectionID, deckID, StatusActive)
		if err != nil {
			return err
		}
	}

	var sets []string
	var args []interface{}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "title = ?")
		args = append(args, t)
	}
	if in.Content != nil {
		sets = append(sets, "content = CAST(? AS JSON)")
		args = append(args, string(contentBytes))
	}
	if in.DeckStatus != nil {
		sets = append(sets, "deck_status = ?")
		args = append(args, newStatus)
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}
	args = append(args, adminID, deckID)
	q := "UPDATE slide_deck SET " + strings.Join(sets, ", ") + ", updated_at = NOW(), updated_by = ? WHERE id = ? AND is_deleted = 0"
	_, err = tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	return tx.Commit()
}
