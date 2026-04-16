package adminexamsource

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

const examSourceRecognizeMaxTokens = 32768

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
	GroupID       *uint64
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

// QuestionGroup is a paper section (大题) with system-classified type and the full rubric line from the scan.
type QuestionGroup struct {
	ID              uint64
	PaperID         uint64
	GroupOrder      int
	SystemKind      string
	TitleLabel      *string
	DescriptionText *string
	PageNo          *int
	Status          int
	UpdatedAt       time.Time
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

type recognizedBBox struct {
	X float64
	Y float64
	W float64
	H float64
}

type recognizedGroup struct {
	GroupOrder      int
	SystemKind      string
	TitleLabel      string
	DescriptionText string
	PageNo          int
}

type recognizedQuestion struct {
	GroupOrder    int
	QuestionNo    string
	QuestionType  string
	PageNo        int
	QuestionOrder int
	BBox          *recognizedBBox
	StemText      string
	AnswerText    string
	Explanation   string
}

type recognitionOutput struct {
	Groups    []recognizedGroup
	Questions []recognizedQuestion
}

type storedPage struct {
	PageNo int
	FileID uint64
	Rel    string
	Abs    string
}

// RecognitionPreviewQuestion bundles a question row with its primary stem crop (if any).
type RecognitionPreviewQuestion struct {
	Question
	StemQuestionFileID *uint64
	StemFileID         *uint64
	StemCropURL        *string
	StemPageNo         *int
	StemBBoxNorm       map[string]float64
}

// PatchStemBBoxInput is normalized bbox (0–1) on a given page.
type PatchStemBBoxInput struct {
	PageNo int
	X      float64
	Y      float64
	W      float64
	H      float64
}

// PatchStemBBoxResult is returned after re-crop and DB update.
type PatchStemBBoxResult struct {
	CropPublicURL string
	PageNo        int
	BBoxNorm      map[string]float64
}

type UploadAnalyzeQuestion struct {
	GroupOrder    int
	QuestionNo    string
	QuestionOrder int
	QuestionType  string
	PageNo        int
	BBoxNorm      map[string]float64
	StemText      string
	AnswerText    string
	Explanation   string
}

type UploadAnalyzeGroup struct {
	GroupOrder      int
	SystemKind      string
	TitleLabel      string
	DescriptionText string
	PageNo          int
}

type UploadAnalyzeResult struct {
	Title           string
	SourceRegion    string
	SourceSchool    string
	ExamYear        *int
	Term            string
	GradeLabel      string
	K12GradeID      *uint64
	K12SubjectID    *uint64
	SuggestedSubject string
	PaperType       string
	TotalScore      *string
	DurationMinutes *int
	Groups          []UploadAnalyzeGroup
	QuestionNos     []string
	Questions       []UploadAnalyzeQuestion
}

type recognizedPaperMeta struct {
	Title           string
	SubjectName     string
	GradeLabel      string
	ExamYear        int
	Term            string
	SourceRegion    string
	SourceSchool    string
	PaperType       string
	TotalScore      string
	DurationMinutes int
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
SELECT id, paper_id, group_id, question_no, question_order, section_no, question_type, score::text,
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
			groupID     sql.NullInt64
			sectionNo   sql.NullString
			scoreText   sql.NullString
			stemText    sql.NullString
			answerText  sql.NullString
			explainText sql.NullString
			pageFrom    sql.NullInt64
			pageTo      sql.NullInt64
		)
		if err := rows.Scan(
			&it.ID, &it.PaperID, &groupID, &it.QuestionNo, &it.QuestionOrder, &sectionNo, &it.QuestionType, &scoreText,
			&stemText, &answerText, &explainText, &pageFrom, &pageTo, &it.Status, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		it.SectionNo = nullableStringPtr(sectionNo)
		it.Score = nullableStringPtr(scoreText)
		it.StemText = nullableStringPtr(stemText)
		it.AnswerText = nullableStringPtr(answerText)
		it.Explanation = nullableStringPtr(explainText)
		if groupID.Valid && groupID.Int64 > 0 {
			v := uint64(groupID.Int64)
			it.GroupID = &v
		}
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

func (s *Service) ListQuestionGroups(ctx context.Context, paperID uint64) ([]QuestionGroup, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if paperID == 0 {
		return nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT id, paper_id, group_order, system_kind, title_label, description_text, page_no, status, updated_at
FROM exam_source_question_group
WHERE paper_id = ? AND is_deleted = 0
ORDER BY group_order ASC, id ASC`), paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]QuestionGroup, 0, 16)
	for rows.Next() {
		var (
			it          QuestionGroup
			titleLabel  sql.NullString
			description sql.NullString
			pageNo      sql.NullInt64
		)
		if err := rows.Scan(
			&it.ID, &it.PaperID, &it.GroupOrder, &it.SystemKind, &titleLabel, &description, &pageNo, &it.Status, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		it.TitleLabel = nullableStringPtr(titleLabel)
		it.DescriptionText = nullableStringPtr(description)
		if pageNo.Valid {
			v := int(pageNo.Int64)
			it.PageNo = &v
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Service) GetRecognitionPreview(ctx context.Context, paperID uint64) ([]Page, []RecognitionPreviewQuestion, error) {
	if s == nil || s.db == nil {
		return nil, nil, ErrNoDatabase
	}
	if paperID == 0 {
		return nil, nil, ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if _, err := s.GetPaper(ctx, paperID); err != nil {
		return nil, nil, err
	}
	pages, err := s.ListPages(ctx, paperID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT q.id, q.paper_id, q.group_id, q.question_no, q.question_order, q.section_no, q.question_type, q.score::text,
       q.stem_text, q.answer_text, q.explanation_text, q.page_from, q.page_to, q.status, q.updated_at,
       qf.id, qf.file_id, f.public_url, qf.page_no, qf.bbox_norm
FROM exam_source_question q
LEFT JOIN LATERAL (
  SELECT qf2.id, qf2.file_id, qf2.page_no, qf2.bbox_norm
  FROM exam_source_question_file qf2
  WHERE qf2.question_id = q.id AND qf2.role = 'stem' AND qf2.is_deleted = 0
  ORDER BY qf2.sort_no ASC, qf2.id ASC
  LIMIT 1
) qf ON true
LEFT JOIN exam_source_file f ON f.id = qf.file_id AND f.is_deleted = 0
WHERE q.paper_id = ? AND q.is_deleted = 0
ORDER BY q.question_order ASC, q.id ASC`), paperID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := make([]RecognitionPreviewQuestion, 0, 32)
	for rows.Next() {
		var (
			it          RecognitionPreviewQuestion
			groupID     sql.NullInt64
			sectionNo   sql.NullString
			scoreText   sql.NullString
			stemText    sql.NullString
			answerText  sql.NullString
			explainText sql.NullString
			pageFrom    sql.NullInt64
			pageTo      sql.NullInt64
			stemQFID    sql.NullInt64
			stemFID     sql.NullInt64
			stemURL     sql.NullString
			stemPage    sql.NullInt64
			bboxRaw     []byte
		)
		if err := rows.Scan(
			&it.ID, &it.PaperID, &groupID, &it.QuestionNo, &it.QuestionOrder, &sectionNo, &it.QuestionType, &scoreText,
			&stemText, &answerText, &explainText, &pageFrom, &pageTo, &it.Status, &it.UpdatedAt,
			&stemQFID, &stemFID, &stemURL, &stemPage, &bboxRaw,
		); err != nil {
			return nil, nil, err
		}
		it.SectionNo = nullableStringPtr(sectionNo)
		it.Score = nullableStringPtr(scoreText)
		it.StemText = nullableStringPtr(stemText)
		it.AnswerText = nullableStringPtr(answerText)
		it.Explanation = nullableStringPtr(explainText)
		if groupID.Valid && groupID.Int64 > 0 {
			v := uint64(groupID.Int64)
			it.GroupID = &v
		}
		if pageFrom.Valid {
			v := int(pageFrom.Int64)
			it.PageFrom = &v
		}
		if pageTo.Valid {
			v := int(pageTo.Int64)
			it.PageTo = &v
		}
		if stemQFID.Valid {
			v := uint64(stemQFID.Int64)
			it.StemQuestionFileID = &v
		}
		if stemFID.Valid {
			v := uint64(stemFID.Int64)
			it.StemFileID = &v
		}
		if stemURL.Valid {
			u := stemURL.String
			it.StemCropURL = &u
		}
		if stemPage.Valid {
			v := int(stemPage.Int64)
			it.StemPageNo = &v
		}
		if len(bboxRaw) > 0 {
			var m map[string]float64
			if err := json.Unmarshal(bboxRaw, &m); err == nil && len(m) > 0 {
				it.StemBBoxNorm = m
			}
		}
		out = append(out, it)
	}
	return pages, out, rows.Err()
}

func finite01(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= 0 && v <= 1
}

func validStemBBoxInput(in PatchStemBBoxInput) bool {
	if in.PageNo <= 0 {
		return false
	}
	return finite01(in.X) && finite01(in.Y) && finite01(in.W) && finite01(in.H) && in.W > 0 && in.H > 0
}

func (s *Service) loadStoredPage(ctx context.Context, paperID uint64, pageNo int) (storedPage, error) {
	var rel string
	var fileID uint64
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT f.object_key, p.file_id
FROM exam_source_paper_page p
JOIN exam_source_file f ON f.id = p.file_id AND f.is_deleted = 0
WHERE p.paper_id = ? AND p.page_no = ? AND p.is_deleted = 0`), paperID, pageNo).Scan(&rel, &fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storedPage{}, ErrNotFound
		}
		return storedPage{}, err
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return storedPage{}, ErrInvalidInput
	}
	abs := filepath.Join(s.uploadDir, filepath.FromSlash(rel))
	return storedPage{
		PageNo: pageNo,
		FileID: fileID,
		Rel:    rel,
		Abs:    abs,
	}, nil
}

func (s *Service) PatchQuestionStemBBox(ctx context.Context, adminID uint64, questionID uint64, in PatchStemBBoxInput) (*PatchStemBBoxResult, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if adminID == 0 || questionID == 0 || !validStemBBoxInput(in) {
		return nil, ErrInvalidInput
	}
	box := recognizedBBox{X: in.X, Y: in.Y, W: in.W, H: in.H}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var paperID uint64
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT paper_id FROM exam_source_question WHERE id = ? AND is_deleted = 0`), questionID).Scan(&paperID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	page, err := s.loadStoredPage(ctx, paperID, in.PageNo)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var cropRel, cropAbs string
	cropCommitted := false
	defer func() {
		if !cropCommitted && cropAbs != "" {
			_ = os.Remove(cropAbs)
		}
	}()

	cropRel, cropAbs, err = s.cropQuestionImage(page, paperID, questionID, box, now)
	if err != nil {
		return nil, err
	}
	pub := "/uploads/" + cropRel
	sz := fileSizeOrZero(cropAbs)
	bboxNorm := map[string]float64{"x": box.X, "y": box.Y, "w": box.W, "h": box.H}
	bboxJSON, err := json.Marshal(bboxNorm)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var oldQfID, oldFileID uint64
	var oldObjectKey sql.NullString
	qErr := tx.QueryRowContext(ctx, dbutil.Rebind(`
SELECT qf.id, qf.file_id, f.object_key
FROM exam_source_question_file qf
JOIN exam_source_file f ON f.id = qf.file_id AND f.is_deleted = 0
WHERE qf.question_id = ? AND qf.role = 'stem' AND qf.is_deleted = 0
ORDER BY qf.sort_no ASC, qf.id ASC
LIMIT 1`), questionID).Scan(&oldQfID, &oldFileID, &oldObjectKey)

	if qErr != nil && !errors.Is(qErr, sql.ErrNoRows) {
		err = qErr
		return nil, err
	}

	if errors.Is(qErr, sql.ErrNoRows) {
		var cropFileID uint64
		err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_file
  (storage_provider, bucket_name, object_key, public_url, original_filename, content_type, file_ext, size_bytes,
   status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES ('local', NULL, ?, ?, ?, 'image/png', 'png', ?, 1, ?, ?, ?, ?, 0)
RETURNING id`),
			cropRel, pub, filepath.Base(cropRel), sz,
			now, adminID, now, adminID,
		).Scan(&cropFileID)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question_file
  (question_id, file_id, role, sort_no, page_no, bbox_norm, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, 'stem', 1, ?, ?::jsonb, 1, ?, ?, ?, ?, 0)`),
			questionID, cropFileID, in.PageNo, string(bboxJSON), now, adminID, now, adminID,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_file
SET object_key = ?, public_url = ?, original_filename = ?, content_type = 'image/png', file_ext = 'png', size_bytes = ?,
    updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`),
			cropRel, pub, filepath.Base(cropRel), sz,
			now, adminID, oldFileID,
		)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_question_file
SET page_no = ?, bbox_norm = ?::jsonb, updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`),
			in.PageNo, string(bboxJSON), now, adminID, oldQfID,
		)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_question
SET page_from = ?, page_to = ?, updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`),
		in.PageNo, in.PageNo, now, adminID, questionID,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	cropCommitted = true

	if oldObjectKey.Valid && strings.TrimSpace(oldObjectKey.String) != "" {
		oldAbs := filepath.Join(s.uploadDir, filepath.FromSlash(filepath.ToSlash(strings.TrimSpace(oldObjectKey.String))))
		if oldAbs != cropAbs {
			_ = os.Remove(oldAbs)
		}
	}

	return &PatchStemBBoxResult{
		CropPublicURL: pub,
		PageNo:        in.PageNo,
		BBoxNorm:      bboxNorm,
	}, nil
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

func (s *Service) AnalyzeUpload(ctx context.Context, titleHint string, images []UploadImage) (*UploadAnalyzeResult, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	if len(images) == 0 {
		return nil, ErrInvalidInput
	}
	titleHint = strings.TrimSpace(titleHint)
	meta, metaErr := s.recognizePaperMeta(ctx, titleHint, images)
	if metaErr != nil {
		meta = nil
	}
	recOut, recErr := s.recognizeQuestions(ctx, titleHint, images)
	var recognized []recognizedQuestion
	var recGroups []recognizedGroup
	if recErr != nil {
		recognized = nil
		recGroups = nil
	} else if recOut != nil {
		recognized = recOut.Questions
		recGroups = recOut.Groups
	}
	out := &UploadAnalyzeResult{
		PaperType: "mock_exam",
	}
	if titleHint != "" {
		out.Title = titleHint
	}
	if meta != nil {
		if t := strings.TrimSpace(meta.Title); t != "" {
			out.Title = t
		}
		out.SourceRegion = strings.TrimSpace(meta.SourceRegion)
		out.SourceSchool = strings.TrimSpace(meta.SourceSchool)
		out.Term = strings.TrimSpace(meta.Term)
		out.GradeLabel = strings.TrimSpace(meta.GradeLabel)
		if meta.ExamYear >= 2000 && meta.ExamYear <= 2100 {
			v := meta.ExamYear
			out.ExamYear = &v
		}
		if t := strings.TrimSpace(meta.PaperType); t != "" {
			out.PaperType = t
		}
		if ts := strings.TrimSpace(meta.TotalScore); ts != "" {
			out.TotalScore = &ts
		}
		if meta.DurationMinutes > 0 {
			v := meta.DurationMinutes
			out.DurationMinutes = &v
		}
		out.SuggestedSubject = strings.TrimSpace(meta.SubjectName)
	}
	if strings.TrimSpace(out.Title) == "" {
		out.Title = guessTitleFromImage(images)
	}
	if strings.TrimSpace(out.Title) == "" {
		out.Title = "未命名试卷"
	}
	if strings.TrimSpace(out.SuggestedSubject) != "" {
		if sid, err := s.matchSubjectIDByName(ctx, out.SuggestedSubject); err == nil {
			out.K12SubjectID = sid
		}
	}
	if strings.TrimSpace(out.GradeLabel) != "" {
		if gid, err := s.matchGradeIDByName(ctx, out.GradeLabel); err == nil {
			out.K12GradeID = gid
		}
	}
	out.Groups = make([]UploadAnalyzeGroup, 0, len(recGroups))
	for _, g := range recGroups {
		out.Groups = append(out.Groups, UploadAnalyzeGroup{
			GroupOrder:      g.GroupOrder,
			SystemKind:      g.SystemKind,
			TitleLabel:      g.TitleLabel,
			DescriptionText: g.DescriptionText,
			PageNo:          g.PageNo,
		})
	}
	out.QuestionNos = make([]string, 0, len(recognized))
	out.Questions = make([]UploadAnalyzeQuestion, 0, len(recognized))
	for _, rq := range recognized {
		no := strings.TrimSpace(rq.QuestionNo)
		if no == "" {
			continue
		}
		out.QuestionNos = append(out.QuestionNos, no)
		item := UploadAnalyzeQuestion{
			GroupOrder:    rq.GroupOrder,
			QuestionNo:    no,
			QuestionOrder: rq.QuestionOrder,
			QuestionType:  rq.QuestionType,
			PageNo:        rq.PageNo,
			StemText:      rq.StemText,
			AnswerText:    rq.AnswerText,
			Explanation:   rq.Explanation,
		}
		if rq.BBox != nil {
			item.BBoxNorm = map[string]float64{
				"x": rq.BBox.X,
				"y": rq.BBox.Y,
				"w": rq.BBox.W,
				"h": rq.BBox.H,
			}
		}
		out.Questions = append(out.Questions, item)
	}
	out.QuestionNos = normalizeQuestionNos(out.QuestionNos)
	return out, nil
}

func guessTitleFromImage(images []UploadImage) string {
	if len(images) == 0 {
		return ""
	}
	name := strings.TrimSpace(images[0].Filename)
	if name == "" {
		return ""
	}
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	return base
}

func normalizeMatchText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "")
	return s
}

func (s *Service) matchSubjectIDByName(ctx context.Context, name string) (*uint64, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	nameNorm := normalizeMatchText(name)
	if nameNorm == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name
FROM k12_subject
WHERE status = 1 AND is_deleted = 0
ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fuzzy *uint64
	for rows.Next() {
		var (
			id uint64
			nm string
		)
		if err := rows.Scan(&id, &nm); err != nil {
			return nil, err
		}
		norm := normalizeMatchText(nm)
		if norm == "" {
			continue
		}
		if norm == nameNorm {
			v := id
			return &v, nil
		}
		if fuzzy == nil && (strings.Contains(norm, nameNorm) || strings.Contains(nameNorm, norm)) {
			v := id
			fuzzy = &v
		}
	}
	return fuzzy, rows.Err()
}

func (s *Service) matchGradeIDByName(ctx context.Context, name string) (*uint64, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	nameNorm := normalizeMatchText(name)
	if nameNorm == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name
FROM k12_grade
WHERE status = 1 AND is_deleted = 0
ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fuzzy *uint64
	for rows.Next() {
		var (
			id uint64
			nm string
		)
		if err := rows.Scan(&id, &nm); err != nil {
			return nil, err
		}
		norm := normalizeMatchText(nm)
		if norm == "" {
			continue
		}
		if norm == nameNorm {
			v := id
			return &v, nil
		}
		if fuzzy == nil && (strings.Contains(norm, nameNorm) || strings.Contains(nameNorm, norm)) {
			v := id
			fuzzy = &v
		}
	}
	return fuzzy, rows.Err()
}

func (s *Service) CreatePaperWithImages(ctx context.Context, adminID uint64, in CreatePaperWithUploadInput, images []UploadImage) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	if adminID == 0 || in.K12SubjectID == 0 || strings.TrimSpace(in.Title) == "" || len(images) == 0 {
		return 0, ErrInvalidInput
	}
	recOut, recErr := s.recognizeQuestions(ctx, strings.TrimSpace(in.Title), images)
	var recognized []recognizedQuestion
	var recGroups []recognizedGroup
	if recErr != nil {
		// Keep upload available even when upstream recognition fails.
		recognized = nil
		recGroups = nil
	} else if recOut != nil {
		recognized = recOut.Questions
		recGroups = recOut.Groups
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
	pageMap := make(map[int]storedPage, len(images))
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
		pageMap[i+1] = storedPage{
			PageNo: i + 1,
			FileID: fileID,
			Rel:    rel,
			Abs:    abs,
		}
	}

	groupOrderToID := make(map[int]uint64)
	if len(recGroups) > 0 {
		sort.SliceStable(recGroups, func(i, j int) bool {
			if recGroups[i].GroupOrder != recGroups[j].GroupOrder {
				return recGroups[i].GroupOrder < recGroups[j].GroupOrder
			}
			return i < j
		})
		seenOrd := make(map[int]struct{})
		for _, g := range recGroups {
			ord := g.GroupOrder
			if ord < 0 {
				ord = 0
			}
			if _, dup := seenOrd[ord]; dup {
				continue
			}
			seenOrd[ord] = struct{}{}
			sk := strings.TrimSpace(g.SystemKind)
			if sk == "" {
				sk = "unknown"
			}
			var pageNo any
			if g.PageNo > 0 {
				pageNo = g.PageNo
			}
			var gid uint64
			err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question_group
  (paper_id, group_order, system_kind, title_label, description_text, page_no, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, 0)
RETURNING id`),
				paperID, ord, sk, emptyToNil(g.TitleLabel), emptyToNil(g.DescriptionText), pageNo,
				now, adminID, now, adminID,
			).Scan(&gid)
			if err != nil {
				return 0, err
			}
			groupOrderToID[ord] = gid
		}
	}

	orderedNos := make([]string, 0, len(recognized)+len(in.QuestionNos))
	recByNo := make(map[string]recognizedQuestion, len(recognized))
	for _, rq := range recognized {
		no := strings.TrimSpace(rq.QuestionNo)
		if no == "" {
			continue
		}
		if _, ok := recByNo[no]; ok {
			continue
		}
		recByNo[no] = rq
		orderedNos = append(orderedNos, no)
	}
	for _, qn := range normalizeQuestionNos(in.QuestionNos) {
		if _, ok := recByNo[qn]; ok {
			continue
		}
		orderedNos = append(orderedNos, qn)
	}

	qCount := 0
	for i, qn := range orderedNos {
		rq, hasRec := recByNo[qn]
		qType := "unknown"
		stem := ""
		ans := ""
		exp := ""
		var pFrom, pTo any
		qOrder := i + 1
		if hasRec {
			if strings.TrimSpace(rq.QuestionType) != "" {
				qType = strings.TrimSpace(rq.QuestionType)
			}
			stem = strings.TrimSpace(rq.StemText)
			ans = strings.TrimSpace(rq.AnswerText)
			exp = strings.TrimSpace(rq.Explanation)
			if rq.PageNo > 0 {
				pFrom = rq.PageNo
				pTo = rq.PageNo
			}
			if rq.QuestionOrder > 0 {
				qOrder = rq.QuestionOrder
			}
		}
		var gID any
		if hasRec && rq.GroupOrder > 0 {
			if id, ok := groupOrderToID[rq.GroupOrder]; ok && id > 0 {
				gID = id
			}
		}
		var questionID uint64
		err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question
  (paper_id, group_id, question_no, question_order, section_no, question_type, score, stem_text, answer_text, explanation_text,
   page_from, page_to, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, NULL, ?, NULL, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, 0)
ON CONFLICT (paper_id, question_no) DO UPDATE SET
  question_order = EXCLUDED.question_order,
  group_id = COALESCE(EXCLUDED.group_id, exam_source_question.group_id),
  question_type = EXCLUDED.question_type,
  stem_text = COALESCE(EXCLUDED.stem_text, exam_source_question.stem_text),
  answer_text = COALESCE(EXCLUDED.answer_text, exam_source_question.answer_text),
  explanation_text = COALESCE(EXCLUDED.explanation_text, exam_source_question.explanation_text),
  page_from = COALESCE(EXCLUDED.page_from, exam_source_question.page_from),
  page_to = COALESCE(EXCLUDED.page_to, exam_source_question.page_to),
  updated_at = EXCLUDED.updated_at,
  updated_by = EXCLUDED.updated_by
RETURNING id`),
			paperID, gID, qn, qOrder, qType, emptyToNil(stem), emptyToNil(ans), emptyToNil(exp), pFrom, pTo,
			now, adminID, now, adminID,
		).Scan(&questionID)
		if err != nil {
			return 0, err
		}
		qCount++

		if hasRec && rq.BBox != nil && rq.PageNo > 0 {
			if page, ok := pageMap[rq.PageNo]; ok {
				cropRel, cropAbs, cropErr := s.cropQuestionImage(page, paperID, questionID, *rq.BBox, now)
				if cropErr == nil && cropRel != "" {
					var cropFileID uint64
					pub := "/uploads/" + cropRel
					err = tx.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_file
  (storage_provider, bucket_name, object_key, public_url, original_filename, content_type, file_ext, size_bytes,
   status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES ('local', NULL, ?, ?, ?, 'image/png', 'png', ?, 1, ?, ?, ?, ?, 0)
RETURNING id`),
						cropRel, pub, filepath.Base(cropRel), fileSizeOrZero(cropAbs), now, adminID, now, adminID,
					).Scan(&cropFileID)
					if err != nil {
						return 0, err
					}
					bboxRaw, _ := json.Marshal(map[string]float64{
						"x": rq.BBox.X, "y": rq.BBox.Y, "w": rq.BBox.W, "h": rq.BBox.H,
					})
					_, err = tx.ExecContext(ctx, dbutil.Rebind(`
INSERT INTO exam_source_question_file
  (question_id, file_id, role, sort_no, page_no, bbox_norm, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, 'stem', 1, ?, ?::jsonb, 1, ?, ?, ?, ?, 0)
ON CONFLICT (question_id, file_id, role) DO NOTHING`),
						questionID, cropFileID, rq.PageNo, string(bboxRaw), now, adminID, now, adminID,
					)
					if err != nil {
						return 0, err
					}
					written = append(written, cropAbs)
				}
			}
		}
	}

	_, err = tx.ExecContext(ctx, dbutil.Rebind(`
UPDATE exam_source_paper
SET question_count = ?, updated_at = ?, updated_by = ?
WHERE id = ? AND is_deleted = 0`), qCount, now, adminID, paperID)
	if err != nil {
		return 0, err
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

func fileSizeOrZero(path string) int64 {
	st, err := os.Stat(path)
	if err != nil || st == nil {
		return 0
	}
	return st.Size()
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (s *Service) cropQuestionImage(page storedPage, paperID, questionID uint64, box recognizedBBox, now time.Time) (string, string, error) {
	f, err := os.Open(page.Abs)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return "", "", err
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 1 || h <= 1 {
		return "", "", ErrInvalidInput
	}
	x := int(clamp01(box.X) * float64(w))
	y := int(clamp01(box.Y) * float64(h))
	bw := int(clamp01(box.W) * float64(w))
	bh := int(clamp01(box.H) * float64(h))
	if bw <= 0 || bh <= 0 {
		return "", "", ErrInvalidInput
	}
	if x+bw > w {
		bw = w - x
	}
	if y+bh > h {
		bh = h - y
	}
	if bw <= 1 || bh <= 1 {
		return "", "", ErrInvalidInput
	}
	rect := image.Rect(0, 0, bw, bh)
	dst := image.NewRGBA(rect)
	draw.Draw(dst, rect, src, image.Point{X: b.Min.X + x, Y: b.Min.Y + y}, draw.Src)

	monthPath := now.Format("200601")
	cropRel := filepath.ToSlash(filepath.Join("exam-source", monthPath, "question-crops",
		fmt.Sprintf("%d_q%d_%s.png", paperID, questionID, randHex(6))))
	cropAbs := filepath.Join(s.uploadDir, cropRel)
	if err := os.MkdirAll(filepath.Dir(cropAbs), 0755); err != nil {
		return "", "", err
	}
	out, err := os.Create(cropAbs)
	if err != nil {
		return "", "", err
	}
	defer out.Close()
	if err := png.Encode(out, dst); err != nil {
		return "", "", err
	}
	return cropRel, cropAbs, nil
}

func buildVisionImages(images []UploadImage) []studentpaper.VisionImage {
	vision := make([]studentpaper.VisionImage, 0, len(images))
	for _, im := range images {
		if len(im.Bytes) == 0 {
			continue
		}
		vision = append(vision, studentpaper.VisionImage{
			MIME: mimeByExt(im.Filename),
			Data: im.Bytes,
		})
	}
	return vision
}

func (s *Service) recognizePaperMeta(ctx context.Context, titleHint string, images []UploadImage) (*recognizedPaperMeta, error) {
	if len(images) == 0 {
		return nil, nil
	}
	adapter := s.resolveRecognitionAdapter(ctx)
	if adapter == nil {
		return nil, nil
	}
	vision := buildVisionImages(images)
	if len(vision) == 0 {
		return nil, nil
	}
	prompt := fmt.Sprintf(`你是试卷元数据提取助手。请从上传的试卷整卷图片中提取元数据，只输出一个JSON对象：
{
  "title":"试卷标题（尽量完整）",
  "subject_name":"学科中文名，如数学/语文/英语/物理/化学/生物/历史/地理/政治",
  "grade_label":"年级，如高一/高二/高三/初三",
  "exam_year":2026,
  "term":"学期或场次，如一模/二模/期中/期末",
  "source_region":"地区（可空）",
  "source_school":"学校（可空）",
  "paper_type":"试卷类型（可空，默认 mock_exam）",
  "total_score":"总分（可空）",
  "duration_minutes":120
}
要求：
1) 只输出JSON，不要解释。
2) 无法确定可留空字符串或0。
3) 若已给出标题提示，可参考它：%s`, titleHint)
	res := adapter.Analyze(studentpaper.AnalyzeInput{
		ChatUserPrompt:          prompt,
		VisionImages:            vision,
		OptionalMaxOutputTokens: examSourceRecognizeMaxTokens,
		FileName:                titleHint,
		Subject:                 "exam_source",
		Stage:                   "admin",
	})
	raw := strings.TrimSpace(res.Out.RawContent)
	if raw == "" {
		raw = strings.TrimSpace(res.Out.Summary)
	}
	if raw == "" {
		return nil, nil
	}
	return parseRecognizedPaperMeta(raw)
}

func (s *Service) recognizeQuestions(ctx context.Context, title string, images []UploadImage) (*recognitionOutput, error) {
	if len(images) == 0 {
		return nil, nil
	}
	adapter := s.resolveRecognitionAdapter(ctx)
	if adapter == nil {
		return nil, nil
	}
	vision := buildVisionImages(images)
	if len(vision) == 0 {
		return nil, nil
	}
	prompt := fmt.Sprintf(`你是试卷结构化助手。请从上传的整卷图片中识别「大题分组」和「小题」，只输出一个JSON对象：
{
  "groups":[
    {
      "group_order":1,
      "system_kind":"single_choice",
      "title_label":"单项选择题",
      "description_text":"一、本题共7小题，每小题4分……（卷面上该大题前的完整说明文字，尽量逐字）",
      "page_no":1
    }
  ],
  "questions":[
    {
      "group_order":1,
      "question_no":"1",
      "question_order":1,
      "question_type":"single_choice",
      "page_no":1,
      "bbox_norm":{"x":0.1,"y":0.1,"w":0.8,"h":0.2},
      "stem_text":"题干摘要",
      "answer_text":"答案（若可识别）",
      "explanation_text":"解析（若可识别）"
    }
  ]
}
字段说明：
- groups：卷面上每个大题区块（如「一、单项选择题」整段说明）。group_order 从 1 递增。
- system_kind：系统题型枚举，使用英文小写：single_choice（单选）、multi_choice（多选）、fill_blank、short_answer、essay、unknown 等。
- title_label：短标题，如「单项选择题」。
- description_text：该大题下完整说明（含分值、作答要求等）。
- questions.group_order：所属大题的 group_order，必须与 groups 中某一组一致；若无法分段则省略 groups 或令 group_order 为 0。
要求：
1) 只输出JSON，不要解释。
2) bbox_norm 为相对坐标，x/y/w/h 范围 0~1。
3) page_no 从1开始。
4) bbox 需要“尽可能紧贴当前题目内容”，不要跨到上一题/下一题的文字，不要把旁栏或页眉页脚包含进来。
5) 无法确定的字段可留空字符串，但 question_no/page_no/bbox_norm 尽量给出。试卷标题参考：%s。`, title)

	res := adapter.Analyze(studentpaper.AnalyzeInput{
		ChatUserPrompt:          prompt,
		VisionImages:            vision,
		OptionalMaxOutputTokens: examSourceRecognizeMaxTokens,
		FileName:                title,
		Subject:                 "exam_source",
		Stage:                   "admin",
	})
	raw := strings.TrimSpace(res.Out.RawContent)
	if raw == "" {
		raw = strings.TrimSpace(res.Out.Summary)
	}
	if raw == "" {
		return nil, nil
	}
	out, err := parseRecognitionResult(raw)
	if err != nil {
		return nil, err
	}
	return refineRecognitionOutput(out), nil
}

func (s *Service) resolveRecognitionAdapter(ctx context.Context) studentpaper.AnalysisAdapter {
	if s == nil || s.db == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var (
		modelID uint64
		url     string
		model   string
		secret  string
	)
	err := s.db.QueryRowContext(ctx2, `
SELECT id, url, model, app_secret
FROM ai_provider_model
WHERE status = 1 AND is_deleted = 0
ORDER BY id DESC
LIMIT 1
`).Scan(&modelID, &url, &model, &secret)
	if err != nil || modelID == 0 || strings.TrimSpace(url) == "" {
		return nil
	}
	return studentpaper.NewHTTPAnalysisAdapter(strings.TrimSpace(url), 180*time.Second, strings.TrimSpace(secret), strings.TrimSpace(model))
}

func extractFirstJSONObject(raw string) (string, bool) {
	s := strings.TrimSpace(strings.TrimPrefix(raw, "\uFEFF"))
	start := strings.Index(s, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if c == '\\' {
				escape = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1]), true
			}
		}
	}
	return "", false
}

