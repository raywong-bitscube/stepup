package studentpaper

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/service/ailog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/prompttemplate"
)

type Paper struct {
	ID         uint64    `json:"id"`
	Identifier string    `json:"identifier"`
	Subject    string    `json:"subject"`
	Stage      string    `json:"stage"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	CreatedAt  time.Time `json:"created_at"`
}

type Analysis struct {
	PaperID         uint64    `json:"paper_id"`
	Status          string    `json:"status"`
	AIModelSnapshot any       `json:"ai_model_snapshot"`
	WeakPoints      []string  `json:"weak_points"`
	Summary         string    `json:"summary"`
	UpdatedAt       time.Time `json:"updated_at"`
	ImprovementPlan []string  `json:"improvement_plan"`
}

type Service struct {
	cfg      config.Config
	db       *sql.DB
	aiLog    *ailog.Writer
	mu       sync.RWMutex
	nextID   uint64
	papers   map[uint64]Paper
	analysis map[uint64]Analysis
}

func New(cfg config.Config, db *sql.DB) *Service {
	return &Service{
		cfg:      cfg,
		db:       db,
		aiLog:    ailog.NewWriter(db),
		nextID:   1,
		papers:   map[uint64]Paper{},
		analysis: map[uint64]Analysis{},
	}
}

// activeModel holds DB-backed endpoint metadata (no secrets) for paper_analysis snapshot.
type activeModel struct {
	ID   uint64
	Name string
	URL  string
}

func (s *Service) resolveAdapter(ctx context.Context) (AnalysisAdapter, *activeModel) {
	if !strings.EqualFold(s.cfg.AnalysisAdapter, "http") {
		return MockAnalysisAdapter{}, nil
	}
	if s.db != nil {
		var modelID uint64
		var name, url, chatModel, appSecret string
		err := s.db.QueryRowContext(ctx, `
SELECT id, name, url, model, app_secret
FROM ai_model
WHERE status = 1 AND is_deleted = 0
ORDER BY id DESC
LIMIT 1
`).Scan(&modelID, &name, &url, &chatModel, &appSecret)
		if err == nil {
			url = strings.TrimSpace(url)
			if url != "" {
				return NewHTTPAnalysisAdapter(url, s.cfg.AIRequestTimeout, appSecret, chatModel), &activeModel{
					ID:   modelID,
					Name: strings.TrimSpace(name),
					URL:  url,
				}
			}
		}
	}
	if ep := strings.TrimSpace(s.cfg.AIEndpoint); ep != "" {
		return NewHTTPAnalysisAdapter(ep, s.cfg.AIRequestTimeout, "", ""), nil
	}
	return MockAnalysisAdapter{}, nil
}

func applyModelSnapshot(out AnalyzeOutput, meta *activeModel) AnalyzeOutput {
	if meta != nil {
		out.ModelSnapshot = map[string]any{
			"name": meta.Name,
			"url":  meta.URL,
		}
	}
	return out
}

func (s *Service) paperChatUserPrompt(ctx context.Context, in AnalyzeInput) string {
	tpl := prompttemplate.DefaultPaperAnalyzeChatUser()
	if s.db != nil {
		ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if c, err := prompttemplate.GetActiveContent(ctx2, s.db, prompttemplate.KeyPaperAnalyzeChatUser); err == nil && c != "" {
			tpl = c
		}
	}
	// 有试卷图片时由多模态消息的 image_url 传入，由模型自行识图/OCR；不在此用 %file_content 预填文案。
	return prompttemplate.Expand(tpl, map[string]string{
		"subject":   in.Subject,
		"stage":     in.Stage,
		"file_name": in.FileName,
	})
}

func (s *Service) writeAILog(ctx context.Context, meta *activeModel, studentID, paperID uint64, subject, stage, originalFileName, filePath string, out AnalyzeOutput, tr AnalyzeTrace) {
	if s.aiLog == nil {
		return
	}
	var aid *uint64
	nameSnap := ""
	if meta != nil {
		if meta.ID != 0 {
			v := meta.ID
			aid = &v
		}
		nameSnap = meta.Name
	}
	metaReq := map[string]any{
		"subject":   subject,
		"stage":     stage,
		"file_name": originalFileName,
	}
	if strings.TrimSpace(filePath) != "" {
		metaReq["file_path"] = filePath
	}
	reqM, _ := json.Marshal(metaReq)
	respM, _ := json.Marshal(map[string]any{
		"summary_len":     len(out.Summary),
		"weak_points_n":   len(out.WeakPoints),
		"plan_n":          len(out.ImprovementPlan),
		"raw_content_len": len(out.RawContent),
	})
	var httpSt *int
	if tr.HTTPStatus != 0 {
		v := tr.HTTPStatus
		httpSt = &v
	}
	lat := tr.LatencyMS
	latPtr := &lat
	var paperPtr *uint64
	if paperID != 0 {
		v := paperID
		paperPtr = &v
	}
	var stuPtr *uint64
	if studentID != 0 {
		v := studentID
		stuPtr = &v
	}
	s.aiLog.Write(ctx, ailog.InsertRow{
		AIModelID:        aid,
		ModelNameSnap:    nameSnap,
		Action:           "paper_analyze",
		AdapterKind:      tr.AdapterKind,
		ResultStatus:     tr.ResultStatus,
		HTTPStatus:       httpSt,
		LatencyMS:        latPtr,
		ErrorPhase:       tr.ErrorPhase,
		ErrorMessage:     tr.ErrorMessage,
		EndpointHost:     tr.EndpointHost,
		ChatModel:        tr.ChatModel,
		FallbackToMock:   tr.FallbackToMock,
		PaperID:          paperPtr,
		StudentID:        stuPtr,
		RequestMetaJSON:  reqM,
		ResponseMetaJSON: respM,
		RequestBody:      tr.RequestBody,
		ResponseBody:     tr.ResponseBody,
	})
}

func (s *Service) Create(identifier, subject, stage, originalFileName string, fileSize int64, contentType string, fileBytes []byte) (Paper, error) {
	if s.db != nil {
		return s.createDB(identifier, subject, stage, originalFileName, fileSize, contentType, fileBytes)
	}

	ctx := context.Background()
	adapter, meta := s.resolveAdapter(ctx)
	vmime, vdata := visionImageForModel(originalFileName, contentType, fileBytes)

	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	s.nextID++

	p := Paper{
		ID:         id,
		Identifier: identifier,
		Subject:    subject,
		Stage:      stage,
		FileName:   originalFileName,
		FileSize:   fileSize,
		CreatedAt:  time.Now(),
	}
	ain := AnalyzeInput{
		Subject:   subject,
		Stage:     stage,
		FileName:  originalFileName,
		ImageMIME: vmime,
		ImageData: vdata,
	}
	ain.ChatUserPrompt = s.paperChatUserPrompt(ctx, ain)
	result := adapter.Analyze(ain)
	out := applyModelSnapshot(result.Out, meta)
	s.analysis[id] = Analysis{
		PaperID:         id,
		Status:          "completed",
		AIModelSnapshot: out.ModelSnapshot,
		Summary:         out.Summary,
		WeakPoints:      out.WeakPoints,
		ImprovementPlan: out.ImprovementPlan,
		UpdatedAt:       time.Now(),
	}
	return p, nil
}

func (s *Service) List(identifier string) []Paper {
	if s.db != nil {
		if items, err := s.listDB(identifier); err == nil {
			return items
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Paper, 0, len(s.papers))
	for _, p := range s.papers {
		if p.Identifier == identifier {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Paper) int {
		return int(b.ID - a.ID)
	})
	return out
}

func (s *Service) GetAnalysis(identifier, paperIDRaw string) (Analysis, error) {
	if s.db != nil {
		return s.getAnalysisDB(identifier, paperIDRaw)
	}

	pid, err := strconv.ParseUint(paperIDRaw, 10, 64)
	if err != nil {
		return Analysis{}, fmt.Errorf("invalid paper id")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.papers[pid]
	if !ok || p.Identifier != identifier {
		return Analysis{}, fmt.Errorf("not found")
	}
	a, ok := s.analysis[pid]
	if !ok {
		return Analysis{}, fmt.Errorf("not found")
	}
	return a, nil
}

func (s *Service) GetPlan(identifier, paperIDRaw string) (map[string]any, error) {
	if s.db != nil {
		return s.getPlanDB(identifier, paperIDRaw)
	}

	a, err := s.GetAnalysis(identifier, paperIDRaw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"paper_id": a.PaperID,
		"plan":     a.ImprovementPlan,
		"updated":  a.UpdatedAt,
	}, nil
}

func (s *Service) createDB(identifier, subject, stage, originalFileName string, fileSize int64, contentType string, fileBytes []byte) (Paper, error) {
	// Short deadline only for pre-analyze DB lookups. AI Analyze may take AIRequestTimeout (e.g. 30s+);
	// reusing one 5s context caused Insert to fail with context canceled after long vision calls.
	lookupCtx, lookupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	var (
		studentID uint64
		stageID   uint64
	)
	err := s.db.QueryRowContext(lookupCtx, `
SELECT id, stage_id
FROM student
WHERE (phone = ? OR email = ?) AND status = 1 AND is_deleted = 0
LIMIT 1
`, identifier, identifier).Scan(&studentID, &stageID)
	if err != nil {
		lookupCancel()
		return Paper{}, err
	}

	var subjectID uint64
	err = s.db.QueryRowContext(lookupCtx, `
SELECT id
FROM subject
WHERE name = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`, subject).Scan(&subjectID)
	if err != nil {
		lookupCancel()
		return Paper{}, err
	}

	vmime, vdata := visionImageForModel(originalFileName, contentType, fileBytes)

	now := time.Now()
	fileType := detectFileType(originalFileName)
	adapter, meta := s.resolveAdapter(lookupCtx)
	ain := AnalyzeInput{
		Subject:   subject,
		Stage:     stage,
		FileName:  originalFileName,
		ImageMIME: vmime,
		ImageData: vdata,
	}
	ain.ChatUserPrompt = s.paperChatUserPrompt(lookupCtx, ain)
	lookupCancel()

	analyzeResult := adapter.Analyze(ain)
	analysisOut := applyModelSnapshot(analyzeResult.Out, meta)

	logAITrace := func(paperID uint64, filePath string) {
		s.writeAILog(context.Background(), meta, studentID, paperID, subject, stage, originalFileName, filePath, analysisOut, analyzeResult.Trace)
	}

	uploadDir := strings.TrimSpace(s.cfg.UploadDir)
	if uploadDir == "" {
		uploadDir = "data/uploads"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logAITrace(0, "")
		return Paper{}, err
	}
	stored := uniqueStoredFileName(originalFileName)
	if err := os.WriteFile(filepath.Join(uploadDir, stored), fileBytes, 0644); err != nil {
		logAITrace(0, "")
		return Paper{}, err
	}
	fileURL := "/uploads/" + stored

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer writeCancel()
	res, err := s.db.ExecContext(writeCtx, `
INSERT INTO exam_paper
  (student_id, subject_id, file_url, file_type, score, exam_date, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, NULL, CURDATE(), ?, ?, ?, ?, 0)
`, studentID, subjectID, fileURL, fileType, now, studentID, now, studentID)
	if err != nil {
		logAITrace(0, fileURL)
		return Paper{}, err
	}
	paperID, _ := res.LastInsertId()

	logAITrace(uint64(paperID), fileURL)

	snapshotRaw, _ := json.Marshal(analysisOut.ModelSnapshot)
	weakRaw, _ := json.Marshal(analysisOut.WeakPoints)
	planRaw, _ := json.Marshal(analysisOut.ImprovementPlan)
	aiResponseRaw, _ := json.Marshal(map[string]any{
		"summary":     analysisOut.Summary,
		"weak_points": analysisOut.WeakPoints,
	})

	_, _ = s.db.ExecContext(writeCtx, `
INSERT INTO paper_analysis
  (paper_id, ai_model_snapshot, raw_content, ai_response, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, 2, ?, ?, ?, ?, 0)
`, paperID, string(snapshotRaw), analysisOut.RawContent, string(aiResponseRaw), now, studentID, now, studentID)

	_, _ = s.db.ExecContext(writeCtx, `
INSERT INTO improvement_plan
  (paper_id, plan_content, weak_points, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, ?, ?, ?, 0)
`, paperID, string(planRaw), string(weakRaw), now, studentID, now, studentID)

	return Paper{
		ID:         uint64(paperID),
		Identifier: identifier,
		Subject:    subject,
		Stage:      stageName(stage, stageID),
		FileName:   originalFileName,
		FileSize:   fileSize,
		CreatedAt:  now,
	}, nil
}

func (s *Service) listDB(identifier string) ([]Paper, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, subj.name, stg.name, p.file_url, p.created_at
FROM exam_paper p
JOIN student stu ON stu.id = p.student_id
JOIN subject subj ON subj.id = p.subject_id
JOIN stage stg ON stg.id = stu.stage_id
WHERE (stu.phone = ? OR stu.email = ?) AND p.is_deleted = 0
ORDER BY p.id DESC
`, identifier, identifier)
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
			ID:         id,
			Identifier: identifier,
			Subject:    subject,
			Stage:      stage,
			FileName:   filepath.Base(fileURL),
			FileSize:   0,
			CreatedAt:  createdAt,
		})
	}
	return out, nil
}

