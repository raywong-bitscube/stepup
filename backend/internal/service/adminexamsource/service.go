package adminexamsource

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Service struct {
	db        *sqlx.DB
	uploadDir string
}

func New(db *sqlx.DB, uploadDir string) *Service {
	uploadDir = strings.TrimSpace(uploadDir)
	if uploadDir == "" {
		uploadDir = "data/uploads"
	}
	return &Service{db: db, uploadDir: uploadDir}
}

type Paper struct {
	ID              uint64
	PaperCode       *string
	Title           string
	SourceRegion    *string
	SourceSchool    *string
	ExamYear        *int
	Term            *string
	GradeLabel      *string
	K12GradeID      *uint64
	K12SubjectID    uint64
	PaperType       string
	TotalScore      *string
	DurationMinutes *int
	PageCount       int
	QuestionCount   int
	Remarks         *string
	Status          int
	UpdatedAt       time.Time
	CreatedAt       time.Time
}

type Page struct {
	ID        uint64
	PaperID   uint64
	PageNo    int
	FileID    uint64
	PublicURL *string
	Status    int
}

type Question struct {
	ID            uint64
	PaperID       uint64
	QuestionNo    string
	QuestionOrder int
	SectionNo     *string
	QuestionType  string
	Score         *string
	StemText      *string
	AnswerText    *string
	Explanation   *string
	PageFrom      *int
	PageTo        *int
	Status        int
	UpdatedAt     time.Time
}

type CreatePaperInput struct {
	PaperCode       string
	Title           string
	SourceRegion    string
	SourceSchool    string
	ExamYear        *int
	Term            string
	GradeLabel      string
	K12GradeID      *uint64
	K12SubjectID    uint64
	PaperType       string
	TotalScore      *string
	DurationMinutes *int
	Remarks         string
	Status          int
}

type PatchPaperInput struct {
	PaperCode       *string
	Title           *string
	SourceRegion    *string
	SourceSchool    *string
	ExamYear        *int
	Term            *string
	GradeLabel      *string
	K12GradeID      *uint64
	K12SubjectID    *uint64
	PaperType       *string
	TotalScore      *string
	DurationMinutes *int
	Remarks         *string
	Status          *int
}

type CreateQuestionInput struct {
	QuestionNo    string
	QuestionOrder int
	SectionNo     string
	QuestionType  string
	Score         *string
	StemText      string
	AnswerText    string
	Explanation   string
	PageFrom      *int
	PageTo        *int
	Status        int
}

type PatchQuestionInput struct {
	QuestionNo    *string
	QuestionOrder *int
	SectionNo     *string
	QuestionType  *string
	Score         *string
	StemText      *string
	AnswerText    *string
	Explanation   *string
	PageFrom      *int
	PageTo        *int
	Status        *int
}

type UploadImage struct {
	Filename string
	Bytes    []byte
}

type CreatePaperWithUploadInput struct {
	CreatePaperInput
	QuestionNos []string
}