func parseRecognitionResult(raw string) (*recognitionOutput, error) {
	c := strings.TrimSpace(raw)
	if strings.HasPrefix(c, "```") {
		c = strings.TrimPrefix(c, "```")
		c = strings.TrimPrefix(strings.TrimSpace(c), "json")
		c = strings.TrimSpace(c)
		if i := strings.LastIndex(c, "```"); i >= 0 {
			c = strings.TrimSpace(c[:i])
		}
	}
	if j, ok := extractFirstJSONObject(c); ok {
		c = j
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(c), &root); err != nil {
		return nil, err
	}
	out := &recognitionOutput{
		Groups:    nil,
		Questions: nil,
	}
	if gArr, ok := root["groups"].([]any); ok && len(gArr) > 0 {
		out.Groups = make([]recognizedGroup, 0, len(gArr))
		for _, it := range gArr {
			m, _ := it.(map[string]any)
			if m == nil {
				continue
			}
			g := recognizedGroup{
				GroupOrder:      toInt(m["group_order"]),
				SystemKind:      strings.TrimSpace(toString(m["system_kind"])),
				TitleLabel:      strings.TrimSpace(toString(m["title_label"])),
				DescriptionText: strings.TrimSpace(toString(m["description_text"])),
				PageNo:          toInt(m["page_no"]),
			}
			if g.SystemKind == "" {
				g.SystemKind = "unknown"
			}
			out.Groups = append(out.Groups, g)
		}
	}
	arr, _ := root["questions"].([]any)
	out.Questions = make([]recognizedQuestion, 0, len(arr))
	for _, it := range arr {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		qn := strings.TrimSpace(toString(m["question_no"]))
		if qn == "" {
			continue
		}
		rq := recognizedQuestion{
			GroupOrder:    toInt(m["group_order"]),
			QuestionNo:    qn,
			QuestionOrder: toInt(m["question_order"]),
			QuestionType:  strings.TrimSpace(toString(m["question_type"])),
			PageNo:        toInt(m["page_no"]),
			StemText:      strings.TrimSpace(toString(m["stem_text"])),
			AnswerText:    strings.TrimSpace(toString(m["answer_text"])),
			Explanation:   strings.TrimSpace(toString(m["explanation_text"])),
		}
		if rq.QuestionType == "" {
			rq.QuestionType = "unknown"
		}
		if bbox := parseBBox(m["bbox_norm"]); bbox != nil {
			rq.BBox = bbox
		}
		out.Questions = append(out.Questions, rq)
	}
	return out, nil
}

