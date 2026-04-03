package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminstudents"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
)

type StudentsHandler struct {
	service *adminstudents.Service
	audit   *auditlog.Writer
}

func NewStudentsHandler(service *adminstudents.Service, audit *auditlog.Writer) *StudentsHandler {
	return &StudentsHandler{service: service, audit: audit}
}

func (h *StudentsHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if errors.Is(err, adminstudents.ErrNoDatabase) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}

	type row struct {
		ID        uint64      `json:"id"`
		Phone     *string     `json:"phone"`
		Email     *string     `json:"email"`
		Name      string      `json:"name"`
		Stage     string      `json:"stage"`
		Status    int         `json:"status"`
		CreatedAt RFC3339Time `json:"created_at"`
	}

	out := make([]row, 0, len(items))
	for _, s := range items {
		out = append(out, row{
			ID:        s.ID,
			Phone:     s.Phone,
			Email:     s.Email,
			Name:      s.Name,
			Stage:     s.StageName,
			Status:    s.Status,
			CreatedAt: RFC3339Time(s.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

type createStudentRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Stage      string `json:"stage"`
}

func (h *StudentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	newID, err := h.service.Create(r.Context(), adminstudents.CreateInput{
		Identifier: req.Identifier,
		Password:   req.Password,
		Name:       req.Name,
		Stage:      req.Stage,
	})
	switch {
	case errors.Is(err, adminstudents.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminstudents.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminstudents.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				snap, _ := json.Marshal(map[string]any{"identifier": req.Identifier, "name": req.Name})
				pid := newID
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     "create",
					EntityType: "student",
					EntityID:   &pid,
					Snapshot:   snap,
					IP:         r.RemoteAddr,
					CreatedBy:  adm,
				})
			}
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
	}
}

type patchStudentRequest struct {
	Name     *string `json:"name"`
	Stage    *string `json:"stage"`
	Status   *int    `json:"status"`
	Password *string `json:"password"`
}

func (h *StudentsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	studentIDRaw := strings.TrimSpace(r.PathValue("studentId"))
	studentID, err := strconv.ParseUint(studentIDRaw, 10, 64)
	if err != nil || studentID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err = h.service.Patch(r.Context(), studentID, adminstudents.UpdateInput{
		Name:     req.Name,
		Stage:    req.Stage,
		Status:   req.Status,
		Password: req.Password,
	})
	switch {
	case errors.Is(err, adminstudents.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminstudents.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminstudents.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		if h.audit != nil {
			if sess, ok := middleware.AdminSession(r.Context()); ok && sess.AdminID != 0 {
				adm := sess.AdminID
				act := "update"
				if req.Password != nil {
					act = "password_change"
				}
				snap, _ := json.Marshal(map[string]any{"has_name": req.Name != nil, "has_stage": req.Stage != nil, "has_status": req.Status != nil, "has_password": req.Password != nil})
				sid := studentID
				h.audit.Write(r.Context(), auditlog.Event{
					UserID:     &adm,
					UserType:   "admin",
					Action:     act,
					EntityType: "student",
					EntityID:   &sid,
					Snapshot:   snap,
					IP:         r.RemoteAddr,
					CreatedBy:  adm,
				})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}
