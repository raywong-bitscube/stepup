package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminslidegen"
)

type SlideDeckGenHandler struct {
	gen *adminslidegen.Service
}

func NewSlideDeckGenHandler(gen *adminslidegen.Service) *SlideDeckGenHandler {
	return &SlideDeckGenHandler{gen: gen}
}

func (h *SlideDeckGenHandler) DefaultPrompt(w http.ResponseWriter, r *http.Request) {
	sid, ok := pathUint(r, "sectionId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	p, err := h.gen.DefaultPromptForSection(r.Context(), sid)
	switch {
	case errors.Is(err, adminslidegen.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminslidegen.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"prompt": p})
	}
}

type generateSlideDeckRequest struct {
	Prompt string `json:"prompt"`
}

func (h *SlideDeckGenHandler) GenerateAI(w http.ResponseWriter, r *http.Request) {
	sid, ok := pathUint(r, "sectionId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req generateSlideDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	sess, ok2 := middleware.AdminSession(r.Context())
	if !ok2 || sess.AdminID == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	nid, err := h.gen.Generate(r.Context(), sid, sess.AdminID, req.Prompt)
	switch {
	case errors.Is(err, adminslidegen.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminslidegen.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, adminslidegen.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminslidegen.ErrAIFailed):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "AI_SLIDE_JSON_INVALID", "message": err.Error()})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"id": nid, "status": "ok"})
	}
}
