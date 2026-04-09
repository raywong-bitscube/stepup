package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminslidedecks"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
)

type SlideDecksHandler struct {
	svc   *adminslidedecks.Service
	audit *auditlog.Writer
}

func NewSlideDecksHandler(svc *adminslidedecks.Service, audit *auditlog.Writer) *SlideDecksHandler {
	return &SlideDecksHandler{svc: svc, audit: audit}
}

func (h *SlideDecksHandler) ListBySection(w http.ResponseWriter, r *http.Request) {
	sid, ok := pathUint(r, "sectionId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	items, err := h.svc.ListSummaries(r.Context(), sid)
	switch {
	case errors.Is(err, adminslidedecks.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		type row struct {
			ID            uint64      `json:"id"`
			SectionID     uint64      `json:"section_id"`
			Title         string      `json:"title"`
			DeckStatus    string      `json:"deck_status"`
			SchemaVersion int         `json:"schema_version"`
			UpdatedAt     RFC3339Time `json:"updated_at"`
		}
		out := make([]row, 0, len(items))
		for _, it := range items {
			out = append(out, row{
				ID:            it.ID,
				SectionID:     it.SectionID,
				Title:         it.Title,
				DeckStatus:    it.DeckStatus,
				SchemaVersion: it.SchemaVersion,
				UpdatedAt:     RFC3339Time(it.UpdatedAt),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

func (h *SlideDecksHandler) Get(w http.ResponseWriter, r *http.Request) {
	did, ok := pathUint(r, "deckId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	f, err := h.svc.Get(r.Context(), did)
	switch {
	case errors.Is(err, adminslidedecks.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminslidedecks.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		payload := map[string]any{
			"id":             f.ID,
			"section_id":     f.SectionID,
			"title":          f.Title,
			"deck_status":    f.DeckStatus,
			"schema_version": f.SchemaVersion,
			"updated_at":     RFC3339Time(f.UpdatedAt),
			"content":        json.RawMessage(f.Content),
		}
		if f.GenerationPrompt != nil {
			payload["generation_prompt"] = *f.GenerationPrompt
		}
		_ = json.NewEncoder(w).Encode(payload)
	}
}

type createSlideDeckRequest struct {
	Title              string          `json:"title"`
	Content            json.RawMessage `json:"content"`
	DeckStatus         string          `json:"deck_status"`
	SchemaVersion      int             `json:"schema_version"`
	GenerationPrompt   *string         `json:"generation_prompt"`
}

func (h *SlideDecksHandler) Create(w http.ResponseWriter, r *http.Request) {
	sid, ok := pathUint(r, "sectionId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req createSlideDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	sess, ok2 := middleware.AdminSession(r.Context())
	if !ok2 || sess.AdminID == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	nid, err := h.svc.Create(r.Context(), sid, sess.AdminID, adminslidedecks.CreateInput{
		Title:              req.Title,
		Content:            req.Content,
		DeckStatus:         req.DeckStatus,
		SchemaVersion:      req.SchemaVersion,
		GenerationPrompt:   req.GenerationPrompt,
	})
	switch {
	case errors.Is(err, adminslidedecks.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminslidedecks.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminslidedecks.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			pid := nid
			h.audit.Write(r.Context(), auditlog.Event{
				UserID:     &sess.AdminID,
				UserType:   "admin",
				Action:     "create",
				EntityType: "slide_deck",
				EntityID:   &pid,
				Snapshot:   []byte(`{}`),
				IP:         r.RemoteAddr,
				CreatedBy:  sess.AdminID,
			})
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": nid, "status": "ok"})
	}
}

type patchSlideDeckRequest struct {
	Title              *string          `json:"title"`
	Content            *json.RawMessage `json:"content"`
	DeckStatus         *string          `json:"deck_status"`
	GenerationPrompt   *string          `json:"generation_prompt"`
}

func (h *SlideDecksHandler) Patch(w http.ResponseWriter, r *http.Request) {
	did, ok := pathUint(r, "deckId")
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchSlideDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	in := adminslidedecks.PatchInput{
		Title:              req.Title,
		Content:            req.Content,
		DeckStatus:         req.DeckStatus,
		GenerationPrompt:   req.GenerationPrompt,
	}
	sess, ok2 := middleware.AdminSession(r.Context())
	if !ok2 || sess.AdminID == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	err := h.svc.Patch(r.Context(), did, sess.AdminID, in)
	switch {
	case errors.Is(err, adminslidedecks.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminslidedecks.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminslidedecks.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			sid := did
			h.audit.Write(r.Context(), auditlog.Event{
				UserID:     &sess.AdminID,
				UserType:   "admin",
				Action:     "update",
				EntityType: "slide_deck",
				EntityID:   &sid,
				Snapshot:   []byte(`{}`),
				IP:         r.RemoteAddr,
				CreatedBy:  sess.AdminID,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func pathUint(r *http.Request, name string) (uint64, bool) {
	raw := strings.TrimSpace(r.PathValue(name))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	return n, err == nil && n > 0
}
