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
	AdminSessionTTL        time.Duration
	CORSAllowedOrigins     []string
	// StaticDir, if set, serves bundled UIs at /admin/ and /student/ (see Dockerfile).
	StaticDir string
}

func Load() Config {
	sessionHours, err := strconv.Atoi(getenv("ADMIN_SESSION_TTL_HOURS", "24"))
	if err != nil || sessionHours <= 0 {
		sessionHours = 24
	}
	aiTimeoutSec, err := strconv.Atoi(getenv("AI_REQUEST_TIMEOUT_SECONDS", "30"))
	if err != nil || aiTimeoutSec <= 0 {
		aiTimeoutSec = 30
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
		AdminSessionTTL:        time.Duration(sessionHours) * time.Hour,
		CORSAllowedOrigins:     splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:8080,http://127.0.0.1:8080")),
		StaticDir:              strings.TrimSpace(getenv("STATIC_DIR", "")),
	}
}

func (c Config) HTTPAddress() string {
	return c.HTTPHost + ":" + c.HTTPPort
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
