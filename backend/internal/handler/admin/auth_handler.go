package admin

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/adminauth"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
)

type AuthHandler struct {
	service *adminauth.Service
	audit   *auditlog.Writer
}

func NewAuthHandler(service *adminauth.Service, audit *auditlog.Writer) *AuthHandler {
	return &AuthHandler{service: service, audit: audit}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}

	session, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, adminauth.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		case errors.Is(err, adminauth.ErrUnauthorized):
			writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		default:
			log.Printf("admin auth login failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		}
		return
	}

	if h.audit != nil {
		snap, _ := json.Marshal(map[string]any{"username": session.Username, "role": session.Role})
		var uid *uint64
		cb := uint64(0)
		if session.AdminID != 0 {
			uid = &session.AdminID
			cb = session.AdminID
		}
		h.audit.Write(r.Context(), auditlog.Event{
			UserID:     uid,
			UserType:   "admin",
			Action:     "login",
			EntityType: "admin",
			EntityID:   uid,
			Snapshot:   snap,
			IP:         r.RemoteAddr,
			CreatedBy:  cb,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
		"user": map[string]any{
			"username": session.Username,
			"role":     session.Role,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	h.service.Logout(token)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	session, err := h.service.Current(token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"username": session.Username,
			"role":     session.Role,
		},
		"expires_at": session.ExpiresAt,
		"last_seen":  session.LastSeen,
	})
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

func writeJSON(w http.ResponseWriter, code int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