func refineRecognitionOutput(out *recognitionOutput) *recognitionOutput {
	if out == nil {
		return &recognitionOutput{}
	}
	// Normalize and tighten bbox for each question first.
	for i := range out.Questions {
		q := &out.Questions[i]
		q.QuestionNo = strings.TrimSpace(q.QuestionNo)
		if q.QuestionType == "" {
			q.QuestionType = "unknown"
		}
		if q.BBox != nil {
			b := normalizeBBox(*q.BBox)
			b = insetBBox(b, 0.006, 0.005)
			q.BBox = &b
		}
	}

	// On the same page, limit current bbox bottom by the next question top to reduce cross-question bleed.
	pageIdx := make(map[int][]int)
	for i := range out.Questions {
		if out.Questions[i].PageNo > 0 && out.Questions[i].BBox != nil {
			pageIdx[out.Questions[i].PageNo] = append(pageIdx[out.Questions[i].PageNo], i)
		}
	}
	for _, idx := range pageIdx {
		sort.SliceStable(idx, func(i, j int) bool {
			ai := out.Questions[idx[i]]
			aj := out.Questions[idx[j]]
			if ai.BBox.Y != aj.BBox.Y {
				return ai.BBox.Y < aj.BBox.Y
			}
			return ai.BBox.X < aj.BBox.X
		})
		for k := 0; k+1 < len(idx); k++ {
			cur := &out.Questions[idx[k]]
			next := &out.Questions[idx[k+1]]
			if cur.BBox == nil || next.BBox == nil {
				continue
			}
			maxBottom := next.BBox.Y - 0.004
			minBottom := cur.BBox.Y + 0.02
			if maxBottom > minBottom {
				currentBottom := cur.BBox.Y + cur.BBox.H
				if currentBottom > maxBottom {
					cur.BBox.H = maxBottom - cur.BBox.Y
				}
			}
			b := normalizeBBox(*cur.BBox)
			cur.BBox = &b
		}
	}

	// Ensure group_order continuity and synthesize groups if model didn't output.
	sort.SliceStable(out.Questions, func(i, j int) bool {
		oi := out.Questions[i].QuestionOrder
		oj := out.Questions[j].QuestionOrder
		if oi > 0 && oj > 0 && oi != oj {
			return oi < oj
		}
		if out.Questions[i].PageNo != out.Questions[j].PageNo {
			return out.Questions[i].PageNo < out.Questions[j].PageNo
		}
		return i < j
	})
	nextOrder := 1
	lastGroup := 0
	lastType := ""
	for i := range out.Questions {
		q := &out.Questions[i]
		if q.QuestionOrder <= 0 {
			q.QuestionOrder = nextOrder
		}
		nextOrder = maxInt(nextOrder, q.QuestionOrder+1)
		if q.GroupOrder <= 0 {
			if lastGroup == 0 {
				lastGroup = 1
			} else if lastType != "" && lastType != q.QuestionType {
				lastGroup++
			}
			q.GroupOrder = lastGroup
		} else {
			lastGroup = q.GroupOrder
		}
		lastType = q.QuestionType
	}

	if len(out.Groups) == 0 {
		out.Groups = synthesizeGroupsFromQuestions(out.Questions)
	} else {
		seen := make(map[int]struct{}, len(out.Groups))
		normGroups := make([]recognizedGroup, 0, len(out.Groups))
		for _, g := range out.Groups {
			ord := g.GroupOrder
			if ord <= 0 {
				continue
			}
			if _, ok := seen[ord]; ok {
				continue
			}
			seen[ord] = struct{}{}
			g.SystemKind = normalizeSystemKind(g.SystemKind)
			if strings.TrimSpace(g.TitleLabel) == "" {
				g.TitleLabel = defaultGroupTitle(g.SystemKind)
			}
			normGroups = append(normGroups, g)
		}
		if len(normGroups) == 0 {
			out.Groups = synthesizeGroupsFromQuestions(out.Questions)
		} else {
			sort.SliceStable(normGroups, func(i, j int) bool { return normGroups[i].GroupOrder < normGroups[j].GroupOrder })
			out.Groups = normGroups
		}
	}
	return out
}

