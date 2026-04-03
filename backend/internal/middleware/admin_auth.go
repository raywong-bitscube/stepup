package middleware

import (
	"context"
	"net/http"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
)

type adminCtxKey string

const adminSessionKey adminCtxKey = "admin_session"

func RequireAdminAuth(service *adminauth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		sess, err := service.Current(token)
		if err != nil {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), adminSessionKey, sess)
		next(w, r.WithContext(ctx))
	}
}

func AdminSession(ctx context.Context) (adminauth.Session, bool) {
	v, ok := ctx.Value(adminSessionKey).(adminauth.Session)
	return v, ok
}
