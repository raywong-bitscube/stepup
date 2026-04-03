package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminaimodels"
)

type AIModelsHandler struct {
	service *adminaimodels.Service
}

func NewAIModelsHandler(service *adminaimodels.Service) *AIModelsHandler {
	return &AIModelsHandler{service: service}
}

func (h *AIModelsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if errors.Is(err, adminaimodels.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	type row struct {
		ID        uint64      `json:"id"`
		Name      string      `json:"name"`
		URL       string      `json:"url"`
		AppKey    string      `json:"app_key"`
		Status    int         `json:"status"`
		CreatedAt RFC3339Time `json:"created_at"`
	}
	out := make([]row, 0, len(items))
	for _, m := range items {
		out = append(out, row{
			ID:        m.ID,
			Name:      m.Name,
			URL:       m.URL,
			AppKey:    m.AppKey,
			Status:    m.Status,
			CreatedAt: RFC3339Time(m.CreatedAt),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

type createAIModelRequest struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
	Status    *int   `json:"status"`
}

func (h *AIModelsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAIModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err := h.service.Create(r.Context(), adminaimodels.CreateInput{
		Name:      req.Name,
		URL:       req.URL,
		AppKey:    req.AppKey,
		AppSecret: req.AppSecret,
		Status:    req.Status,
	})
	switch {
	case errors.Is(err, adminaimodels.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminaimodels.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
	}
}

type patchAIModelRequest struct {
	Name      *string `json:"name"`
	URL       *string `json:"url"`
	AppKey    *string `json:"app_key"`
	AppSecret *string `json:"app_secret"`
	Status    *int    `json:"status"`
}

func (h *AIModelsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	idRaw := strings.TrimSpace(r.PathValue("modelId"))
	id, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchAIModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.service.Patch(r.Context(), id, adminaimodels.UpdateInput{
		Name:      req.Name,
		URL:       req.URL,
		AppKey:    req.AppKey,
		AppSecret: req.AppSecret,
		Status:    req.Status,
	})
	switch {
	case errors.Is(err, adminaimodels.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminaimodels.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminaimodels.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}
