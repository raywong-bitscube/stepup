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
		token := BearerToken(r.Header.Get("Authorization"))
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

