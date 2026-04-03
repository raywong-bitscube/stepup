package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminaudit"
)

type AuditLogsHandler struct {
	service *adminaudit.Service
}

func NewAuditLogsHandler(service *adminaudit.Service) *AuditLogsHandler {
	return &AuditLogsHandler{service: service}
}

func parseAuditLimit(raw string) int {
	const def, max = 100, 500
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (h *AuditLogsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseAuditLimit(r.URL.Query().Get("limit"))
	items, err := h.service.List(r.Context(), limit)
	if errors.Is(err, adminaudit.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if errors.Is(err, adminaudit.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	type row struct {
		ID         uint64          `json:"id"`
		UserID     *uint64         `json:"user_id"`
		UserType   string          `json:"user_type"`
		Action     string          `json:"action"`
		EntityType string          `json:"entity_type"`
		EntityID   *uint64         `json:"entity_id"`
		Snapshot   json.RawMessage `json:"snapshot"`
		IPAddress  *string         `json:"ip_address"`
		CreatedAt  RFC3339Time     `json:"created_at"`
		CreatedBy  uint64          `json:"created_by"`
	}
	out := make([]row, 0, len(items))
	for _, e := range items {
		out = append(out, row{
			ID:         e.ID,
			UserID:     e.UserID,
			UserType:   e.UserType,
			Action:     e.Action,
			EntityType: e.EntityType,
			EntityID:   e.EntityID,
			Snapshot:   e.Snapshot,
			IPAddress:  e.IPAddress,
			CreatedAt:  RFC3339Time(e.CreatedAt),
			CreatedBy:  e.CreatedBy,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}
