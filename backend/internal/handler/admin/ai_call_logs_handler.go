package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/service/ailog"
)

type AICallLogsHandler struct {
	service *ailog.ListService
}

func NewAICallLogsHandler(service *ailog.ListService) *AICallLogsHandler {
	return &AICallLogsHandler{service: service}
}

func parseAICallLimit(raw string) int {
	if raw == "" {
		return 50
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

func parseAICallOffset(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func parseOptionalUint64(raw string) (*uint64, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseTimeBound(raw string, endOfDay bool) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Nanosecond)
	}
	return &t, nil
}

func (h *AICallLogsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := ailog.ListParams{
		Limit:        parseAICallLimit(q.Get("limit")),
		Offset:       parseAICallOffset(q.Get("offset")),
		Action:       q.Get("action"),
		ResultStatus: q.Get("result_status"),
		AdapterKind:  q.Get("adapter_kind"),
	}

	var err error
	p.AIModelID, err = parseOptionalUint64(q.Get("ai_model_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	p.From, err = parseTimeBound(q.Get("from"), false)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	p.To, err = parseTimeBound(q.Get("to"), true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}

	items, err := h.service.List(r.Context(), p)
	if errors.Is(err, ailog.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}

	type row struct {
		ID             uint64          `json:"id"`
		CreatedAt      RFC3339Time     `json:"created_at"`
		ModelNameSnap  string          `json:"model_name_snapshot"`
		Action         string          `json:"action"`
		AdapterKind    string          `json:"adapter_kind"`
		Outcome        string          `json:"outcome"`
		LatencyMS      *int64          `json:"latency_ms"`
		ErrorPhase     string          `json:"error_phase"`
		ErrorMessage   string          `json:"error_message"`
		EndpointHost   string          `json:"endpoint_host"`
		ChatModel      string          `json:"chat_model"`
		FallbackToMock bool            `json:"fallback_to_mock"`
		RequestMeta    json.RawMessage `json:"request_meta"`
		ResponseMeta   json.RawMessage `json:"response_meta"`
		RequestBody    string          `json:"request_body"`
		ResponseBody   string          `json:"response_body"`
	}
	out := make([]row, 0, len(items))
	for _, e := range items {
		out = append(out, row{
			ID:             e.ID,
			CreatedAt:      RFC3339Time(e.CreatedAt),
			ModelNameSnap:  e.ModelNameSnap,
			Action:         e.Action,
			AdapterKind:    e.AdapterKind,
			Outcome:        e.Outcome,
			LatencyMS:      e.LatencyMS,
			ErrorPhase:     e.ErrorPhase,
			ErrorMessage:   e.ErrorMessage,
			EndpointHost:   e.EndpointHost,
			ChatModel:      e.ChatModel,
			FallbackToMock: e.FallbackToMock,
			RequestMeta:    e.RequestMeta,
			ResponseMeta:   e.ResponseMeta,
			RequestBody:    e.RequestBody,
			ResponseBody:   e.ResponseBody,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}
