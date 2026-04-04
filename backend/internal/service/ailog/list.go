package ailog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrNoDatabase = errors.New("database not configured")

type ListService struct {
	db *sql.DB
}

func NewListService(db *sql.DB) *ListService {
	return &ListService{db: db}
}

func clampLimit(n int) int {
	const def, max = 50, 200
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (s *ListService) List(ctx context.Context, p ListParams) ([]ListEntry, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	limit := clampLimit(p.Limit)
	if p.Offset < 0 {
		p.Offset = 0
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var conds []string
	var args []any

	if p.AIModelID != nil {
		conds = append(conds, "ai_model_id = ?")
		args = append(args, *p.AIModelID)
	}
	if p.Action != "" {
		conds = append(conds, "action = ?")
		args = append(args, p.Action)
	}
	if p.ResultStatus != "" {
		conds = append(conds, "result_status = ?")
		args = append(args, p.ResultStatus)
	}
	if p.AdapterKind != "" {
		conds = append(conds, "adapter_kind = ?")
		args = append(args, p.AdapterKind)
	}
	if p.From != nil {
		conds = append(conds, "created_at >= ?")
		args = append(args, *p.From)
	}
	if p.To != nil {
		conds = append(conds, "created_at <= ?")
		args = append(args, *p.To)
	}

	q := `
SELECT id, created_at, ai_model_id, model_name_snapshot, action, adapter_kind, result_status,
       http_status, latency_ms, error_phase, error_message, endpoint_host, chat_model,
       fallback_to_mock, paper_id, student_id, request_meta, response_meta,
       request_body, response_body
FROM ai_call_log`
	if len(conds) > 0 {
		q += " WHERE " + joinAnd(conds)
	}
	q += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, p.Offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ListEntry, 0, limit)
	for rows.Next() {
		var (
			e            ListEntry
			aid          sql.NullInt64
			httpSt       sql.NullInt64
			lat          sql.NullInt64
			paper        sql.NullInt64
			stu          sql.NullInt64
			reqM         sql.NullString
			respM        sql.NullString
			reqB         sql.NullString
			respB        sql.NullString
			fallbackTiny int
		)
		if err := rows.Scan(
			&e.ID, &e.CreatedAt, &aid, &e.ModelNameSnap, &e.Action, &e.AdapterKind, &e.ResultStatus,
			&httpSt, &lat, &e.ErrorPhase, &e.ErrorMessage, &e.EndpointHost, &e.ChatModel,
			&fallbackTiny, &paper, &stu, &reqM, &respM, &reqB, &respB,
		); err != nil {
			return nil, err
		}
		if aid.Valid {
			v := uint64(aid.Int64)
			e.AIModelID = &v
		}
		if httpSt.Valid {
			v := int(httpSt.Int64)
			e.HTTPStatus = &v
		}
		if lat.Valid {
			v := lat.Int64
			e.LatencyMS = &v
		}
		if paper.Valid {
			v := uint64(paper.Int64)
			e.PaperID = &v
		}
		if stu.Valid {
			v := uint64(stu.Int64)
			e.StudentID = &v
		}
		e.FallbackToMock = fallbackTiny != 0
		if reqM.Valid && reqM.String != "" {
			e.RequestMeta = json.RawMessage(reqM.String)
		} else {
			e.RequestMeta = json.RawMessage("null")
		}
		if respM.Valid && respM.String != "" {
			e.ResponseMeta = json.RawMessage(respM.String)
		} else {
			e.ResponseMeta = json.RawMessage("null")
		}
		if reqB.Valid {
			e.RequestBody = reqB.String
		}
		if respB.Valid {
			e.ResponseBody = respB.String
		}
		e.Outcome = FormatOutcome(e.ResultStatus, e.HTTPStatus)
		out = append(out, e)
	}
	return out, rows.Err()
}

func joinAnd(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += " AND " + parts[i]
	}
	return s
}
