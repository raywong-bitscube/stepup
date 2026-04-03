package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv                 string
	HTTPHost               string
	HTTPPort               string
	DBDSN                  string
	AdminBootstrapUsername string
	AdminBootstrapPassword string
	AdminSessionTTL        time.Duration
}

func Load() Config {
	sessionHours, err := strconv.Atoi(getenv("ADMIN_SESSION_TTL_HOURS", "24"))
	if err != nil || sessionHours <= 0 {
		sessionHours = 24
	}

	return Config{
		AppEnv:                 getenv("APP_ENV", "dev"),
		HTTPHost:               getenv("HTTP_HOST", "0.0.0.0"),
		HTTPPort:               getenv("HTTP_PORT", "8080"),
		DBDSN:                  getenv("DB_DSN", ""),
		AdminBootstrapUsername: getenv("ADMIN_BOOTSTRAP_USERNAME", "admin"),
		AdminBootstrapPassword: getenv("ADMIN_BOOTSTRAP_PASSWORD", "admin123"),
		AdminSessionTTL:        time.Duration(sessionHours) * time.Hour,
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
