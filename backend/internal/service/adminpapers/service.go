package adminpapers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
)

var (
	ErrNoDatabase = errors.New("database_required")
	ErrNotFound   = errors.New("not_found")
)

type Paper struct {
	ID        uint64    `json:"id"`
	Subject   string    `json:"subject"`
	Stage     string    `json:"stage"`
	FileURL   string    `json:"file_url"`
	FileName  string    `json:"file_name"`
	CreatedAt time.Time `json:"created_at"`
}

type Analysis struct {
	PaperID         uint64         `json:"paper_id"`
	Status          string         `json:"status"`
	AIModelSnapshot map[string]any `json:"ai_model_snapshot"`
	WeakPoints      []string       `json:"weak_points"`
	Summary         string         `json:"summary"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ImprovementPlan []string       `json:"improvement_plan"`
}

type Service struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) studentExists(ctx context.Context, studentID uint64) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT 1 FROM sys_user WHERE id = ? AND is_deleted = 0 LIMIT 1`), studentID).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) List(ctx context.Context, studentID uint64) ([]Paper, error) {
	if s.db == nil {
		return nil, ErrNoDatabase
	}
	ok, err := s.studentExists(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT p.id, subj.name, stg.name, p.file_url, p.created_at
FROM student_exam_paper p
JOIN sys_user stu ON stu.id = p.sys_user_id
JOIN k12_subject subj ON subj.id = p.k12_subject_id
JOIN k12_grade stg ON stg.id = stu.k12_grade_id
WHERE p.sys_user_id = ? AND p.is_deleted = 0
ORDER BY p.id DESC
`), studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Paper, 0, 16)
	for rows.Next() {
		var (
			id        uint64
			subject   string
			stage     string
			fileURL   string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &subject, &stage, &fileURL, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, Paper{
			ID:        id,
			Subject:   subject,
			Stage:     stage,
			FileURL:   fileURL,
			FileName:  filepath.Base(fileURL),
			CreatedAt: createdAt,
		})
	}
	return out, rows.Err()
}

func mapAnalysisStatus(v int) string {
	switch v {
	case 0:
		return "pending"
	case 1:
		return "processing"
	case 2:
		return "completed"
	case 3:
		return "failed"
	default:
		return "unknown"
	}
}

func (s *Service) GetAnalysis(ctx context.Context, studentID uint64, paperIDRaw string) (Analysis, error) {
	if s.db == nil {
		return Analysis{}, ErrNoDatabase
	}
	pid, err := strconv.ParseUint(paperIDRaw, 10, 64)
	if err != nil {
		return Analysis{}, ErrNotFound
	}
	ok, err := s.studentExists(ctx, studentID)
	if err != nil {
		return Analysis{}, err
	}
	if !ok {
		return Analysis{}, ErrNotFound
	}

	var (
		aiSnapshot string
		aiResp     string
		status     int
		updatedAt  time.Time
	)
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT pa.ai_model_snapshot, pa.ai_response, pa.status, pa.updated_at
FROM student_paper_analysis pa
JOIN student_exam_paper p ON p.id = pa.paper_id
WHERE pa.paper_id = ?
  AND p.sys_user_id = ?
  AND pa.is_deleted = 0
  AND p.is_deleted = 0
LIMIT 1
`), pid, studentID).Scan(&aiSnapshot, &aiResp, &status, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Analysis{}, ErrNotFound
		}
		return Analysis{}, err
	}

	var snapshot map[string]any
	_ = json.Unmarshal([]byte(aiSnapshot), &snapshot)

	var response map[string]any
	_ = json.Unmarshal([]byte(aiResp), &response)

	weak := make([]string, 0)
	if raw, ok := response["weak_points"].([]any); ok {
		for _, item := range raw {
			weak = append(weak, fmt.Sprintf("%v", item))
		}
	}
	summary := fmt.Sprintf("%v", response["summary"])

	plan, _ := s.GetPlanSlice(ctx, studentID, pid)

	return Analysis{
		PaperID:         pid,
		Status:          mapAnalysisStatus(status),
		AIModelSnapshot: snapshot,
		WeakPoints:      weak,
		Summary:         summary,
		UpdatedAt:       updatedAt,
		ImprovementPlan: plan,
	}, nil
}

func (s *Service) GetPlanSlice(ctx context.Context, studentID, paperID uint64) ([]string, error) {
	var (
		planRaw   string
		updatedAt time.Time
	)
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT ip.plan_content, ip.updated_at
FROM student_improvement_plan ip
JOIN student_exam_paper p ON p.id = ip.paper_id
WHERE ip.paper_id = ?
  AND p.sys_user_id = ?
  AND ip.is_deleted = 0
  AND p.is_deleted = 0
LIMIT 1
`), paperID, studentID).Scan(&planRaw, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	plan := make([]string, 0)
	_ = json.Unmarshal([]byte(planRaw), &plan)
	return plan, nil
}

func (s *Service) GetPlan(ctx context.Context, studentID uint64, paperIDRaw string) (map[string]any, error) {
	if s.db == nil {
		return nil, ErrNoDatabase
	}
	pid, err := strconv.ParseUint(paperIDRaw, 10, 64)
	if err != nil {
		return nil, ErrNotFound
	}
	ok, err := s.studentExists(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var (
		planRaw   string
		updatedAt time.Time
	)
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT ip.plan_content, ip.updated_at
FROM student_improvement_plan ip
JOIN student_exam_paper p ON p.id = ip.paper_id
WHERE ip.paper_id = ?
  AND p.sys_user_id = ?
  AND ip.is_deleted = 0
  AND p.is_deleted = 0
LIMIT 1
`), pid, studentID).Scan(&planRaw, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	plan := make([]string, 0)
	_ = json.Unmarshal([]byte(planRaw), &plan)
	return map[string]any{
		"paper_id": pid,
		"plan":     plan,
		"updated":  updatedAt,
	}, nil
}
