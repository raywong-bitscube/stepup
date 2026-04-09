package router

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/admin"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/catalog"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/health"
	"github.com/raywong-bitscube/stepup/backend/internal/handler/student"
	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminaimodels"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminaudit"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminpapers"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminprompts"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminslidegen"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminslidedecks"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminstages"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminstudents"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminsubjects"
	"github.com/raywong-bitscube/stepup/backend/internal/service/admintextbookcatalog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/ailog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentauth"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentessayoutline"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentslidedeck"
)

func New(cfg config.Config, db *sqlx.DB) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Get)
	mux.HandleFunc("GET /readyz", health.ReadyHandler(cfg.DBDSN != "", db))
	// Go 1.22+ ServeMux: pattern "GET /" (no trailing slash) matches only path "/" exactly, not /admin/ or /student/.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"stepup","env":"` + cfg.AppEnv + `","ui":{"admin":"/admin/","student":"/student/"}}`))
	})

	registerAPIRoutes(mux, cfg, db)
	registerStatic(mux, cfg)
	return middleware.CORS(cfg.CORSAllowedOrigins, mux)
}

func registerAPIRoutes(mux *http.ServeMux, cfg config.Config, db *sqlx.DB) {
	auditWriter := auditlog.New(db)
	adminAuthService := adminauth.New(cfg, db)
	adminAuthHandler := admin.NewAuthHandler(adminAuthService, auditWriter)
	adminStudentsHandler := admin.NewStudentsHandler(adminstudents.New(db), auditWriter)
	adminSubjectsHandler := admin.NewSubjectsHandler(adminsubjects.New(db), auditWriter)
	adminTextbookCatalogHandler := admin.NewTextbookCatalogHandler(admintextbookcatalog.New(db))
	adminStagesHandler := admin.NewStagesHandler(adminstages.New(db), auditWriter)
	adminAIModelsHandler := admin.NewAIModelsHandler(adminaimodels.New(db), auditWriter)
	adminPromptsHandler := admin.NewPromptsHandler(adminprompts.New(db), auditWriter)
	adminStudentPapersHandler := admin.NewStudentPapersHandler(adminpapers.New(db))
	adminAuditLogsHandler := admin.NewAuditLogsHandler(adminaudit.New(db))
	adminAICallLogsHandler := admin.NewAICallLogsHandler(ailog.NewListService(db))
	adminSlideDecksHandler := admin.NewSlideDecksHandler(adminslidedecks.New(db), auditWriter)
	adminSlideDeckGenHandler := admin.NewSlideDeckGenHandler(adminslidegen.New(cfg, db))
	studentAuthService := studentauth.New(cfg, db)
	studentAuthHandler := student.NewAuthHandler(studentAuthService, auditWriter)
	studentPaperHandler := student.NewPaperHandler(studentpaper.New(cfg, db), auditWriter)
	studentEssayHandler := student.NewEssayOutlineHandler(studentessayoutline.New(cfg, db))
	studentSlideDeckHandler := student.NewSlideDeckHandler(studentslidedeck.New(db))

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

	mux.HandleFunc("GET /api/v1/catalog", catalog.Handler(db))

	// Student paper routes
	mux.HandleFunc("POST /api/v1/student/papers", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Create))
	mux.HandleFunc("GET /api/v1/student/papers", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.List))
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/analysis", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Analysis))
	mux.HandleFunc("GET /api/v1/student/papers/{paperId}/plan", middleware.RequireStudentAuth(studentAuthService, studentPaperHandler.Plan))

	mux.HandleFunc("POST /api/v1/student/essay-outline/generate-topic", middleware.RequireStudentAuth(studentAuthService, studentEssayHandler.GenerateTopic))
	mux.HandleFunc("POST /api/v1/student/essay-outline/ocr-topic", middleware.RequireStudentAuth(studentAuthService, studentEssayHandler.OCRTopic))
	mux.HandleFunc("POST /api/v1/student/essay-outline/review", middleware.RequireStudentAuth(studentAuthService, studentEssayHandler.Review))
	mux.HandleFunc("GET /api/v1/student/essay-outline/practices", middleware.RequireStudentAuth(studentAuthService, studentEssayHandler.ListPractices))
	mux.HandleFunc("GET /api/v1/student/essay-outline/practices/{practiceId}", middleware.RequireStudentAuth(studentAuthService, studentEssayHandler.GetPractice))

	mux.HandleFunc("GET /api/v1/student/sections/{sectionId}/slide-deck", middleware.RequireStudentAuth(studentAuthService, studentSlideDeckHandler.GetActive))

	// Admin management routes
	mux.HandleFunc("GET /api/v1/admin/students", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.List))
	mux.HandleFunc("POST /api/v1/admin/students", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.Create))
	mux.HandleFunc("PATCH /api/v1/admin/students/{studentId}", middleware.RequireAdminAuth(adminAuthService, adminStudentsHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/students/{studentId}/papers/{paperId}/analysis", middleware.RequireAdminAuth(adminAuthService, adminStudentPapersHandler.Analysis))
	mux.HandleFunc("GET /api/v1/admin/students/{studentId}/papers/{paperId}/plan", middleware.RequireAdminAuth(adminAuthService, adminStudentPapersHandler.Plan))
	mux.HandleFunc("GET /api/v1/admin/students/{studentId}/papers", middleware.RequireAdminAuth(adminAuthService, adminStudentPapersHandler.List))

	mux.HandleFunc("GET /api/v1/admin/subjects", middleware.RequireAdminAuth(adminAuthService, adminSubjectsHandler.List))
	mux.HandleFunc("POST /api/v1/admin/subjects", middleware.RequireAdminAuth(adminAuthService, adminSubjectsHandler.Create))
	mux.HandleFunc("GET /api/v1/admin/subjects/{subjectId}/textbooks", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.ListTextbooksBySubject))
	mux.HandleFunc("PATCH /api/v1/admin/subjects/{subjectId}", middleware.RequireAdminAuth(adminAuthService, adminSubjectsHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/textbooks/{textbookId}/chapters", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.ListChapters))
	mux.HandleFunc("PATCH /api/v1/admin/textbooks/{textbookId}", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.PatchTextbook))
	mux.HandleFunc("GET /api/v1/admin/chapters/{chapterId}/sections", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.ListSections))
	mux.HandleFunc("PATCH /api/v1/admin/chapters/{chapterId}", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.PatchChapter))
	mux.HandleFunc("PATCH /api/v1/admin/sections/{sectionId}", middleware.RequireAdminAuth(adminAuthService, adminTextbookCatalogHandler.PatchSection))

	mux.HandleFunc("GET /api/v1/admin/sections/{sectionId}/slide-decks", middleware.RequireAdminAuth(adminAuthService, adminSlideDecksHandler.ListBySection))
	mux.HandleFunc("POST /api/v1/admin/sections/{sectionId}/slide-decks", middleware.RequireAdminAuth(adminAuthService, adminSlideDecksHandler.Create))
	mux.HandleFunc("GET /api/v1/admin/sections/{sectionId}/slide-generate/default-prompt", middleware.RequireAdminAuth(adminAuthService, adminSlideDeckGenHandler.DefaultPrompt))
	mux.HandleFunc("POST /api/v1/admin/sections/{sectionId}/slide-decks/generate-ai", middleware.RequireAdminAuth(adminAuthService, adminSlideDeckGenHandler.GenerateAI))
	mux.HandleFunc("GET /api/v1/admin/slide-decks/{deckId}", middleware.RequireAdminAuth(adminAuthService, adminSlideDecksHandler.Get))
	mux.HandleFunc("PATCH /api/v1/admin/slide-decks/{deckId}", middleware.RequireAdminAuth(adminAuthService, adminSlideDecksHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/stages", middleware.RequireAdminAuth(adminAuthService, adminStagesHandler.List))
	mux.HandleFunc("POST /api/v1/admin/stages", middleware.RequireAdminAuth(adminAuthService, adminStagesHandler.Create))
	mux.HandleFunc("PATCH /api/v1/admin/stages/{stageId}", middleware.RequireAdminAuth(adminAuthService, adminStagesHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/ai-models", middleware.RequireAdminAuth(adminAuthService, adminAIModelsHandler.List))
	mux.HandleFunc("POST /api/v1/admin/ai-models", middleware.RequireAdminAuth(adminAuthService, adminAIModelsHandler.Create))
	mux.HandleFunc("PATCH /api/v1/admin/ai-models/{modelId}", middleware.RequireAdminAuth(adminAuthService, adminAIModelsHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/prompts", middleware.RequireAdminAuth(adminAuthService, adminPromptsHandler.List))
	mux.HandleFunc("PATCH /api/v1/admin/prompts/{promptId}", middleware.RequireAdminAuth(adminAuthService, adminPromptsHandler.Patch))

	mux.HandleFunc("GET /api/v1/admin/audit-logs", middleware.RequireAdminAuth(adminAuthService, adminAuditLogsHandler.List))
	mux.HandleFunc("GET /api/v1/admin/ai-call-logs", middleware.RequireAdminAuth(adminAuthService, adminAICallLogsHandler.List))
}
