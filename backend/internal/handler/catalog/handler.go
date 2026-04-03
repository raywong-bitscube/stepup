package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// Handler returns active subjects and stages for student forms (no auth).
func Handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, `{"code":"METHOD_NOT_ALLOWED"}`, http.StatusMethodNotAllowed)
			return
		}
		if db == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "DATABASE_REQUIRED"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		type item struct {
			ID   uint64 `json:"id"`
			Name string `json:"name"`
		}

		subjects, err := loadNames(ctx, db, `
SELECT id, name FROM subject WHERE status = 1 AND is_deleted = 0 ORDER BY id ASC`)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "INTERNAL_ERROR"})
			return
		}
		stages, err := loadNames(ctx, db, `
SELECT id, name FROM stage WHERE status = 1 AND is_deleted = 0 ORDER BY id ASC`)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "INTERNAL_ERROR"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"subjects": subjects,
			"stages":   stages,
		})
	}
}

func loadNames(ctx context.Context, db *sql.DB, q string) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0, 8)
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	return out, rows.Err()
}
