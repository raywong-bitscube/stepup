package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReadyHandler returns 200 when the process can serve traffic.
// If requireDB is true, a non-nil *sqlx.DB must ping successfully; otherwise 503.
func ReadyHandler(requireDB bool, db *sqlx.DB) http.HandlerFunc {
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
