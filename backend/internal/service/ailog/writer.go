package ailog

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/dbutil"
)

type Writer struct {
	db *sqlx.DB
}

func NewWriter(db *sqlx.DB) *Writer {
	return &Writer{db: db}
}

func (w *Writer) Write(ctx context.Context, row InsertRow) {
	if w == nil || w.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var aid sql.NullInt64
	if row.ProviderModelID != nil {
		aid = sql.NullInt64{Int64: int64(*row.ProviderModelID), Valid: true}
	}
	var httpSt sql.NullInt64
	if row.HTTPStatus != nil {
		httpSt = sql.NullInt64{Int64: int64(*row.HTTPStatus), Valid: true}
	}
	var lat sql.NullInt64
	if row.LatencyMS != nil {
		lat = sql.NullInt64{Int64: *row.LatencyMS, Valid: true}
	}
	var stu sql.NullInt64
	if row.SysUserID != nil {
		stu = sql.NullInt64{Int64: int64(*row.SysUserID), Valid: true}
	}
	var refTbl sql.NullString
	if row.RefTable != nil && strings.TrimSpace(*row.RefTable) != "" {
		refTbl = sql.NullString{String: strings.TrimSpace(*row.RefTable), Valid: true}
	}
	var refID sql.NullInt64
	if row.RefID != nil {
		refID = sql.NullInt64{Int64: int64(*row.RefID), Valid: true}
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

	_, _ = w.db.ExecContext(ctx, dbutil.Rebind(`
INSERT INTO ai_call_log (
  ai_provider_model_id, model_name_snapshot, action, adapter_kind, result_status,
  http_status, latency_ms, error_phase, error_message, endpoint_host, chat_model,
  fallback_to_mock, sys_user_id, ref_table, ref_id, request_meta, response_meta,
  request_body, response_body
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`), nullInt64(aid), row.ModelNameSnap, row.Action, row.AdapterKind, row.ResultStatus,
		nullInt64(httpSt), nullInt64(lat), row.ErrorPhase, row.ErrorMessage,
		row.EndpointHost, row.ChatModel, fallback,
		nullInt64(stu), nullString(refTbl), nullInt64(refID), req, resp, rb, rsp)
}

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func nullString(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	return n.String
}
