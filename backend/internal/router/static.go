package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
)

// registerStatic mounts /admin/ and /student/ from cfg.StaticDir/{admin,student} when the directories exist.
func registerStatic(mux *http.ServeMux, cfg config.Config) {
	root := strings.TrimSpace(cfg.StaticDir)
	if root == "" {
		return
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return
	}

	adminDir := filepath.Join(root, "admin")
	stuDir := filepath.Join(root, "student")
	if fi, err := os.Stat(adminDir); err == nil && fi.IsDir() {
		fs := http.FileServer(http.Dir(adminDir))
		mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
		})
		mux.Handle("GET /admin/", http.StripPrefix("/admin", fs))
	}
	if fi, err := os.Stat(stuDir); err == nil && fi.IsDir() {
		fs := http.FileServer(http.Dir(stuDir))
		mux.HandleFunc("GET /student", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/student/", http.StatusTemporaryRedirect)
		})
		mux.Handle("GET /student/", http.StripPrefix("/student", fs))
	}
}
