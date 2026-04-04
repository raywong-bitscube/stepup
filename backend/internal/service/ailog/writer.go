package ailog

import (
	"context"
	"database/sql"
	"time"
)

type Writer struct {
	db *sql.DB
}

func NewWriter(db *sql.DB) *Writer {
	return &Writer{db: db}
}

func (w *Writer) Write(ctx context.Context, row InsertRow) {
	if w == nil || w.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var aid sql.NullInt64
	if row.AIModelID != nil {
		aid = sql.NullInt64{Int64: int64(*row.AIModelID), Valid: true}
	}
	var httpSt sql.NullInt64
	if row.HTTPStatus != nil {
		httpSt = sql.NullInt64{Int64: int64(*row.HTTPStatus), Valid: true}
	}
	var lat sql.NullInt64
	if row.LatencyMS != nil {
		lat = sql.NullInt64{Int64: *row.LatencyMS, Valid: true}
	}
	var paper sql.NullInt64
	if row.PaperID != nil {
		paper = sql.NullInt64{Int64: int64(*row.PaperID), Valid: true}
	}
	var stu sql.NullInt64
	if row.StudentID != nil {
		stu = sql.NullInt64{Int64: int64(*row.StudentID), Valid: true}
	}

	req := row.RequestMetaJSON
	if len(req) == 0 {
		req = []byte("null")
	}
	resp := row.ResponseMetaJSON
	if len(resp) == 0 {
		resp = []byte("null")
	}

	fallback := 0
	if row.FallbackToMock {
		fallback = 1
	}

	rb := row.RequestBody
	rb = TruncateBody(rb)
	rsp := row.ResponseBody
	rsp = TruncateBody(rsp)

	_, _ = w.db.ExecContext(ctx, `
INSERT INTO ai_call_log (
  ai_model_id, model_name_snapshot, action, adapter_kind, result_status,
  http_status, latency_ms, error_phase, error_message, endpoint_host, chat_model,
  fallback_to_mock, paper_id, student_id, request_meta, response_meta,
  request_body, response_body
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, nullInt64(aid), row.ModelNameSnap, row.Action, row.AdapterKind, row.ResultStatus,
		nullInt64(httpSt), nullInt64(lat), row.ErrorPhase, row.ErrorMessage,
		row.EndpointHost, row.ChatModel, fallback,
		nullInt64(paper), nullInt64(stu), req, resp, rb, rsp)
}

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}
