package middleware

import (
	"net/http"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
)

func RequireAdminAuth(service *adminauth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		if _, err := service.Current(token); err != nil {
			http.Error(w, `{"code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
