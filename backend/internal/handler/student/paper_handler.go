package student

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/auditlog"
	"github.com/raywong-bitscube/stepup/backend/internal/service/studentpaper"
)

var errPaperFileTooLarge = errors.New("paper file too large")

type PaperHandler struct {
	service *studentpaper.Service
	audit   *auditlog.Writer
}

func NewPaperHandler(service *studentpaper.Service, audit *auditlog.Writer) *PaperHandler {
	return &PaperHandler{service: service, audit: audit}
}

func (h *PaperHandler) Create(w http.ResponseWriter, r *http.Request) {
	const maxMultipartBytes = 120 << 20
	if err := r.ParseMultipartForm(maxMultipartBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_MULTIPART"})
		return
	}

	identifier := middleware.StudentIdentifier(r.Context())
	subject := strings.TrimSpace(r.FormValue("subject"))
	stage := strings.TrimSpace(r.FormValue("stage"))
	if subject == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}

	const maxBytes = 25 << 20
	var parts []studentpaper.UploadPart

	appendHeader := func(fh *multipart.FileHeader) error {
		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
		if err != nil {
			return err
		}
		if len(raw) > maxBytes {
			return errPaperFileTooLarge
		}
		parts = append(parts, studentpaper.UploadPart{
			Filename:    fh.Filename,
			ContentType: fh.Header.Get("Content-Type"),
			Bytes:       raw,
		})
		return nil
	}

	if r.MultipartForm != nil && len(r.MultipartForm.File["files"]) > 0 {
		for _, fh := range r.MultipartForm.File["files"] {
			if err := appendHeader(fh); err != nil {
				if errors.Is(err, errPaperFileTooLarge) {
					writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"code": "FILE_TOO_LARGE"})
					return
				}
				writeJSON(w, http.StatusBadRequest, map[string]any{"code": "READ_FAILED"})
				return
			}
		}
	} else {
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "FILE_REQUIRED"})
			return
		}
		defer func() { _ = file.Close() }()
		raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "READ_FAILED"})
			return
		}
		if len(raw) > maxBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"code": "FILE_TOO_LARGE"})
			return
		}
		parts = append(parts, studentpaper.UploadPart{
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			Bytes:       raw,
		})
	}

	paper, err := h.service.Create(identifier, subject, stage, parts)
	if err != nil {
		switch {
		case errors.Is(err, studentpaper.ErrStudentUploadEmpty):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "FILE_REQUIRED"})
		case errors.Is(err, studentpaper.ErrStudentUploadTooMany):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "TOO_MANY_FILES"})
		case errors.Is(err, studentpaper.ErrStudentUploadPDFMultiple):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "PDF_REQUIRES_SINGLE_FILE"})
		case errors.Is(err, studentpaper.ErrStudentUploadImageSet):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_IMAGE_BATCH"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "PAPER_CREATE_FAILED"})
		}
		return
	}
	if h.audit != nil {
		if sid := middleware.StudentDBID(r.Context()); sid != 0 {
			pid := paper.ID
			snap, _ := json.Marshal(map[string]any{
				"subject": subject, "stage": stage, "files_n": len(parts), "label": paper.FileName,
			})
			h.audit.Write(r.Context(), auditlog.Event{
				UserID:     &sid,
				UserType:   "student",
				Action:     "create",
				EntityType: "exam_paper",
				EntityID:   &pid,
				Snapshot:   snap,
				IP:         r.RemoteAddr,
				CreatedBy:  0,
			})
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"paper": paper})
}

func (h *PaperHandler) List(w http.ResponseWriter, r *http.Request) {
	identifier := middleware.StudentIdentifier(r.Context())
	if identifier == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": h.service.List(identifier)})
}

func (h *PaperHandler) Analysis(w http.ResponseWriter, r *http.Request) {
	identifier := middleware.StudentIdentifier(r.Context())
	if identifier == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	paperID := r.PathValue("paperId")
	analysis, err := h.service.GetAnalysis(identifier, paperID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"analysis": analysis})
}

func (h *PaperHandler) Plan(w http.ResponseWriter, r *http.Request) {
	identifier := middleware.StudentIdentifier(r.Context())
	if identifier == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"code": "UNAUTHORIZED"})
		return
	}
	paperID := r.PathValue("paperId")
	plan, err := h.service.GetPlan(identifier, paperID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
		return
	}
	writeJSON(w, http.StatusOK, plan)
}
