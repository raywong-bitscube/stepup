package studentessayoutline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
	"github.com/raywong-bitscube/stepup/backend/internal/service/ailog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/prompttemplate"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
)

var (
	ErrDatabaseRequired = errors.New("essay outline requires database")
	ErrInvalidInput     = errors.New("invalid input")
	ErrPracticeNotFound = errors.New("essay outline practice not found")
)

// PracticeSummary is one row for the student history list (lightweight).
type PracticeSummary struct {
	ID           uint64    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	TopicLabel   string    `json:"topic_label"`
	TopicSource  string    `json:"topic_source"`
	TopicPreview string    `json:"topic_preview"`
}

// PracticeDetail is a full saved practice record for the detail view.
type PracticeDetail struct {
	ID          uint64         `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	TopicText   string         `json:"topic_text"`
	TopicLabel  string         `json:"topic_label"`
	TopicSource string         `json:"topic_source"`
	Genre       string         `json:"genre,omitempty"`
	TaskType    string         `json:"task_type,omitempty"`
	OutlineText string         `json:"outline_text"`
	Review      map[string]any `json:"review"`
	RawReview   string         `json:"raw_review,omitempty"`
}

const subjectNameChinese = "语文"

var (
	allowedGenres    = map[string]struct{}{"记叙文": {}, "议论文": {}, "散文": {}, "应用文": {}, "说明文": {}}
	allowedTaskTypes = map[string]struct{}{"命题作文": {}, "材料作文": {}, "话题作文": {}, "任务驱动型作文": {}}
)

type TopicResult struct {
	TopicText string `json:"topic_text"`
	Label     string `json:"label"`
	Raw       string `json:"raw,omitempty"`
}

type ReviewResult struct {
	ID        uint64         `json:"id"`
	Review    map[string]any `json:"review"`
	RawReview string         `json:"raw_review,omitempty"`
}

type Service struct {
	cfg   config.Config
	db    *sqlx.DB
	aiLog *ailog.Writer
}

func New(cfg config.Config, db *sqlx.DB) *Service {
	return &Service{cfg: cfg, db: db, aiLog: ailog.NewWriter(db)}
}

type activeModel struct {
	ID   uint64
	Name string
	URL  string
}

func (s *Service) resolveAdapter(ctx context.Context) (studentpaper.AnalysisAdapter, *activeModel) {
	if !strings.EqualFold(s.cfg.AnalysisAdapter, "http") {
		return studentpaper.MockAnalysisAdapter{}, nil
	}
	if s.db != nil {
		var modelID uint64
		var name, url, chatModel, appSecret string
		err := s.db.QueryRowContext(ctx, `
SELECT id, name, url, model, app_secret
FROM ai_provider_model
WHERE status = 1 AND is_deleted = 0
ORDER BY id DESC
LIMIT 1
`).Scan(&modelID, &name, &url, &chatModel, &appSecret)
		if err == nil {
			url = strings.TrimSpace(url)
			if url != "" {
				return studentpaper.NewHTTPAnalysisAdapter(url, s.cfg.AIRequestTimeout, appSecret, chatModel), &activeModel{
					ID:   modelID,
					Name: strings.TrimSpace(name),
					URL:  url,
				}
			}
		}
	}
	if ep := strings.TrimSpace(s.cfg.AIEndpoint); ep != "" {
		return studentpaper.NewHTTPAnalysisAdapter(ep, s.cfg.AIRequestTimeout, "", ""), nil
	}
	return studentpaper.MockAnalysisAdapter{}, nil
}

func (s *Service) promptContent(ctx context.Context, key string, fallback string) string {
	if s.db == nil {
		return fallback
	}
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c, err := prompttemplate.GetActiveContent(ctx2, s.db, key)
	if err == nil && strings.TrimSpace(c) != "" {
		return c
	}
	return fallback
}

func (s *Service) writeAILog(ctx context.Context, meta *activeModel, studentID uint64, action string,
	in studentpaper.AnalyzeInput, out studentpaper.AnalyzeOutput, tr studentpaper.AnalyzeTrace) {

	if s.aiLog == nil || s.db == nil {
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
	reqM, _ := json.Marshal(map[string]any{
		"subject":  subjectNameChinese,
		"file_name": in.FileName,
	})
	respM, _ := json.Marshal(map[string]any{
		"summary_len": len(out.Summary),
	})
	var httpSt *int
	if tr.HTTPStatus != 0 {
		v := tr.HTTPStatus
		httpSt = &v
	}
	lat := tr.LatencyMS
	latPtr := &lat
	stuPtr := &studentID
	rt := "student"
	rid := studentID
	s.aiLog.Write(ctx, ailog.InsertRow{
		ProviderModelID: aid,
		ModelNameSnap:    nameSnap,
		Action:           action,
		AdapterKind:      tr.AdapterKind,
		ResultStatus:     tr.ResultStatus,
		HTTPStatus:       httpSt,
		LatencyMS:        latPtr,
		ErrorPhase:       tr.ErrorPhase,
		ErrorMessage:     tr.ErrorMessage,
		EndpointHost:     tr.EndpointHost,
		ChatModel:        tr.ChatModel,
		FallbackToMock:   tr.FallbackToMock,
		SysUserID:        stuPtr,
		RefTable:         &rt,
		RefID:            &rid,
		RequestMetaJSON:  reqM,
		ResponseMetaJSON: respM,
		RequestBody:      tr.RequestBody,
		ResponseBody:     tr.ResponseBody,
	})
}

func mockTopic(genre, taskType string) TopicResult {
	t := "有人说，真正的成长始于直面差异。请结合你的生活体验，写一篇议论文，谈谈你如何看待个体与集体的关系。"
	return TopicResult{
		TopicText: t,
		Label:     genre + " · " + taskType,
		Raw:       t + " | " + genre + " · " + taskType,
	}
}

func mockOCRTopic() TopicResult {
	return TopicResult{
		TopicText: "阅读下面的材料，根据要求写作。生活中，人们常用「底线」形容不可逾越的边界……",
		Label:     "自定义",
	}
}

func mockReview() string {
	return "提纲能回应题目核心概念，论点方向清楚。|匹配度4星/结构3星/素材4星|详细建议：1.分论点建议再分层，避免并列过平；2.素材可换更具时代感的例证；3.开头可增设情境引入增强代入感。"
}

// GenerateTopic calls AI with genre + task type.
func (s *Service) GenerateTopic(ctx context.Context, studentID uint64, genre, taskType string) (TopicResult, error) {
	genre = strings.TrimSpace(genre)
	taskType = strings.TrimSpace(taskType)
	if _, ok := allowedGenres[genre]; !ok {
		return TopicResult{}, ErrInvalidInput
	}
	if _, ok := allowedTaskTypes[taskType]; !ok {
		return TopicResult{}, ErrInvalidInput
	}

	adapter, meta := s.resolveAdapter(ctx)
	if _, ok := adapter.(studentpaper.MockAnalysisAdapter); ok {
		return mockTopic(genre, taskType), nil
	}

	tpl := s.promptContent(ctx, prompttemplate.KeyEssayOutlineGenerateTopic, prompttemplate.DefaultEssayOutlineGenerateTopic())
	prompt := prompttemplate.Expand(tpl, map[string]string{
		"genre":     genre,
		"task_type": taskType,
	})
	in := studentpaper.AnalyzeInput{
		Subject:        subjectNameChinese,
		Stage:          "",
		FileName:       "essay_topic_gen",
		VisionImages:   nil,
		ChatUserPrompt: prompt,
	}
	res := adapter.Analyze(in)
	out := res.Out
	if meta != nil {
		out.ModelSnapshot = map[string]any{"name": meta.Name, "url": meta.URL}
	}
	raw := strings.TrimSpace(out.Summary)
	if raw == "" && out.RawContent != "" {
		raw = strings.TrimSpace(out.RawContent)
	}
	s.writeAILog(ctx, meta, studentID, "essay_outline_generate_topic", in, out, res.Trace)

	topic, label := ParseTopicFromAI(raw, genre, taskType)
	if topic == "" {
		topic = raw
	}
	if label == "" {
		label = genre + " · " + taskType
	}
	return TopicResult{TopicText: topic, Label: label, Raw: raw}, nil
}

// OCRTopic extracts topic text from an image using vision model.
func (s *Service) OCRTopic(ctx context.Context, studentID uint64, raw []byte, filename, contentType string) (TopicResult, error) {
	if len(raw) == 0 {
		return TopicResult{}, ErrInvalidInput
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" || ct == "application/octet-stream" {
		ext := ""
		if i := strings.LastIndex(filename, "."); i >= 0 {
			ext = strings.ToLower(filename[i:])
		}
		if t := mime.TypeByExtension(ext); t != "" {
			ct = t
		}
	}
	if !strings.HasPrefix(ct, "image/") {
		return TopicResult{}, ErrInvalidInput
	}

	adapter, meta := s.resolveAdapter(ctx)
	if _, ok := adapter.(studentpaper.MockAnalysisAdapter); ok {
		return mockOCRTopic(), nil
	}

	tpl := s.promptContent(ctx, prompttemplate.KeyEssayOutlineOCRTopic, prompttemplate.DefaultEssayOutlineOCRTopic())
	in := studentpaper.AnalyzeInput{
		Subject:      subjectNameChinese,
		Stage:        "",
		FileName:     strings.TrimSpace(filename),
		VisionImages: []studentpaper.VisionImage{{MIME: ct, Data: raw}},
	}
	in.ChatUserPrompt = tpl
	res := adapter.Analyze(in)
	out := res.Out
	rawText := strings.TrimSpace(out.Summary)
	if rawText == "" && out.RawContent != "" {
		rawText = strings.TrimSpace(out.RawContent)
	}
	s.writeAILog(ctx, meta, studentID, "essay_outline_ocr_topic", in, out, res.Trace)

	return TopicResult{TopicText: rawText, Label: "自定义", Raw: rawText}, nil
}

// SubmitReview runs review AI and inserts essay_outline_practice.
func (s *Service) SubmitReview(ctx context.Context, studentID uint64, topicText, topicLabel, topicSource, genre, taskType, outlineText string) (ReviewResult, error) {
	if s.db == nil {
		return ReviewResult{}, ErrDatabaseRequired
	}
	topicText = strings.TrimSpace(topicText)
	topicLabel = strings.TrimSpace(topicLabel)
	outlineText = strings.TrimSpace(outlineText)
	if topicText == "" || outlineText == "" {
		return ReviewResult{}, ErrInvalidInput
	}
	ts := strings.TrimSpace(topicSource)
	if ts != "ai_category" && ts != "custom_text" && ts != "ocr_image" {
		return ReviewResult{}, ErrInvalidInput
	}

	genre = strings.TrimSpace(genre)
	taskType = strings.TrimSpace(taskType)
	if ts == "ai_category" {
		if _, ok := allowedGenres[genre]; !ok {
			return ReviewResult{}, ErrInvalidInput
		}
		if _, ok := allowedTaskTypes[taskType]; !ok {
			return ReviewResult{}, ErrInvalidInput
		}
	} else {
		genre, taskType = "", ""
	}

	adapter, meta := s.resolveAdapter(ctx)
	var reviewRaw string
	if _, ok := adapter.(studentpaper.MockAnalysisAdapter); ok {
		reviewRaw = mockReview()
	} else {
		tpl := s.promptContent(ctx, prompttemplate.KeyEssayOutlineReview, prompttemplate.DefaultEssayOutlineReview())
		prompt := prompttemplate.Expand(tpl, map[string]string{
			"topic_text":    topicText,
			"outline_text":  outlineText,
		})
		in := studentpaper.AnalyzeInput{
			Subject:        subjectNameChinese,
			Stage:          "",
			FileName:       "essay_outline_review",
			ChatUserPrompt: prompt,
		}
		res := adapter.Analyze(in)
		out := res.Out
		reviewRaw = strings.TrimSpace(out.Summary)
		if reviewRaw == "" {
			reviewRaw = strings.TrimSpace(out.RawContent)
		}
		s.writeAILog(ctx, meta, studentID, "essay_outline_review", in, out, res.Trace)
	}

	reviewObj := BuildReviewJSON(reviewRaw)
	reviewBytes, _ := json.Marshal(reviewObj)

	var subjectID sql.NullInt64
	_ = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT id FROM k12_subject WHERE name = ? AND status = 1 AND is_deleted = 0 LIMIT 1
`), subjectNameChinese).Scan(&subjectID)

	now := time.Now()
	var subArg any
	if subjectID.Valid {
		subArg = subjectID.Int64
	} else {
		subArg = nil
	}

	var newID uint64
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
INSERT INTO student_essay_outline_practice (
  sys_user_id, k12_subject_id, topic_text, topic_label, topic_source, genre, task_type,
  outline_text, review_json, raw_review_response,
  created_at, created_by, updated_at, updated_by, is_deleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
RETURNING id
`), studentID, subArg, topicText, topicLabel, ts, nullIfEmpty(genre), nullIfEmpty(taskType), outlineText,
		string(reviewBytes), reviewRaw, now, studentID, now, studentID).Scan(&newID)
	if err != nil {
		return ReviewResult{}, err
	}
	return ReviewResult{
		ID:        newID,
		Review:    reviewObj,
		RawReview: reviewRaw,
	}, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func previewRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…"
}

// ListPractices returns recent non-deleted rows for the student, newest first.
func (s *Service) ListPractices(ctx context.Context, studentID uint64, limit int) ([]PracticeSummary, error) {
	if s.db == nil {
		return nil, ErrDatabaseRequired
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, dbutil.Rebind(`
SELECT id, created_at, topic_label, topic_source, topic_text
FROM student_essay_outline_practice
WHERE sys_user_id = ? AND is_deleted = 0
ORDER BY created_at DESC, id DESC
LIMIT ?
`), studentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PracticeSummary
	for rows.Next() {
		var p PracticeSummary
		var topicFull string
		if err := rows.Scan(&p.ID, &p.CreatedAt, &p.TopicLabel, &p.TopicSource, &topicFull); err != nil {
			return nil, err
		}
		p.TopicPreview = previewRunes(topicFull, 100)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// PracticeVisibilityCounts returns (all rows for student, rows with is_deleted=0). Helps explain empty lists.
func (s *Service) PracticeVisibilityCounts(ctx context.Context, studentID uint64) (total, active int, err error) {
	if s.db == nil {
		return 0, 0, ErrDatabaseRequired
	}
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT COUNT(*) FROM student_essay_outline_practice WHERE sys_user_id = ?`), studentID).Scan(&total)
	if err != nil {
		return 0, 0, err
	}
	err = s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT COUNT(*) FROM student_essay_outline_practice WHERE sys_user_id = ? AND is_deleted = 0`), studentID).Scan(&active)
	if err != nil {
		return 0, 0, err
	}
	return total, active, nil
}

// GetPractice returns one practice if it belongs to the student.
func (s *Service) GetPractice(ctx context.Context, studentID, practiceID uint64) (PracticeDetail, error) {
	if s.db == nil {
		return PracticeDetail{}, ErrDatabaseRequired
	}
	var p PracticeDetail
	var genreN, taskN sql.NullString
	var reviewBlob []byte
	var rawN sql.NullString
	err := s.db.QueryRowContext(ctx, dbutil.Rebind(`
SELECT id, created_at, topic_text, topic_label, topic_source, genre, task_type, outline_text, review_json, raw_review_response
FROM student_essay_outline_practice
WHERE id = ? AND sys_user_id = ? AND is_deleted = 0
LIMIT 1
`), practiceID, studentID).Scan(
		&p.ID, &p.CreatedAt, &p.TopicText, &p.TopicLabel, &p.TopicSource,
		&genreN, &taskN, &p.OutlineText, &reviewBlob, &rawN,
	)
	if err == sql.ErrNoRows {
		return PracticeDetail{}, ErrPracticeNotFound
	}
	if err != nil {
		return PracticeDetail{}, err
	}
	if genreN.Valid {
		p.Genre = genreN.String
	}
	if taskN.Valid {
		p.TaskType = taskN.String
	}
	if rawN.Valid {
		p.RawReview = rawN.String
	}
	p.Review = map[string]any{}
	if len(reviewBlob) > 0 {
		_ = json.Unmarshal(reviewBlob, &p.Review)
	}
	if p.Review == nil {
		p.Review = map[string]any{}
	}
	return p, nil
}
