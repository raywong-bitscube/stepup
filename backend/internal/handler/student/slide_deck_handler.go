package student

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/studentslidedeck"
)

type SlideDeckHandler struct {
	svc *studentslidedeck.Service
}

func NewSlideDeckHandler(svc *studentslidedeck.Service) *SlideDeckHandler {
	return &SlideDeckHandler{svc: svc}
}

func (h *SlideDeckHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.PathValue("sectionId"))
	sid, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || sid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	d, err := h.svc.GetActive(r.Context(), sid)
	switch {
	case errors.Is(err, studentslidedeck.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, studentslidedeck.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             d.ID,
			"section_id":     d.SectionID,
			"title":          d.Title,
			"schema_version": d.SchemaVersion,
			"updated_at":     d.UpdatedAt,
			"content":        json.RawMessage(d.Content),
		})
	}
}
