package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminstudents"
)

type StudentsHandler struct {
	service *adminstudents.Service
}

func NewStudentsHandler(service *adminstudents.Service) *StudentsHandler {
	return &StudentsHandler{service: service}
}

func (h *StudentsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if errors.Is(err, adminstudents.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}

	type row struct {
		ID        uint64    `json:"id"`
		Phone     *string   `json:"phone"`
		Email     *string   `json:"email"`
		Name      string    `json:"name"`
		Stage     string    `json:"stage"`
		Status    int       `json:"status"`
		CreatedAt timeJSON  `json:"created_at"`
	}

	out := make([]row, 0, len(items))
	for _, s := range items {
		out = append(out, row{
			ID:        s.ID,
			Phone:     s.Phone,
			Email:     s.Email,
			Name:      s.Name,
			Stage:     s.StageName,
			Status:    s.Status,
			CreatedAt: timeJSON(s.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

// timeJSON encodes time in RFC3339 for JSON without exporting a custom type on the public API structs.
type timeJSON time.Time

func (t timeJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UTC().Format(time.RFC3339Nano))
}