func synthesizeGroupsFromQuestions(questions []recognizedQuestion) []recognizedGroup {
	if len(questions) == 0 {
		return nil
	}
	type agg struct {
		systemKind string
		pageNo     int
	}
	byOrder := make(map[int]agg)
	for _, q := range questions {
		ord := q.GroupOrder
		if ord <= 0 {
			continue
		}
		if _, ok := byOrder[ord]; ok {
			continue
		}
		byOrder[ord] = agg{
			systemKind: normalizeSystemKind(q.QuestionType),
			pageNo:     q.PageNo,
		}
	}
	if len(byOrder) == 0 {
		return nil
	}
	ords := make([]int, 0, len(byOrder))
	for ord := range byOrder {
		ords = append(ords, ord)
	}
	sort.Ints(ords)
	out := make([]recognizedGroup, 0, len(ords))
	for _, ord := range ords {
		a := byOrder[ord]
		out = append(out, recognizedGroup{
			GroupOrder: ord,
			SystemKind: a.systemKind,
			TitleLabel: defaultGroupTitle(a.systemKind),
			PageNo:     a.pageNo,
		})
	}
	return out
}

func normalizeSystemKind(kind string) string {
	t := strings.ToLower(strings.TrimSpace(kind))
	switch t {
	case "", "未知", "unknown":
		return "unknown"
	case "single_choice", "single", "单选", "单项选择", "单项选择题":
		return "single_choice"
	case "multi_choice", "multiple_choice", "多选", "多项选择", "多项选择题":
		return "multi_choice"
	case "fill_blank", "填空", "填空题":
		return "fill_blank"
	case "short_answer", "简答", "简答题":
		return "short_answer"
	case "essay", "作文":
		return "essay"
	default:
		return t
	}
}