func (s *Service) ListPapers(ctx context.Context) ([]Paper, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
SELECT id, paper_code, title, source_region, source_school, exam_year, term, grade_label,
       k12_grade_id, k12_subject_id, paper_type, total_score::text, duration_minutes,
       page_count, question_count, remarks, status, updated_at, created_at
FROM exam_source_paper
WHERE is_deleted = 0
ORDER BY id DESC
LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Paper, 0, 64)
	for rows.Next() {
		var (
			it              Paper
			paperCode       sql.NullString
			sourceRegion    sql.NullString
			sourceSchool    sql.NullString
			examYear        sql.NullInt64
			term            sql.NullString
			gradeLabel      sql.NullString
			k12GradeID      sql.NullInt64
			totalScore      sql.NullString
			durationMinutes sql.NullInt64
			remarks         sql.NullString
		)
		if err := rows.Scan(
			&it.ID, &paperCode, &it.Title, &sourceRegion, &sourceSchool, &examYear, &term, &gradeLabel,
			&k12GradeID, &it.K12SubjectID, &it.PaperType, &totalScore, &durationMinutes,
			&it.PageCount, &it.QuestionCount, &remarks, &it.Status, &it.UpdatedAt, &it.CreatedAt,
		); err != nil {
			return nil, err
		}
		it.PaperCode = nullableStringPtr(paperCode)
		it.SourceRegion = nullableStringPtr(sourceRegion)
		it.SourceSchool = nullableStringPtr(sourceSchool)
		if examYear.Valid {
			v := int(examYear.Int64)
			it.ExamYear = &v
		}
		it.Term = nullableStringPtr(term)
		it.GradeLabel = nullableStringPtr(gradeLabel)
		if k12GradeID.Valid && k12GradeID.Int64 > 0 {
			v := uint64(k12GradeID.Int64)
			it.K12GradeID = &v
		}
		it.TotalScore = nullableStringPtr(totalScore)
		if durationMinutes.Valid {
			v := int(durationMinutes.Int64)
			it.DurationMinutes = &v
		}
		it.Remarks = nullableStringPtr(remarks)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) GetPaper(ctx context.Context, id uint64) (*Paper, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if id == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var (
		it              Paper
		paperCode       sql.NullString
		sourceRegion    sql.NullString
		sourceSchool    sql.NullString
		examYear        sql.NullInt64
		term            sql.NullString
		gradeLabel      sql.NullString
		k12GradeID      sql.NullInt64
		totalScore      sql.NullString
		durationMinutes sql.NullInt64
		remarks         sql.NullString
	)
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT id, paper_code, title, source_region, source_school, exam_year, term, grade_label,
       k12_grade_id, k12_subject_id, paper_type, total_score::text, duration_minutes,
       page_count, question_count, remarks, status, updated_at, created_at
FROM exam_source_paper
WHERE id = ? AND is_deleted = 0`), id).Scan(
		&it.ID, &paperCode, &it.Title, &sourceRegion, &sourceSchool, &examYear, &term, &gradeLabel,
		&k12GradeID, &it.K12SubjectID, &it.PaperType, &totalScore, &durationMinutes,
		&it.PageCount, &it.QuestionCount, &remarks, &it.Status, &it.UpdatedAt, &it.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	it.PaperCode = nullableStringPtr(paperCode)
	it.SourceRegion = nullableStringPtr(sourceRegion)
	it.SourceSchool = nullableStringPtr(sourceSchool)
	if examYear.Valid {
		v := int(examYear.Int64)
		it.ExamYear = &v
	}
	it.Term = nullableStringPtr(term)
	it.GradeLabel = nullableStringPtr(gradeLabel)
	if k12GradeID.Valid && k12GradeID.Int64 > 0 {
		v := uint64(k12GradeID.Int64)
		it.K12GradeID = &v
	}
	it.TotalScore = nullableStringPtr(totalScore)
	if durationMinutes.Valid {
		v := int(durationMinutes.Int64)
		it.DurationMinutes = &v
	}
	it.Remarks = nullableStringPtr(remarks)
	return &it, nil
}

func (s *Service) CreatePaper(ctx context.Context, adminID uint64, in CreatePaperInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	if adminID == 0 || in.K12SubjectID == 0 || strings.TrimSpace(in.Title) == "" {
		return 0, ErrInvalidInput
	}
	now := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	status := in.Status
	if status == 0 {
		status = 1
	}
	paperType := strings.TrimSpace(in.PaperType)
	if paperType == "" {
		paperType = "mock_exam"
	}
	var id uint64
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_paper
  (paper_code, title, source_region, source_school, exam_year, term, grade_label, k12_grade_id, k12_subject_id,
   paper_type, total_score, duration_minutes, page_count, question_count, remarks, status,
   created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, 0)
RETURNING id`),
		emptyToNil(in.PaperCode), strings.TrimSpace(in.Title), emptyToNil(in.SourceRegion), emptyToNil(in.SourceSchool),
		nullableIntArg(in.ExamYear), emptyToNil(in.Term), emptyToNil(in.GradeLabel), nullableUintArg(in.K12GradeID), in.K12SubjectID,
		paperType, nullableNumericText(in.TotalScore), nullableIntArg(in.DurationMinutes), emptyToNil(in.Remarks), status,
		now, adminID, now, adminID,
	).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return 0, ErrConflict
		}
		return 0, err
	}
	return id, nil
}

