package studentslidedeck

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNoDatabase = errors.New("database not configured")
	ErrNotFound   = errors.New("not found")
)

type ActiveDeck struct {
	ID            uint64          `json:"id"`
	SectionID     uint64          `json:"section_id"`
	Title         string          `json:"title"`
	SchemaVersion int             `json:"schema_version"`
	Content       json.RawMessage `json:"content"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// StripQuestionAnswers removes "answer" keys from question elements (student-safe JSON).
func StripQuestionAnswers(content []byte) ([]byte, error) {
	var root interface{}
	if err := json.Unmarshal(content, &root); err != nil {
		return nil, err
	}
	stripWalk(root)
	return json.Marshal(root)
}

func stripWalk(v interface{}) {
	switch x := v.(type) {
	case map[string]interface{}:
		if t, ok := x["type"].(string); ok && t == "question" {
			delete(x, "answer")
		}
		for _, val := range x {
			stripWalk(val)
		}
	case []interface{}:
		for _, el := range x {
			stripWalk(el)
		}
	}
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

// GetActive returns the active slide deck for a section; content omits question answers.
func (s *Service) GetActive(ctx context.Context, sectionID uint64) (*ActiveDeck, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if sectionID == 0 {
		return nil, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	ok, err := s.sectionExists(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var d ActiveDeck
	var raw []byte
	err = s.db.QueryRowContext(ctx, `
SELECT id, section_id, title, schema_version, content, updated_at
FROM slide_deck
WHERE section_id = ? AND deck_status = 'active' AND is_deleted = 0
ORDER BY id DESC
LIMIT 1`, sectionID).Scan(&d.ID, &d.SectionID, &d.Title, &d.SchemaVersion, &raw, &d.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sanitized, err := StripQuestionAnswers(raw)
	if err != nil {
		return nil, err
	}
	d.Content = sanitized
	return &d, nil
}
