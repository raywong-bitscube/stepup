package adminslidegen

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminslidedecks"
	"github.com/raywong-bitscube/stepup/backend/internal/service/ailog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
)

var (
	ErrNoDatabase   = errors.New("database not configured")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrAIFailed     = errors.New("ai slide json failed")
)

type Service struct {
	cfg   config.Config
	db    *sql.DB
	aiLog *ailog.Writer
	decks *adminslidedecks.Service
}

func New(cfg config.Config, db *sql.DB) *Service {
	return &Service{
		cfg:   cfg,
		db:    db,
		aiLog: ailog.NewWriter(db),
		decks: adminslidedecks.New(db),
	}
}

type activeModel struct {
	ID     uint64
	Name   string
	URL    string
	Secret string
	Model  string
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
FROM ai_model
WHERE status = 1 AND is_deleted = 0
ORDER BY id DESC
LIMIT 1
`).Scan(&modelID, &name, &url, &chatModel, &appSecret)
		if err == nil {
			url = strings.TrimSpace(url)
			if url != "" {
				return studentpaper.NewHTTPAnalysisAdapter(url, s.cfg.AIRequestTimeout, appSecret, chatModel), &activeModel{
					ID: modelID, Name: strings.TrimSpace(name), URL: url,
					Secret: appSecret, Model: chatModel,
				}
			}
		}
	}
	if ep := strings.TrimSpace(s.cfg.AIEndpoint); ep != "" {
		return studentpaper.NewHTTPAnalysisAdapter(ep, s.cfg.AIRequestTimeout, "", ""), nil
	}
	return studentpaper.MockAnalysisAdapter{}, nil
}

// SectionContext is textbook path for prompts.
type SectionContext struct {
	SectionID       uint64
	SecNum          int
	SectionTitle    string
	SectionFull     string
	ChapterNum      int
	ChapterTitle    string
	TextbookName    string
	TextbookVersion string
	Subject         string
}

func (s *Service) loadSectionContext(ctx context.Context, sectionID uint64) (*SectionContext, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var c SectionContext
	var full sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT s.id, s.number, s.title, s.full_title, ch.number, ch.title, t.name, t.version, t.subject
FROM section s
JOIN chapter ch ON ch.id = s.chapter_id AND ch.is_deleted = 0
JOIN textbook t ON t.id = ch.textbook_id AND t.is_deleted = 0
WHERE s.id = ? AND s.is_deleted = 0`, sectionID).Scan(
		&c.SectionID, &c.SecNum, &c.SectionTitle, &full, &c.ChapterNum, &c.ChapterTitle,
		&c.TextbookName, &c.TextbookVersion, &c.Subject,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if full.Valid {
		c.SectionFull = full.String
	}
	return &c, nil
}

// DefaultPrompt builds a template the admin UI can pre-fill.
func DefaultPrompt(c *SectionContext) string {
	if c == nil {
		return ""
	}
	ft := strings.TrimSpace(c.SectionFull)
	if ft == "" {
		ft = c.SectionTitle
	}
	return fmt.Sprintf(`你是 StepUp 课件结构生成助手。根据以下教材节信息，输出**仅一个**合法 JSON 对象（不要 Markdown 代码围栏），符合 slide schemaVersion 1：
- 顶层：schemaVersion:1，meta:{ "title": string, "theme":"dark-physics" }，slides: 数组
- 每页：id, layoutTemplate（选用 cover-image | title-body | formula-focus | split-left-right | split-top-bottom | quiz-center | bullet-steps | two-column-text），elements: 扁平数组；每项含 type(text|latex|image|question)、role、step（从 1 起的整数）
- question：mode 为 single 或 multi；data:{ "text", "options":[{ "id","text" }] }

教材：《%s》 %s，学科 %s
章：第 %d 章 %s
节：第 %d 节 %s（%s）

请生成约 3～5 页、适合课堂讲解的幻灯片 JSON。`,
		c.TextbookName, c.TextbookVersion, c.Subject,
		c.ChapterNum, c.ChapterTitle,
		c.SecNum, c.SectionTitle, ft,
	)
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := strings.TrimPrefix(s, "```")
	rest = strings.TrimPrefix(strings.TrimSpace(rest), "json")
	rest = strings.TrimSpace(rest)
	if idx := strings.LastIndex(rest, "```"); idx >= 0 {
		rest = rest[:idx]
	}
	return strings.TrimSpace(rest)
}

func mockDeckJSON(c *SectionContext) json.RawMessage {
	t := "演示课件"
	if c != nil && strings.TrimSpace(c.SectionTitle) != "" {
		t = strings.TrimSpace(c.SectionTitle)
	}
	raw := fmt.Sprintf(`{"schemaVersion":1,"meta":{"title":%q,"theme":"dark-physics"},"slides":[{"id":"s1","layoutTemplate":"cover-image","elements":[{"type":"text","role":"title","content":%q,"step":1},{"type":"text","role":"subtitle","content":"（mock 环境占位，配置 ANALYSIS_ADAPTER=http 后由模型生成）","step":2}]}]}`, t, t)
	return json.RawMessage(raw)
}

