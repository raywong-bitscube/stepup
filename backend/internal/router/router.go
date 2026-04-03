package router

import (
	"database/sql"
	"net/http"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/admin"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/health"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/student"
	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminstudents"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentauth"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
)

func New(cfg config.Config, db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Get)
	mux.HandleFunc("GET /readyz", health.Ready)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"stepup","env":"` + cfg.AppEnv + `"}`))
	})

	registerAPIRoutes(mux, cfg, db)
	return middleware.CORS(cfg.CORSAllowedOrigins, mux)
}

func registerAPIRoutes(mux *http.ServeMux, cfg config.Config, db *sql.DB) {
	adminAuthService := adminauth.New(cfg, db)
	adminAuthHandler := admin.NewAuthHandler(adminAuthService)
	adminStudentsHandler := admin.NewStudentsHandler(adminstudents.New(db))
	studentAuthService := studentauth.New(cfg, db)
	studentAuthHandler := student.NewAuthHandler(studentAuthService)
	studentPaperHandler := student.NewPaperHandler(studentpaper.New(cfg, db))

	// Admin routes
	mux.HandleFunc("POST /api/v1/admin/auth/login", adminAuthHandler.Login)
	mux.HandleFunc("POST /api/v1/admin/auth/logout", adminAuthHandler.Logout)
	mux.HandleFunc("GET /api/v1/admin/auth/me", adminAuthHandler.Me)

	// Student auth routes
	mux.HandleFunc("POST /api/v1/student/auth/send-code", studentAuthHandler.SendCode)
	mux.HandleFunc("POST /api/v1/student/auth/verify-code", studentAuthHandler.VerifyCode)
	mux.HandleFunc("POST /api/v1/student/auth/set-password", studentAuthHandler.SetPassword)
	mux.HandleFunc("POST /api/v1/student/auth/login", studentAuthHandler.Login)
	mux.HandleFunc("POST /api/v1/student/auth/logout", studentAuthHandler.Logout)
	mux.HandleFunc("GET /api/v1/student/auth/me", studentAuthHandler.Me)

	// Student paper routes
	mux.HandleFunc("POST /api/v1/student/papers", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Create))
	mux.HandleFunc("GET /api/v1/student/papers", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.List))
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/analysis", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Analysis))
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/plan", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Plan))

	// Admin management routes
	mux.HandleFunc("GET /api/v1/admin/students", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.List))
	mux.HandleFunc("POST /api/v1/admin/students", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.Create))
	mux.HandleFunc("PATCH /api/v1/admin/students/{studentId}", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.Patch))

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
