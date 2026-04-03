package student

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentauth"
)

type AuthHandler struct {
	service *studentauth.Service
	audit   *auditlog.Writer
}

func NewAuthHandler(service *studentauth.Service, audit *auditlog.Writer) *AuthHandler {
	return &AuthHandler{service: service, audit: audit}
}

type identifierRequest struct {
	Identifier string `json:"identifier"`
}

type verifyRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
}

type setPasswordRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req identifierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	code, err := h.service.SendCode(req.Identifier)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"note":   "dev-only response includes verification code",
		"code":   code,
	})
}

func (h *AuthHandler) VerifyCode(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err := h.service.VerifyCode(req.Identifier, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, studentauth.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		case errors.Is(err, studentauth.ErrCodeExpired):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "CODE_EXPIRED"})
		case errors.Is(err, studentauth.ErrCodeUsed):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "CODE_USED"})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "CODE_INVALID"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *AuthHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req setPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	if err := h.service.SetPassword(req.Identifier, req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	session, err := h.service.Login(req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, studentauth.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		case errors.Is(err, studentauth.ErrPasswordUnset):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "PASSWORD_UNSET"})
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		}
		return
	}

	if h.audit != nil && session.StudentID != 0 {
		snap, _ := json.Marshal(map[string]any{"identifier": session.Identifier})
		sid := session.StudentID
		h.audit.Write(r.Context(), auditlog.Event{
			UserID:     &sid,
			UserType:   "student",
			Action:     "login",
			EntityType: "student",
			EntityID:   &sid,
			Snapshot:   snap,
			IP:         r.RemoteAddr,
			CreatedBy:  0,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":      session.Token,
		"expires_at": session.ExpiresAt,
		"user": map[string]any{
			"identifier": session.Identifier,
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
			"identifier": session.Identifier,
		},
		"expires_at": session.ExpiresAt,
		"last_seen":  session.LastSeenAt,
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
