package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminpapers"
)

type StudentPapersHandler struct {
	service *adminpapers.Service
}

func NewStudentPapersHandler(service *adminpapers.Service) *StudentPapersHandler {
	return &StudentPapersHandler{service: service}
}

func (h *StudentPapersHandler) List(w http.ResponseWriter, r *http.Request) {
	studentID, ok := parseStudentID(r.PathValue("studentId"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	items, err := h.service.List(r.Context(), studentID)
	switch {
	case errors.Is(err, adminpapers.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminpapers.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func (h *StudentPapersHandler) Analysis(w http.ResponseWriter, r *http.Request) {
	studentID, ok := parseStudentID(r.PathValue("studentId"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	paperID := strings.TrimSpace(r.PathValue("paperId"))
	a, err := h.service.GetAnalysis(r.Context(), studentID, paperID)
	switch {
	case errors.Is(err, adminpapers.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminpapers.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"analysis": a})
	}
}

func (h *StudentPapersHandler) Plan(w http.ResponseWriter, r *http.Request) {
	studentID, ok := parseStudentID(r.PathValue("studentId"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	paperID := strings.TrimSpace(r.PathValue("paperId"))
	plan, err := h.service.GetPlan(r.Context(), studentID, paperID)
	switch {
	case errors.Is(err, adminpapers.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminpapers.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, plan)
	}
}

func parseStudentID(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return n, true
}