func normalizeSlideJSON(raw string) (json.RawMessage, error) {
	s := stripCodeFence(raw)
	b := []byte(s)
	if err := adminslidedecks.ValidateSlideJSON(b); err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	out, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func deckTitleFromJSON(b json.RawMessage) string {
	var root map[string]interface{}
	if err := json.Unmarshal(b, &root); err != nil {
		return ""
	}
	meta, _ := root["meta"].(map[string]interface{})
	if meta == nil {
		return ""
	}
	t, _ := meta["title"].(string)
	return strings.TrimSpace(t)
}

func (s *Service) writeAILog(ctx context.Context, meta *activeModel, trace studentpaper.AnalyzeTrace,
	action string, sectionID uint64, reqMeta, respMeta map[string]any) {

	if s.aiLog == nil {
		return
	}
	var aid *uint64
	nameSnap := ""
	if meta != nil && meta.ID != 0 {
		v := meta.ID
		aid = &v
		nameSnap = meta.Name
	}
	reqM, _ := json.Marshal(reqMeta)
	respM, _ := json.Marshal(respMeta)
	var httpSt *int
	if trace.HTTPStatus != 0 {
		v := trace.HTTPStatus
		httpSt = &v
	}
	lat := trace.LatencyMS
	latPtr := &lat
	rt := "section"
	rid := sectionID
	s.aiLog.Write(ctx, ailog.InsertRow{
		AIModelID:        aid,
		ModelNameSnap:    nameSnap,
		Action:           action,
		AdapterKind:      trace.AdapterKind,
		ResultStatus:     trace.ResultStatus,
		HTTPStatus:       httpSt,
		LatencyMS:        latPtr,
		ErrorPhase:       trace.ErrorPhase,
		ErrorMessage:     trace.ErrorMessage,
		EndpointHost:     trace.EndpointHost,
		ChatModel:        trace.ChatModel,
		FallbackToMock:   trace.FallbackToMock,
		RefTable:         &rt,
		RefID:            &rid,
		RequestMetaJSON:  reqM,
		ResponseMetaJSON: respM,
		RequestBody:      trace.RequestBody,
		ResponseBody:     trace.ResponseBody,
	})
}

// Generate calls the active AI model with prompt, validates JSON, inserts slide_deck as draft, writes ai_call_log.
func (s *Service) Generate(ctx context.Context, sectionID uint64, adminID uint64, prompt string) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, ErrNoDatabase
	}
	prompt = strings.TrimSpace(prompt)
	if sectionID == 0 || adminID == 0 || prompt == "" {
		return 0, ErrInvalidInput
	}

	secCtx, err := s.loadSectionContext(ctx, sectionID)
	if err != nil {
		return 0, err
	}

	adapter, meta := s.resolveAdapter(ctx)
	res := adapter.Analyze(studentpaper.AnalyzeInput{ChatUserPrompt: prompt})

	raw := strings.TrimSpace(res.Out.RawContent)
	if res.Trace.AdapterKind == "mock_builtin" || res.Trace.ResultStatus == "mock_only" {
		b := mockDeckJSON(secCtx)
		raw = string(b)
	}

	reqMeta := map[string]any{
		"section_id":   sectionID,
		"prompt_chars": len([]rune(prompt)),
	}
	var deckID uint64
	var respMeta map[string]any

	content, normErr := normalizeSlideJSON(raw)
	if normErr != nil {
		respMeta = map[string]any{"error": "invalid_slide_json", "detail": normErr.Error()}
		if res.Trace.ResultStatus == "success" {
			res.Trace.ResultStatus = "parse_error"
		}
		res.Trace.ErrorPhase = "slide_json_validate"
		res.Trace.ErrorMessage = normErr.Error()
		s.writeAILog(ctx, meta, res.Trace, "slide_deck_generate_ai", sectionID, reqMeta, respMeta)
		return 0, fmt.Errorf("%w: %v", ErrAIFailed, normErr)
	}

	title := deckTitleFromJSON(content)
	if title == "" {
		title = secCtx.SectionTitle + " · AI草稿"
	}
	if len([]rune(title)) > 200 {
		title = string([]rune(title)[:200])
	}

	gp := prompt
	nid, cerr := s.decks.Create(ctx, sectionID, adminID, adminslidedecks.CreateInput{
		Title:            title,
		Content:          content,
		DeckStatus:       adminslidedecks.StatusDraft,
		SchemaVersion:    1,
		GenerationPrompt: &gp,
	})
	if cerr != nil {
		respMeta = map[string]any{"error": "db_create", "detail": cerr.Error()}
		s.writeAILog(ctx, meta, res.Trace, "slide_deck_generate_ai", sectionID, reqMeta, respMeta)
		return 0, cerr
	}
	deckID = nid
	respMeta = map[string]any{"deck_id": deckID, "deck_status": "draft"}
	s.writeAILog(ctx, meta, res.Trace, "slide_deck_generate_ai", sectionID, reqMeta, respMeta)
	return deckID, nil
}

// DefaultPromptForSection returns the template for GET handler.
func (s *Service) DefaultPromptForSection(ctx context.Context, sectionID uint64) (string, error) {
	c, err := s.loadSectionContext(ctx, sectionID)
	if err != nil {
		return "", err
	}
	return DefaultPrompt(c), nil
}
