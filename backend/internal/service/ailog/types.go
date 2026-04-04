package ailog

import (
	"encoding/json"
	"time"
)

// InsertRow is one persisted AI call (no secrets).
type InsertRow struct {
	AIModelID        *uint64
	ModelNameSnap    string
	Action           string
	AdapterKind      string
	ResultStatus     string
	HTTPStatus       *int
	LatencyMS        *int64
	ErrorPhase       string
	ErrorMessage     string
	EndpointHost     string
	ChatModel        string
	FallbackToMock   bool
	PaperID          *uint64
	StudentID        *uint64
	RequestMetaJSON  json.RawMessage
	ResponseMetaJSON json.RawMessage
	RequestBody      string
	ResponseBody     string
}

// ListEntry is a row returned to admin API.
type ListEntry struct {
	ID             uint64          `json:"id"`
	CreatedAt      time.Time       `json:"created_at"`
	AIModelID      *uint64         `json:"ai_model_id"`
	ModelNameSnap  string          `json:"model_name_snapshot"`
	Action         string          `json:"action"`
	AdapterKind    string          `json:"adapter_kind"`
	ResultStatus   string          `json:"result_status"`
	HTTPStatus     *int            `json:"http_status"`
	LatencyMS      *int64          `json:"latency_ms"`
	ErrorPhase     string          `json:"error_phase"`
	ErrorMessage   string          `json:"error_message"`
	EndpointHost   string          `json:"endpoint_host"`
	ChatModel      string          `json:"chat_model"`
	FallbackToMock bool            `json:"fallback_to_mock"`
	PaperID        *uint64         `json:"paper_id"`
	StudentID      *uint64         `json:"student_id"`
	RequestMeta    json.RawMessage `json:"request_meta"`
	ResponseMeta   json.RawMessage `json:"response_meta"`
	RequestBody    string          `json:"request_body"`
	ResponseBody   string          `json:"response_body"`
	Outcome        string          `json:"outcome"`
}

// ListParams filters listing.
type ListParams struct {
	Limit        int
	Offset       int
	AIModelID    *uint64
	Action       string
	ResultStatus string
	AdapterKind  string
	From         *time.Time
	To           *time.Time
}