func (s *Service) getAnalysisDB(identifier, paperIDRaw string) (Analysis, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pid, err := strconv.ParseUint(paperIDRaw, 10, 64)
	if err != nil {
		return Analysis{}, fmt.Errorf("invalid paper id")
	}

	var (
		aiSnapshot string
		aiResp     string
		status     int
		updatedAt  time.Time
	)
	err = s.db.QueryRowContext(ctx, `
SELECT pa.ai_model_snapshot, pa.ai_response, pa.status, pa.updated_at
FROM paper_analysis pa
JOIN exam_paper p ON p.id = pa.paper_id
JOIN student stu ON stu.id = p.student_id
WHERE pa.paper_id = ?
  AND (stu.phone = ? OR stu.email = ?)
  AND pa.is_deleted = 0
  AND p.is_deleted = 0
LIMIT 1
`, pid, identifier, identifier).Scan(&aiSnapshot, &aiResp, &status, &updatedAt)
	if err != nil {
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

	planMap, _ := s.getPlanDB(identifier, paperIDRaw)
	plan := make([]string, 0)
	if raw, ok := planMap["plan"].([]string); ok {
		plan = raw
	}

	return Analysis{
		PaperID:         pid,
		Status:          mapAnalysisStatus(status),
		AIModelSnapshot: snapshot,
		WeakPoints:      weak,
		Summary:         fmt.Sprintf("%v", response["summary"]),
		UpdatedAt:       updatedAt,
		ImprovementPlan: plan,
	}, nil
}

func (s *Service) getPlanDB(identifier, paperIDRaw string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pid, err := strconv.ParseUint(paperIDRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid paper id")
	}

	var (
		planRaw   string
		updatedAt time.Time
	)
	err = s.db.QueryRowContext(ctx, `
SELECT ip.plan_content, ip.updated_at
FROM improvement_plan ip
JOIN exam_paper p ON p.id = ip.paper_id
JOIN student stu ON stu.id = p.student_id
WHERE ip.paper_id = ?
  AND (stu.phone = ? OR stu.email = ?)
  AND ip.is_deleted = 0
  AND p.is_deleted = 0
LIMIT 1
`, pid, identifier, identifier).Scan(&planRaw, &updatedAt)
	if err != nil {
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

func detectFileType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".pdf" {
		return "pdf"
	}
	return "image"
}

func stageName(input string, stageID uint64) string {
	if strings.TrimSpace(input) != "" {
		return input
	}
	return fmt.Sprintf("stage-%d", stageID)
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
