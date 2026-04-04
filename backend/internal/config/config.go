package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                 string
	HTTPHost               string
	HTTPPort               string
	DBDSN                  string
	AnalysisAdapter        string
	AIEndpoint             string
	AIRequestTimeout       time.Duration
	AdminBootstrapUsername string
	AdminBootstrapPassword string
	// SessionTTL applies to admin and student login sessions (DB-backed and in-memory).
	SessionTTL         time.Duration
	CORSAllowedOrigins []string
	// StaticDir, if set, serves bundled UIs at /admin/ and /student/ (see Dockerfile).
	StaticDir string
	// UploadDir is where student paper files are stored; also served at GET /uploads/ when non-empty.
	UploadDir string
}

func Load() Config {
	aiTimeoutSec, err := strconv.Atoi(getenv("AI_REQUEST_TIMEOUT_SECONDS", "180"))
	if err != nil || aiTimeoutSec <= 0 {
		aiTimeoutSec = 180
	}

	return Config{
		AppEnv:                 getenv("APP_ENV", "dev"),
		HTTPHost:               getenv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:               getenv("HTTP_PORT", "8080"),
		DBDSN:                  getenv("DB_DSN", ""),
		AnalysisAdapter:        getenv("ANALYSIS_ADAPTER", "mock"),
		AIEndpoint:             getenv("AI_ENDPOINT", ""),
		AIRequestTimeout:       time.Duration(aiTimeoutSec) * time.Second,
		AdminBootstrapUsername: getenv("ADMIN_BOOTSTRAP_USERNAME", "admin"),
		AdminBootstrapPassword: getenv("ADMIN_BOOTSTRAP_PASSWORD", "admin123"),
		SessionTTL:             loadSessionTTL(),
		CORSAllowedOrigins:     splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:8080,http://127.0.0.1:8080")),
		StaticDir:              strings.TrimSpace(getenv("STATIC_DIR", "")),
		UploadDir:              strings.TrimSpace(getenv("UPLOAD_DIR", "data/uploads")),
	}
}

func (c Config) HTTPAddress() string {
	return c.HTTPHost + ":" + c.HTTPPort
}

// loadSessionTTL: SESSION_TTL_MINUTES (default 30) takes precedence; if unset, ADMIN_SESSION_TTL_HOURS (legacy) is used when set.
func loadSessionTTL() time.Duration {
	if v := strings.TrimSpace(os.Getenv("SESSION_TTL_MINUTES")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	if v := strings.TrimSpace(os.Getenv("ADMIN_SESSION_TTL_HOURS")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return time.Duration(n) * time.Hour
		}
	}
	return 30 * time.Minute
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
