package student

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentessayoutline"
)

type EssayOutlineHandler struct {
	svc *studentessayoutline.Service
}

func NewEssayOutlineHandler(svc *studentessayoutline.Service) *EssayOutlineHandler {
	return &EssayOutlineHandler{svc: svc}
}

// GenerateTopic POST /api/v1/student/essay-outline/generate-topic
func (h *EssayOutlineHandler) GenerateTopic(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "SERVICE_UNAVAILABLE"})
		return
	}
	var body struct {
		Genre    string `json:"genre"`
		TaskType string `json:"task_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	sid := middleware.StudentDBID(r.Context())
	if sid == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	out, err := h.svc.GenerateTopic(r.Context(), sid, body.Genre, body.TaskType)
	if err != nil {
		if errors.Is(err, studentessayoutline.ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "GENERATE_FAILED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"topic_text": out.TopicText,
		"label":      out.Label,
		"raw":        out.Raw,
	})
}

// OCRTopic POST multipart file
func (h *EssayOutlineHandler) OCRTopic(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "SERVICE_UNAVAILABLE"})
		return
	}
	const maxBytes = 12 << 20
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_MULTIPART"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "FILE_REQUIRED"})
		return
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "READ_FAILED"})
		return
	}
	if len(raw) > maxBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"code": "FILE_TOO_LARGE"})
		return
	}
	sid := middleware.StudentDBID(r.Context())
	if sid == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	fn := header.Filename
	ct := header.Header.Get("Content-Type")
	out, err := h.svc.OCRTopic(r.Context(), sid, raw, fn, ct)
	if err != nil {
		if errors.Is(err, studentessayoutline.ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_IMAGE"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "OCR_FAILED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"topic_text": out.TopicText,
		"label":      out.Label,
	})
}

// Review POST JSON
func (h *EssayOutlineHandler) Review(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "SERVICE_UNAVAILABLE"})
		return
	}
	var body struct {
		TopicText   string `json:"topic_text"`
		TopicLabel  string `json:"topic_label"`
		TopicSource string `json:"topic_source"`
		Genre       string `json:"genre"`
		TaskType    string `json:"task_type"`
		OutlineText string `json:"outline_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	sid := middleware.StudentDBID(r.Context())
	if sid == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	out, err := h.svc.SubmitReview(r.Context(), sid,
		body.TopicText, body.TopicLabel, body.TopicSource, body.Genre, body.TaskType, body.OutlineText)
	if err != nil {
		if errors.Is(err, studentessayoutline.ErrDatabaseRequired) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
			return
		}
		if errors.Is(err, studentessayoutline.ErrInvalidInput) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "REVIEW_FAILED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         out.ID,
		"review":     out.Review,
		"raw_review": out.RawReview,
	})
}

// ListPractices GET /api/v1/student/essay-outline/practices?limit=
func (h *EssayOutlineHandler) ListPractices(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "SERVICE_UNAVAILABLE"})
		return
	}
	sid := middleware.StudentDBID(r.Context())
	if sid == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := h.svc.ListPractices(r.Context(), sid, limit)
	if err != nil {
		if errors.Is(err, studentessayoutline.ErrDatabaseRequired) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "LIST_FAILED"})
		return
	}
	if items == nil {
		items = []studentessayoutline.PracticeSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// GetPractice GET /api/v1/student/essay-outline/practices/{practiceId}
func (h *EssayOutlineHandler) GetPractice(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "SERVICE_UNAVAILABLE"})
		return
	}
	sid := middleware.StudentDBID(r.Context())
	if sid == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	idStr := strings.TrimSpace(r.PathValue("practiceId"))
	pid, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || pid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_ID"})
		return
	}
	out, err := h.svc.GetPractice(r.Context(), sid, pid)
	if err != nil {
		if errors.Is(err, studentessayoutline.ErrDatabaseRequired) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
			return
		}
		if errors.Is(err, studentessayoutline.ErrPracticeNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "GET_FAILED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"practice": out,
	})
}
