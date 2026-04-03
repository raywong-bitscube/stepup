package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// ReadyHandler returns 200 when the process can serve traffic.
// If requireDB is true, a non-nil *sql.DB must ping successfully; otherwise 503.
func ReadyHandler(requireDB bool, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !requireDB {
			_, _ = w.Write([]byte(`{"status":"ready"}`))
			return
		}
		if db == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "not_ready",
				"code":   "DATABASE_UNAVAILABLE",
			})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "not_ready",
				"code":   "DATABASE_UNREACHABLE",
			})
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}
