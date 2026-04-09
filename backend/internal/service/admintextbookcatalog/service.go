package admintextbookcatalog

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
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) subjectExists(ctx context.Context, subjectID uint64) error {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM subject WHERE id = ? AND is_deleted = 0 LIMIT 1`, subjectID).Scan(&one)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}

// TextbookRow is a row for admin textbook list (category read-only for display).
type TextbookRow struct {
	ID        uint64
	Name      string
	Version   string
	Subject   string
	Category  string
	Remarks   *string
	Status    int
	UpdatedAt time.Time
}

func (s *Service) ListTextbooksBySubject(ctx context.Context, subjectID uint64) ([]TextbookRow, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if subjectID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.subjectExists(ctx, subjectID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, version, subject, category, remarks, status, updated_at
FROM textbook
WHERE subject_id = ? AND is_deleted = 0
ORDER BY id ASC`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TextbookRow
	for rows.Next() {
		var (
			r         TextbookRow
			remarks   sql.NullString
			updatedAt time.Time
		)
		if err := rows.Scan(&r.ID, &r.Name, &r.Version, &r.Subject, &r.Category, &remarks, &r.Status, &updatedAt); err != nil {
			return nil, err
		}
		r.UpdatedAt = updatedAt
		if remarks.Valid && remarks.String != "" {
			rs := remarks.String
			r.Remarks = &rs
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type TextbookPatch struct {
	Name    *string
	Version *string
	Subject *string
	Remarks *string
	Status  *int
}

func (s *Service) PatchTextbook(ctx context.Context, id uint64, in TextbookPatch) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
	if in.Version != nil {
		v := strings.TrimSpace(*in.Version)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "version = ?")
		args = append(args, v)
	}
	if in.Subject != nil {
		v := strings.TrimSpace(*in.Subject)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "subject = ?")
		args = append(args, v)
	}
	if in.Remarks != nil {
		v := strings.TrimSpace(*in.Remarks)
		if v == "" {
			sets = append(sets, "remarks = NULL")
		} else {
			sets = append(sets, "remarks = ?")
			args = append(args, v)
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
	now := time.Now()
	args = append(args, now, id)

	q := `UPDATE textbook SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
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

type ChapterRow struct {
	ID         uint64
	TextbookID uint64
	Number     uint32
	Title      string
	FullTitle  *string
	Status     int
	UpdatedAt  time.Time
}

func (s *Service) ListChapters(ctx context.Context, textbookID uint64) ([]ChapterRow, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if textbookID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var one uint64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM textbook WHERE id = ? AND is_deleted = 0`, textbookID).Scan(&one)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, textbook_id, number, title, full_title, status, updated_at
FROM chapter
WHERE textbook_id = ? AND is_deleted = 0
ORDER BY number ASC, id ASC`, textbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChapterRow
	for rows.Next() {
		var (
			r         ChapterRow
			ft        sql.NullString
			updatedAt time.Time
			num       int
		)
		if err := rows.Scan(&r.ID, &r.TextbookID, &num, &r.Title, &ft, &r.Status, &updatedAt); err != nil {
			return nil, err
		}
		r.Number = uint32(num)
		r.UpdatedAt = updatedAt
		if ft.Valid && ft.String != "" {
			s := ft.String
			r.FullTitle = &s
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type ChapterPatch struct {
	Number    *uint32
	Title     *string
	FullTitle *string
	Status    *int
}

func (s *Service) PatchChapter(ctx context.Context, id uint64, in ChapterPatch) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var sets []string
	var args []any
	if in.Number != nil {
		sets = append(sets, "number = ?")
		args = append(args, *in.Number)
	}
	if in.Title != nil {
		v := strings.TrimSpace(*in.Title)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "title = ?")
		args = append(args, v)
	}
	if in.FullTitle != nil {
		v := strings.TrimSpace(*in.FullTitle)
		if v == "" {
			sets = append(sets, "full_title = NULL")
		} else {
			sets = append(sets, "full_title = ?")
			args = append(args, v)
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
	now := time.Now()
	args = append(args, now, id)

	q := `UPDATE chapter SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type SectionRow struct {
	ID             uint64
	ChapterID      uint64
	Number         uint32
	Title          string
	FullTitle      *string
	Status         int
	SlideDeckCount int
	UpdatedAt      time.Time
}

func (s *Service) ListSections(ctx context.Context, chapterID uint64) ([]SectionRow, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if chapterID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var one uint64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM chapter WHERE id = ? AND is_deleted = 0`, chapterID).Scan(&one)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT s.id, s.chapter_id, s.number, s.title, s.full_title, s.status, s.updated_at,
       (SELECT COUNT(*) FROM slide_deck sd WHERE sd.section_id = s.id AND sd.is_deleted = 0) AS slide_deck_count
FROM section s
WHERE s.chapter_id = ? AND s.is_deleted = 0
ORDER BY s.number ASC, s.id ASC`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SectionRow
	for rows.Next() {
		var (
			r         SectionRow
			ft        sql.NullString
			updatedAt time.Time
			num       int
			sdc       int
		)
		if err := rows.Scan(&r.ID, &r.ChapterID, &num, &r.Title, &ft, &r.Status, &updatedAt, &sdc); err != nil {
			return nil, err
		}
		r.SlideDeckCount = sdc
		r.Number = uint32(num)
		r.UpdatedAt = updatedAt
		if ft.Valid && ft.String != "" {
			s := ft.String
			r.FullTitle = &s
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type SectionPatch struct {
	Number    *uint32
	Title     *string
	FullTitle *string
	Status    *int
}

func (s *Service) PatchSection(ctx context.Context, id uint64, in SectionPatch) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var sets []string
	var args []any
	if in.Number != nil {
		sets = append(sets, "number = ?")
		args = append(args, *in.Number)
	}
	if in.Title != nil {
		v := strings.TrimSpace(*in.Title)
		if v == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "title = ?")
		args = append(args, v)
	}
	if in.FullTitle != nil {
		v := strings.TrimSpace(*in.FullTitle)
		if v == "" {
			sets = append(sets, "full_title = NULL")
		} else {
			sets = append(sets, "full_title = ?")
			args = append(args, v)
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
	now := time.Now()
	args = append(args, now, id)

	q := `UPDATE section SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
