package middleware

import (
	"net/http"
	"strings"
)

func originAllowedToReflect(origin string) bool {
	if origin == "" {
		return false
	}
	return strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://")
}

// CORS handles OPTIONS preflight and sets Access-Control-* on responses.
// If allowedOrigins contains the token "*", any request Origin that looks like http(s) is echoed back
// (for LAN / public-IP + split-port UIs where enumerating every Origin is awkward). Do not use * on
// internet-facing production unless you accept that browser callers from any page origin are permitted.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	reflectAny := false
	for _, origin := range allowedOrigins {
		v := strings.TrimSpace(origin)
		if v == "" {
			continue
		}
		if v == "*" {
			reflectAny = true
			continue
		}
		allowed[v] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			ok := false
			if reflectAny && originAllowedToReflect(origin) {
				ok = true
			} else if _, exists := allowed[origin]; exists {
				ok = true
			}
			if ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
