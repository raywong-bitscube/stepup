package router

import (
	"net/http"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/admin"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/health"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
)

func New(cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Get)
	mux.HandleFunc("GET /readyz", health.Ready)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"stepup","env":"` + cfg.AppEnv + `"}`))
	})

	registerAPIRoutes(mux, cfg)
	return mux
}

func registerAPIRoutes(mux *http.ServeMux, cfg config.Config) {
	adminAuthHandler := admin.NewAuthHandler(adminauth.New(cfg))

	// Admin routes
	mux.HandleFunc("POST /api/v1/admin/auth/login", adminAuthHandler.Login)
	mux.HandleFunc("POST /api/v1/admin/auth/logout", adminAuthHandler.Logout)
	mux.HandleFunc("GET /api/v1/admin/auth/me", adminAuthHandler.Me)

	// Student auth routes
	mux.HandleFunc("POST /api/v1/student/auth/send-code", notImplemented)
	mux.HandleFunc("POST /api/v1/student/auth/verify-code", notImplemented)
	mux.HandleFunc("POST /api/v1/student/auth/set-password", notImplemented)
	mux.HandleFunc("POST /api/v1/student/auth/login", notImplemented)

	// Student paper routes
	mux.HandleFunc("POST /api/v1/student/papers", notImplemented)
	mux.HandleFunc("GET /api/v1/student/papers", notImplemented)
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/analysis", notImplemented)
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/plan", notImplemented)

	// Admin management routes
	mux.HandleFunc("GET /api/v1/admin/students", notImplemented)
	mux.HandleFunc("POST /api/v1/admin/students", notImplemented)
	mux.HandleFunc("PATCH /api/v1/admin/students/{studentId}", notImplemented)

	mux.HandleFunc("GET /api/v1/admin/subjects", notImplemented)
	mux.HandleFunc("POST /api/v1/admin/subjects", notImplemented)
	mux.HandleFunc("PATCH /api/v1/admin/subjects/{subjectId}", notImplemented)

	mux.HandleFunc("GET /api/v1/admin/stages", notImplemented)
	mux.HandleFunc("POST /api/v1/admin/stages", notImplemented)
	mux.HandleFunc("PATCH /api/v1/admin/stages/{stageId}", notImplemented)

	mux.HandleFunc("GET /api/v1/admin/ai-models", notImplemented)
	mux.HandleFunc("POST /api/v1/admin/ai-models", notImplemented)
	mux.HandleFunc("PATCH /api/v1/admin/ai-models/{modelId}", notImplemented)

	mux.HandleFunc("GET /api/v1/admin/prompts", notImplemented)
	mux.HandleFunc("POST /api/v1/admin/prompts", notImplemented)
	mux.HandleFunc("PATCH /api/v1/admin/prompts/{promptId}", notImplemented)

	mux.HandleFunc("GET /api/v1/admin/audit-logs", notImplemented)
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","message":"endpoint scaffolded"}`))
}
