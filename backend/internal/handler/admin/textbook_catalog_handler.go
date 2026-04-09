package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/admintextbookcatalog"
)

type TextbookCatalogHandler struct {
	svc *admintextbookcatalog.Service
}

func NewTextbookCatalogHandler(svc *admintextbookcatalog.Service) *TextbookCatalogHandler {
	return &TextbookCatalogHandler{svc: svc}
}

func (h *TextbookCatalogHandler) ListTextbooksBySubject(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("subjectId"))
	subjectID, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || subjectID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	items, err := h.svc.ListTextbooksBySubject(r.Context(), subjectID)
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		type row struct {
			ID        uint64      `json:"id"`
			Name      string      `json:"name"`
			Version   string      `json:"version"`
			Subject   string      `json:"subject"`
			Category  string      `json:"category"`
			Remarks   *string     `json:"remarks"`
			Status    int         `json:"status"`
			UpdatedAt RFC3339Time `json:"updated_at"`
		}
		out := make([]row, 0, len(items))
		for _, t := range items {
			out = append(out, row{
				ID:        t.ID,
				Name:      t.Name,
				Version:   t.Version,
				Subject:   t.Subject,
				Category:  t.Category,
				Remarks:   t.Remarks,
				Status:    t.Status,
				UpdatedAt: RFC3339Time(t.UpdatedAt),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

type patchTextbookBody struct {
	Name    *string `json:"name"`
	Version *string `json:"version"`
	Subject *string `json:"subject"`
	Remarks *string `json:"remarks"`
	Status  *int    `json:"status"`
}

func (h *TextbookCatalogHandler) PatchTextbook(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("textbookId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var body patchTextbookBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.svc.PatchTextbook(r.Context(), id, admintextbookcatalog.TextbookPatch{
		Name:    body.Name,
		Version: body.Version,
		Subject: body.Subject,
		Remarks: body.Remarks,
		Status:  body.Status,
	})
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, admintextbookcatalog.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func (h *TextbookCatalogHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("textbookId"))
	tid, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || tid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	items, err := h.svc.ListChapters(r.Context(), tid)
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		type row struct {
			ID         uint64      `json:"id"`
			TextbookID uint64      `json:"textbook_id"`
			Number     uint32      `json:"number"`
			Title      string      `json:"title"`
			FullTitle  *string     `json:"full_title"`
			Status     int         `json:"status"`
			UpdatedAt  RFC3339Time `json:"updated_at"`
		}
		out := make([]row, 0, len(items))
		for _, c := range items {
			out = append(out, row{
				ID:         c.ID,
				TextbookID: c.TextbookID,
				Number:     c.Number,
				Title:      c.Title,
				FullTitle:  c.FullTitle,
				Status:     c.Status,
				UpdatedAt:  RFC3339Time(c.UpdatedAt),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

type patchChapterBody struct {
	Number    *uint32 `json:"number"`
	Title     *string `json:"title"`
	FullTitle *string `json:"full_title"`
	Status    *int    `json:"status"`
}

func (h *TextbookCatalogHandler) PatchChapter(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("chapterId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var body patchChapterBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.svc.PatchChapter(r.Context(), id, admintextbookcatalog.ChapterPatch{
		Number:    body.Number,
		Title:     body.Title,
		FullTitle: body.FullTitle,
		Status:    body.Status,
	})
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func (h *TextbookCatalogHandler) ListSections(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("chapterId"))
	cid, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || cid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	items, err := h.svc.ListSections(r.Context(), cid)
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		type row struct {
			ID        uint64      `json:"id"`
			ChapterID uint64      `json:"chapter_id"`
			Number    uint32      `json:"number"`
			Title     string      `json:"title"`
			FullTitle *string     `json:"full_title"`
			Status    int         `json:"status"`
			UpdatedAt RFC3339Time `json:"updated_at"`
		}
		out := make([]row, 0, len(items))
		for _, s := range items {
			out = append(out, row{
				ID:        s.ID,
				ChapterID: s.ChapterID,
				Number:    s.Number,
				Title:     s.Title,
				FullTitle: s.FullTitle,
				Status:    s.Status,
				UpdatedAt: RFC3339Time(s.UpdatedAt),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
	}
}

type patchSectionBody struct {
	Number    *uint32 `json:"number"`
	Title     *string `json:"title"`
	FullTitle *string `json:"full_title"`
	Status    *int    `json:"status"`
}

func (h *TextbookCatalogHandler) PatchSection(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("sectionId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var body patchSectionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.svc.PatchSection(r.Context(), id, admintextbookcatalog.SectionPatch{
		Number:    body.Number,
		Title:     body.Title,
		FullTitle: body.FullTitle,
		Status:    body.Status,
	})
	switch {
	case errors.Is(err, admintextbookcatalog.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, admintextbookcatalog.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, admintextbookcatalog.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}
