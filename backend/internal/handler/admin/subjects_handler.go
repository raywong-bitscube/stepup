package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminsubjects"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
)

type SubjectsHandler struct {
	service *adminsubjects.Service
	audit   *auditlog.Writer
}

func NewSubjectsHandler(service *adminsubjects.Service, audit *auditlog.Writer) *SubjectsHandler {
	return &SubjectsHandler{service: service, audit: audit}
}

func (h *SubjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if errors.Is(err, adminsubjects.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	type row struct {
		ID             uint64      `json:"id"`
		Name           string      `json:"name"`
		Description    *string     `json:"description"`
		Status         int         `json:"status"`
		CreatedAt      RFC3339Time `json:"created_at"`
		TextbookCount  int         `json:"textbook_count"`
	}
	out := make([]row, 0, len(items))
	for _, s := range items {
		out = append(out, row{
			ID:             s.ID,
			Name:           s.Name,
			Description:    s.Description,
			Status:         s.Status,
			CreatedAt:      RFC3339Time(s.CreatedAt),
			TextbookCount:  s.TextbookCount,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

type createSubjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *SubjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	nid, err := h.service.Create(r.Context(), adminsubjects.CreateInput{
		Name:        req.Name,
		Description: req.Description,
	})
	switch {
	case errors.Is(err, adminsubjects.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminsubjects.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminsubjects.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				snap, _ := json.Marshal(map[string]any{"name": req.Name})
				pid := nid
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     "create",
					EntityType: "subject",
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

type patchSubjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *int    `json:"status"`
}

func (h *SubjectsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("subjectId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.service.Patch(r.Context(), id, adminsubjects.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
	switch {
	case errors.Is(err, adminsubjects.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminsubjects.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminsubjects.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, adminsubjects.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				snap, _ := json.Marshal(map[string]any{"has_name": req.Name != nil, "has_description": req.Description != nil, "has_status": req.Status != nil})
				sid := id
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     "update",
					EntityType: "subject",
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