func (s *Service) PatchPaper(ctx context.Context, id uint64, adminID uint64, in PatchPaperInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if id == 0 || adminID == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var sets []string
	var args []any
	if in.PaperCode != nil {
		sets = append(sets, "paper_code = ?")
		args = append(args, emptyToNil(*in.PaperCode))
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "title = ?")
		args = append(args, t)
	}
	if in.SourceRegion != nil {
		sets = append(sets, "source_region = ?")
		args = append(args, emptyToNil(*in.SourceRegion))
	}
	if in.SourceSchool != nil {
		sets = append(sets, "source_school = ?")
		args = append(args, emptyToNil(*in.SourceSchool))
	}
	if in.ExamYear != nil {
		sets = append(sets, "exam_year = ?")
		args = append(args, nullableIntArg(in.ExamYear))
	}
	if in.Term != nil {
		sets = append(sets, "term = ?")
		args = append(args, emptyToNil(*in.Term))
	}
	if in.GradeLabel != nil {
		sets = append(sets, "grade_label = ?")
		args = append(args, emptyToNil(*in.GradeLabel))
	}
	if in.K12GradeID != nil {
		sets = append(sets, "k12_grade_id = ?")
		args = append(args, nullableUintArg(in.K12GradeID))
	}
	if in.K12SubjectID != nil {
		if *in.K12SubjectID == 0 {
			return ErrInvalidInput
		}
		sets = append(sets, "k12_subject_id = ?")
		args = append(args, *in.K12SubjectID)
	}
	if in.PaperType != nil {
		pt := strings.TrimSpace(*in.PaperType)
		if pt == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "paper_type = ?")
		args = append(args, pt)
	}
	if in.TotalScore != nil {
		sets = append(sets, "total_score = ?")
		args = append(args, nullableNumericText(in.TotalScore))
	}
	if in.DurationMinutes != nil {
		sets = append(sets, "duration_minutes = ?")
		args = append(args, nullableIntArg(in.DurationMinutes))
	}
	if in.Remarks != nil {
		sets = append(sets, "remarks = ?")
		args = append(args, emptyToNil(*in.Remarks))
	}
	if in.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}
	sets = append(sets, "updated_at = ?", "updated_by = ?")
	args = append(args, time.Now(), adminID, id)
	q := `UPDATE exam_source_paper SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, dbutil.Rebind(q), args...)
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

func (s *Service) ListPages(ctx context.Context, paperID uint64) ([]Page, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if paperID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT p.id, p.paper_id, p.page_no, p.file_id, f.public_url, p.status
FROM exam_source_paper_page p
JOIN exam_source_file f ON f.id = p.file_id AND f.is_deleted = 0
WHERE p.paper_id = ? AND p.is_deleted = 0
ORDER BY p.page_no ASC, p.id ASC`), paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Page, 0, 16)
	for rows.Next() {
		var it Page
		var pu sql.NullString
		if err := rows.Scan(&it.ID, &it.PaperID, &it.PageNo, &it.FileID, &pu, &it.Status); err != nil {
			return nil, err
		}
		it.PublicURL = nullableStringPtr(pu)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) ListQuestions(ctx context.Context, paperID uint64) ([]Question, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if paperID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT id, paper_id, question_no, question_order, section_no, question_type, score::text,
       stem_text, answer_text, explanation_text, page_from, page_to, status, updated_at
FROM exam_source_question
WHERE paper_id = ? AND is_deleted = 0
ORDER BY question_order ASC, id ASC`), paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Question, 0, 32)
	for rows.Next() {
		var (
			it          Question
			sectionNo   sql.NullString
			scoreText   sql.NullString
			stemText    sql.NullString
			answerText  sql.NullString
			explainText sql.NullString
			pageFrom    sql.NullInt64
			pageTo      sql.NullInt64
		)
		if err := rows.Scan(
			&it.ID, &it.PaperID, &it.QuestionNo, &it.QuestionOrder, &sectionNo, &it.QuestionType, &scoreText,
			&stemText, &answerText, &explainText, &pageFrom, &pageTo, &it.Status, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		it.SectionNo = nullableStringPtr(sectionNo)
		it.Score = nullableStringPtr(scoreText)
		it.StemText = nullableStringPtr(stemText)
		it.AnswerText = nullableStringPtr(answerText)
		it.Explanation = nullableStringPtr(explainText)
		if pageFrom.Valid {
			v := int(pageFrom.Int64)
			it.PageFrom = &v
		}
		if pageTo.Valid {
			v := int(pageTo.Int64)
			it.PageTo = &v
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) CreateQuestion(ctx context.Context, paperID uint64, adminID uint64, in CreateQuestionInput) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	if paperID == 0 || adminID == 0 || strings.TrimSpace(in.QuestionNo) == "" {
		return 0, ErrInvalidInput
	}
	qType := strings.TrimSpace(in.QuestionType)
	if qType == "" {
		qType = "unknown"
	}
	status := in.Status
	if status == 0 {
		status = 1
	}
	now := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var id uint64
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question
  (paper_id, question_no, question_order, section_no, question_type, score, stem_text, answer_text, explanation_text,
   page_from, page_to, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
RETURNING id`),
		paperID, strings.TrimSpace(in.QuestionNo), in.QuestionOrder, emptyToNil(in.SectionNo), qType,
		nullableNumericText(in.Score), emptyToNil(in.StemText), emptyToNil(in.AnswerText), emptyToNil(in.Explanation),
		nullableIntArg(in.PageFrom), nullableIntArg(in.PageTo), status, now, adminID, now, adminID,
	).Scan(&id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return 0, ErrConflict
		}
		return 0, err
	}
	_, _ = s.db.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_paper
