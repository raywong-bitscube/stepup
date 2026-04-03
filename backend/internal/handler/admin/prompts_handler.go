package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminprompts"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
)

type PromptsHandler struct {
	service *adminprompts.Service
	audit   *auditlog.Writer
}

func NewPromptsHandler(service *adminprompts.Service, audit *auditlog.Writer) *PromptsHandler {
	return &PromptsHandler{service: service, audit: audit}
}

func (h *PromptsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if errors.Is(err, adminprompts.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	type row struct {
		ID          uint64      `json:"id"`
		Key         string      `json:"key"`
		Description *string     `json:"description"`
		Content     string      `json:"content"`
		Status      int         `json:"status"`
		CreatedAt   RFC3339Time `json:"created_at"`
	}
	out := make([]row, 0, len(items))
	for _, p := range items {
		out = append(out, row{
			ID:          p.ID,
			Key:         p.Key,
			Description: p.Description,
			Content:     p.Content,
			Status:      p.Status,
			CreatedAt:   RFC3339Time(p.CreatedAt),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

type createPromptRequest struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Status      *int   `json:"status"`
}

func (h *PromptsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	nid, err := h.service.Create(r.Context(), adminprompts.CreateInput{
		Key:         req.Key,
		Description: req.Description,
		Content:     req.Content,
		Status:      req.Status,
	})
	switch {
	case errors.Is(err, adminprompts.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminprompts.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminprompts.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				snap, _ := json.Marshal(map[string]any{"key": strings.TrimSpace(req.Key)})
				pid := nid
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     "create",
					EntityType: "prompt_template",
					EntityID:   &pid,
					Snapshot:   snap,
					IP:         r.RemoteAddr,
					CreatedBy:  adm,
				})
			}
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
	}
}

type patchPromptRequest struct {
	Key         *string `json:"key"`
	Description *string `json:"description"`
	Content     *string `json:"content"`
	Status      *int    `json:"status"`
}

func (h *PromptsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("promptId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.service.Patch(r.Context(), id, adminprompts.UpdateInput{
		Key:         req.Key,
		Description: req.Description,
		Content:     req.Content,
		Status:      req.Status,
	})
	switch {
	case errors.Is(err, adminprompts.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminprompts.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminprompts.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, adminprompts.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				snap, _ := json.Marshal(map[string]any{
					"has_key": req.Key != nil, "has_description": req.Description != nil,
					"has_content": req.Content != nil, "has_status": req.Status != nil,
				})
				sid := id
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     "update",
					EntityType: "prompt_template",
					EntityID:   &sid,
					Snapshot:   snap,
					IP:         r.RemoteAddr,
					CreatedBy:  adm,
				})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}
