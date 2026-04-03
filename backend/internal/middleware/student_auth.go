package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/studentauth"
)

type studentCtxKey string

const studentIdentifierKey studentCtxKey = "student_identifier"

func RequireStudentAuth(service *studentauth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		session, err := service.Current(token)
		if err != nil {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), studentIdentifierKey, session.Identifier)
		next(w, r.WithContext(ctx))
	}
}

func StudentIdentifier(ctx context.Context) string {
	v, _ := ctx.Value(studentIdentifierKey).(string)
	return strings.TrimSpace(v)
}

func bearerToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