func defaultGroupTitle(kind string) string {
	switch normalizeSystemKind(kind) {
	case "single_choice":
		return "单项选择题"
	case "multi_choice":
		return "多项选择题"
	case "fill_blank":
		return "填空题"
	case "short_answer":
		return "简答题"
	case "essay":
		return "作文题"
	default:
		return "试题"
	}
}

func normalizeBBox(b recognizedBBox) recognizedBBox {
	x := clamp01(b.X)
	y := clamp01(b.Y)
	w := clamp01(b.W)
	h := clamp01(b.H)
	if x+w > 1 {
		w = 1 - x
	}
	if y+h > 1 {
		h = 1 - y
	}
	if w < 0.01 {
		w = 0.01
	}
	if h < 0.01 {
		h = 0.01
	}
	return recognizedBBox{X: x, Y: y, W: w, H: h}
}

func insetBBox(b recognizedBBox, padX, padY float64) recognizedBBox {
	if padX <= 0 && padY <= 0 {
		return b
	}
	maxPadX := b.W * 0.2
	maxPadY := b.H * 0.2
	if padX > maxPadX {
		padX = maxPadX
	}
	if padY > maxPadY {
		padY = maxPadY
	}
	b.X += padX
	b.Y += padY
	b.W -= padX * 2
	b.H -= padY * 2
	return normalizeBBox(b)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parseRecognizedPaperMeta(raw string) (*recognizedPaperMeta, error) {
	c := strings.TrimSpace(raw)
	if strings.HasPrefix(c, "```") {
		c = strings.TrimPrefix(c, "```")
		c = strings.TrimPrefix(strings.TrimSpace(c), "json")
		c = strings.TrimSpace(c)
		if i := strings.LastIndex(c, "```"); i >= 0 {
			c = strings.TrimSpace(c[:i])
		}
	}
	if j, ok := extractFirstJSONObject(c); ok {
		c = j
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(c), &m); err != nil {
		return nil, err
	}
	out := &recognizedPaperMeta{
		Title:           strings.TrimSpace(toString(m["title"])),
		SubjectName:     strings.TrimSpace(toString(m["subject_name"])),
		GradeLabel:      strings.TrimSpace(toString(m["grade_label"])),
		ExamYear:        toInt(m["exam_year"]),
		Term:            strings.TrimSpace(toString(m["term"])),
		SourceRegion:    strings.TrimSpace(toString(m["source_region"])),
		SourceSchool:    strings.TrimSpace(toString(m["source_school"])),
		PaperType:       strings.TrimSpace(toString(m["paper_type"])),
		TotalScore:      strings.TrimSpace(toString(m["total_score"])),
		DurationMinutes: toInt(m["duration_minutes"]),
	}
	return out, nil
}

func parseBBox(v any) *recognizedBBox {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]any); ok && len(arr) >= 4 {
		return &recognizedBBox{
			X: toFloat(arr[0]),
			Y: toFloat(arr[1]),
			W: toFloat(arr[2]),
			H: toFloat(arr[3]),
		}
	}
	m, _ := v.(map[string]any)
	if m == nil {
		return nil
	}
	return &recognizedBBox{
		X: toFloat(m["x"]),
		Y: toFloat(m["y"]),
		W: toFloat(m["w"]),
		H: toFloat(m["h"]),
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	default:
		return ""
	}
}

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(x))
		return n
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n
	default:
		return 0
	}
}

func mimeByExt(name string) string {
	switch strings.ToLower(strings.TrimSpace(filepath.Ext(strings.TrimSpace(name)))) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}