SET question_count = (SELECT COUNT(*) FROM exam_source_question q WHERE q.paper_id = ? AND q.is_deleted = 0),
    updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`), paperID, now, adminID, paperID)
	return id, nil
}

func (s *Service) PatchQuestion(ctx context.Context, questionID uint64, adminID uint64, in PatchQuestionInput) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if questionID == 0 || adminID == 0 {
		return ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var sets []string
	var args []any
	if in.QuestionNo != nil {
		no := strings.TrimSpace(*in.QuestionNo)
		if no == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "question_no = ?")
		args = append(args, no)
	}
	if in.QuestionOrder != nil {
		sets = append(sets, "question_order = ?")
		args = append(args, *in.QuestionOrder)
	}
	if in.SectionNo != nil {
		sets = append(sets, "section_no = ?")
		args = append(args, emptyToNil(*in.SectionNo))
	}
	if in.QuestionType != nil {
		qt := strings.TrimSpace(*in.QuestionType)
		if qt == "" {
			return ErrInvalidInput
		}
		sets = append(sets, "question_type = ?")
		args = append(args, qt)
	}
	if in.Score != nil {
		sets = append(sets, "score = ?")
		args = append(args, nullableNumericText(in.Score))
	}
	if in.StemText != nil {
		sets = append(sets, "stem_text = ?")
		args = append(args, emptyToNil(*in.StemText))
	}
	if in.AnswerText != nil {
		sets = append(sets, "answer_text = ?")
		args = append(args, emptyToNil(*in.AnswerText))
	}
	if in.Explanation != nil {
		sets = append(sets, "explanation_text = ?")
		args = append(args, emptyToNil(*in.Explanation))
	}
	if in.PageFrom != nil {
		sets = append(sets, "page_from = ?")
		args = append(args, nullableIntArg(in.PageFrom))
	}
	if in.PageTo != nil {
		sets = append(sets, "page_to = ?")
		args = append(args, nullableIntArg(in.PageTo))
	}
	if in.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *in.Status)
	}
	if len(sets) == 0 {
		return ErrInvalidInput
	}
	sets = append(sets, "updated_at = ?", "updated_by = ?")
	args = append(args, time.Now(), adminID, questionID)
	q := `UPDATE exam_source_question SET ` + strings.Join(sets, ", ") + ` WHERE id = ? AND is_deleted = 0`
	res, err := s.db.ExecContext(ctx, dbutil.Rebind(q), args...)
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

func (s *Service) CreatePaperWithImages(ctx context.Context, adminID uint64, in CreatePaperWithUploadInput, images []UploadImage) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	if adminID == 0 || in.K12SubjectID == 0 || strings.TrimSpace(in.Title) == "" || len(images) == 0 {
		return 0, ErrInvalidInput
	}
	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		return 0, err
	}
	now := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	written := make([]string, 0, len(images))
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			for _, p := range written {
				_ = os.Remove(p)
			}
		}
	}()

	var paperID uint64
	status := in.Status
	if status == 0 {
		status = 1
	}
	pType := strings.TrimSpace(in.PaperType)
	if pType == "" {
		pType = "mock_exam"
	}
	err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_paper
  (paper_code, title, source_region, source_school, exam_year, term, grade_label, k12_grade_id, k12_subject_id,
   paper_type, total_score, duration_minutes, page_count, question_count, remarks, status,
   created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, 0)
RETURNING id`),
		emptyToNil(in.PaperCode), strings.TrimSpace(in.Title), emptyToNil(in.SourceRegion), emptyToNil(in.SourceSchool),
		nullableIntArg(in.ExamYear), emptyToNil(in.Term), emptyToNil(in.GradeLabel), nullableUintArg(in.K12GradeID),
		in.K12SubjectID, pType, nullableNumericText(in.TotalScore), nullableIntArg(in.DurationMinutes), len(images), emptyToNil(in.Remarks),
		status, now, adminID, now, adminID,
	).Scan(&paperID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return 0, ErrConflict
		}
		return 0, err
	}

	monthPath := now.Format("200601")
	baseRel := filepath.ToSlash(filepath.Join("exam-source", monthPath))
	for i, img := range images {
		if len(img.Bytes) == 0 {
			err = ErrInvalidInput
			return 0, err
		}
		ext := fileExt(img.Filename)
		stored := fmt.Sprintf("%d_p%d_%s%s", paperID, i+1, randHex(8), ext)
		rel := filepath.ToSlash(filepath.Join(baseRel, stored))
		abs := filepath.Join(s.uploadDir, rel)
		if mkErr := os.MkdirAll(filepath.Dir(abs), 0755); mkErr != nil {
			err = mkErr
			return 0, err
		}
		if wrErr := os.WriteFile(abs, img.Bytes, 0644); wrErr != nil {
			err = wrErr
			return 0, err
		}
		written = append(written, abs)

		publicURL := "/uploads/" + rel
		var fileID uint64
		err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_file
  (storage_provider, bucket_name, object_key, public_url, original_filename, content_type, file_ext, size_bytes,
   status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES ('local', NULL, ?, ?, ?, NULL, ?, ?, 1, ?, ?, ?, ?, 0)
RETURNING id`),
			rel, publicURL, strings.TrimSpace(img.Filename), strings.TrimPrefix(ext, "."), int64(len(img.Bytes)),
			now, adminID, now, adminID,
		).Scan(&fileID)
		if err != nil {
			return 0, err
		}
		_, err = tx.ExecContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_paper_page
  (paper_id, page_no, file_id, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, 1, ?, ?, ?, ?, 0)`),
			paperID, i+1, fileID, now, adminID, now, adminID,
		)
		if err != nil {
			return 0, err
		}
	}

	qNos := normalizeQuestionNos(in.QuestionNos)
	if len(qNos) > 0 {
		for i, qn := range qNos {
			_, err = tx.ExecContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question
  (paper_id, question_no, question_order, question_type, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, 'unknown', 1, ?, ?, ?, ?, 0)`),
				paperID, qn, i+1, now, adminID, now, adminID,
			)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
					return 0, ErrConflict
				}
				return 0, err
			}
		}
		_, err = tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_paper
SET question_count = ?, updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`), len(qNos), now, adminID, paperID)
		if err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return paperID, nil
}

func emptyToNil(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func nullableIntArg(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableUintArg(v *uint64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableNumericText(v *string) any {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(t, 64); err != nil {
		return nil
	}
	return t
}

func nullableStringPtr(v sql.NullString) *string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	s := v.String
	return &s
}

func fileExt(name string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(strings.TrimSpace(name))))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		return ext
	default:
		return ".jpg"
	}
}

func randHex(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func normalizeQuestionNos(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, x := range in {
		v := strings.TrimSpace(x)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
