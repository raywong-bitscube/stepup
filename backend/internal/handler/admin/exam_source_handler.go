package admin

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/raywong-bitscube/stepup/backend/internal/middleware"
	"github.com/raywong-bitscube/stepup/backend/internal/service/adminexamsource"
)

type ExamSourceHandler struct {
	svc *adminexamsource.Service
}

type examSourcePageRow struct {
	ID        uint64  `json:"id"`
	PaperID   uint64  `json:"paper_id"`
	PageNo    int     `json:"page_no"`
	FileID    uint64  `json:"file_id"`
	PublicURL *string `json:"public_url"`
	Status    int     `json:"status"`
}

type examSourceQuestionRow struct {
	ID            uint64      `json:"id"`
	PaperID       uint64      `json:"paper_id"`
	QuestionNo    string      `json:"question_no"`
	QuestionOrder int         `json:"question_order"`
	SectionNo     *string     `json:"section_no"`
	QuestionType  string      `json:"question_type"`
	Score         *string     `json:"score"`
	StemText      *string     `json:"stem_text"`
	AnswerText    *string     `json:"answer_text"`
	Explanation   *string     `json:"explanation_text"`
	PageFrom      *int        `json:"page_from"`
	PageTo        *int        `json:"page_to"`
	Status        int         `json:"status"`
	UpdatedAt     RFC3339Time `json:"updated_at"`
}

func toExamSourcePageRows(items []adminexamsource.Page) []examSourcePageRow {
	out := make([]examSourcePageRow, 0, len(items))
	for _, p := range items {
		out = append(out, examSourcePageRow{
			ID:        p.ID,
			PaperID:   p.PaperID,
			PageNo:    p.PageNo,
			FileID:    p.FileID,
			PublicURL: p.PublicURL,
			Status:    p.Status,
		})
	}
	return out
}

func toExamSourceQuestionRows(items []adminexamsource.Question) []examSourceQuestionRow {
	out := make([]examSourceQuestionRow, 0, len(items))
	for _, q := range items {
		out = append(out, examSourceQuestionRow{
			ID:            q.ID,
			PaperID:       q.PaperID,
			QuestionNo:    q.QuestionNo,
			QuestionOrder: q.QuestionOrder,
			SectionNo:     q.SectionNo,
			QuestionType:  q.QuestionType,
			Score:         q.Score,
			StemText:      q.StemText,
			AnswerText:    q.AnswerText,
			Explanation:   q.Explanation,
			PageFrom:      q.PageFrom,
			PageTo:        q.PageTo,
			Status:        q.Status,
			UpdatedAt:     RFC3339Time(q.UpdatedAt),
		})
	}
	return out
}

func NewExamSourceHandler(svc *adminexamsource.Service) *ExamSourceHandler {
	return &ExamSourceHandler{svc: svc}
}

func (h *ExamSourceHandler) ListPapers(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListPapers(r.Context())
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	type row struct {
		ID              uint64      `json:"id"`
		PaperCode       *string     `json:"paper_code"`
		Title           string      `json:"title"`
		SourceRegion    *string     `json:"source_region"`
		SourceSchool    *string     `json:"source_school"`
		ExamYear        *int        `json:"exam_year"`
		Term            *string     `json:"term"`
		GradeLabel      *string     `json:"grade_label"`
		K12GradeID      *uint64     `json:"k12_grade_id"`
		K12SubjectID    uint64      `json:"k12_subject_id"`
		PaperType       string      `json:"paper_type"`
		TotalScore      *string     `json:"total_score"`
		DurationMinutes *int        `json:"duration_minutes"`
		PageCount       int         `json:"page_count"`
		QuestionCount   int         `json:"question_count"`
		Remarks         *string     `json:"remarks"`
		Status          int         `json:"status"`
		UpdatedAt       RFC3339Time `json:"updated_at"`
		CreatedAt       RFC3339Time `json:"created_at"`
	}
	out := make([]row, 0, len(items))
	for _, it := range items {
		out = append(out, row{
			ID:              it.ID,
			PaperCode:       it.PaperCode,
			Title:           it.Title,
			SourceRegion:    it.SourceRegion,
			SourceSchool:    it.SourceSchool,
			ExamYear:        it.ExamYear,
			Term:            it.Term,
			GradeLabel:      it.GradeLabel,
			K12GradeID:      it.K12GradeID,
			K12SubjectID:    it.K12SubjectID,
			PaperType:       it.PaperType,
			TotalScore:      it.TotalScore,
			DurationMinutes: it.DurationMinutes,
			PageCount:       it.PageCount,
			QuestionCount:   it.QuestionCount,
			Remarks:         it.Remarks,
			Status:          it.Status,
			UpdatedAt:       RFC3339Time(it.UpdatedAt),
			CreatedAt:       RFC3339Time(it.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type createPaperBody struct {
	PaperCode       string  `json:"paper_code"`
	Title           string  `json:"title"`
	SourceRegion    string  `json:"source_region"`
	SourceSchool    string  `json:"source_school"`
	ExamYear        *int    `json:"exam_year"`
	Term            string  `json:"term"`
	GradeLabel      string  `json:"grade_label"`
	K12GradeID      *uint64 `json:"k12_grade_id"`
	K12SubjectID    uint64  `json:"k12_subject_id"`
	PaperType       string  `json:"paper_type"`
	TotalScore      *string `json:"total_score"`
	DurationMinutes *int    `json:"duration_minutes"`
	Remarks         string  `json:"remarks"`
	Status          int     `json:"status"`
}

func adminIDFromReq(r *http.Request) uint64 {
	if sess, ok := middleware.AdminSession(r.Context()); ok {
		return sess.AdminID
	}
	return 0
}

func (h *ExamSourceHandler) CreatePaper(w http.ResponseWriter, r *http.Request) {
	var req createPaperBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	nid, err := h.svc.CreatePaper(r.Context(), adminIDFromReq(r), adminexamsource.CreatePaperInput{
		PaperCode:       req.PaperCode,
		Title:           req.Title,
		SourceRegion:    req.SourceRegion,
		SourceSchool:    req.SourceSchool,
		ExamYear:        req.ExamYear,
		Term:            req.Term,
		GradeLabel:      req.GradeLabel,
		K12GradeID:      req.K12GradeID,
		K12SubjectID:    req.K12SubjectID,
		PaperType:       req.PaperType,
		TotalScore:      req.TotalScore,
		DurationMinutes: req.DurationMinutes,
		Remarks:         req.Remarks,
		Status:          req.Status,
	})
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "id": nid})
	}
}

func (h *ExamSourceHandler) GetRecognitionPreview(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUint64(r.PathValue("paperId"))
	if !ok || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	pages, qs, err := h.svc.GetRecognitionPreview(r.Context(), id)
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		items := make([]map[string]any, 0, len(qs))
		for _, q := range qs {
			row := map[string]any{
				"id":              q.ID,
				"paper_id":        q.PaperID,
				"question_no":     q.QuestionNo,
				"question_order":  q.QuestionOrder,
				"section_no":      q.SectionNo,
				"question_type":   q.QuestionType,
				"score":           q.Score,
				"stem_text":       q.StemText,
				"answer_text":     q.AnswerText,
				"explanation_text": q.Explanation,
				"page_from":       q.PageFrom,
				"page_to":         q.PageTo,
				"status":          q.Status,
				"updated_at":      RFC3339Time(q.UpdatedAt),
			}
			if q.StemQuestionFileID != nil {
				row["stem_question_file_id"] = *q.StemQuestionFileID
			} else {
				row["stem_question_file_id"] = nil
			}
			if q.StemFileID != nil {
				row["stem_file_id"] = *q.StemFileID
			} else {
				row["stem_file_id"] = nil
			}
			row["stem_crop_public_url"] = q.StemCropURL
			if q.StemPageNo != nil {
				row["stem_page_no"] = *q.StemPageNo
			} else {
				row["stem_page_no"] = nil
			}
			row["stem_bbox_norm"] = q.StemBBoxNorm
			items = append(items, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"pages":     toExamSourcePageRows(pages),
			"questions": items,
		})
	}
}

type patchStemBBoxBody struct {
	PageNo   int     `json:"page_no"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	H        float64 `json:"h"`
}

func (h *ExamSourceHandler) PatchQuestionStemBBox(w http.ResponseWriter, r *http.Request) {
	qid, ok := parsePathUint64(r.PathValue("questionId"))
	if !ok || qid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchStemBBoxBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	res, err := h.svc.PatchQuestionStemBBox(r.Context(), adminIDFromReq(r), qid, adminexamsource.PatchStemBBoxInput{
		PageNo: req.PageNo,
		X:      req.X,
		Y:      req.Y,
		W:      req.W,
		H:      req.H,
	})
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"status":              "ok",
			"crop_public_url":     res.CropPublicURL,
			"page_no":             res.PageNo,
			"bbox_norm":           res.BBoxNorm,
		})
	}
}

func (h *ExamSourceHandler) GetPaper(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUint64(r.PathValue("paperId"))
	if !ok || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	p, err := h.svc.GetPaper(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, adminexamsource.ErrNoDatabase):
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
		case errors.Is(err, adminexamsource.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		case errors.Is(err, adminexamsource.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		}
		return
	}
	pages, err := h.svc.ListPages(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	qs, err := h.svc.ListQuestions(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"paper": map[string]any{
			"id":               p.ID,
			"paper_code":       p.PaperCode,
			"title":            p.Title,
			"source_region":    p.SourceRegion,
			"source_school":    p.SourceSchool,
			"exam_year":        p.ExamYear,
			"term":             p.Term,
			"grade_label":      p.GradeLabel,
			"k12_grade_id":     p.K12GradeID,
			"k12_subject_id":   p.K12SubjectID,
			"paper_type":       p.PaperType,
			"total_score":      p.TotalScore,
			"duration_minutes": p.DurationMinutes,
			"page_count":       p.PageCount,
			"question_count":   p.QuestionCount,
			"remarks":          p.Remarks,
			"status":           p.Status,
			"updated_at":       RFC3339Time(p.UpdatedAt),
			"created_at":       RFC3339Time(p.CreatedAt),
		},
		"pages":     toExamSourcePageRows(pages),
		"questions": toExamSourceQuestionRows(qs),
	})
}

type patchPaperBody struct {
	PaperCode       *string `json:"paper_code"`
	Title           *string `json:"title"`
	SourceRegion    *string `json:"source_region"`
	SourceSchool    *string `json:"source_school"`
	ExamYear        *int    `json:"exam_year"`
	Term            *string `json:"term"`
	GradeLabel      *string `json:"grade_label"`
	K12GradeID      *uint64 `json:"k12_grade_id"`
	K12SubjectID    *uint64 `json:"k12_subject_id"`
	PaperType       *string `json:"paper_type"`
	TotalScore      *string `json:"total_score"`
	DurationMinutes *int    `json:"duration_minutes"`
	Remarks         *string `json:"remarks"`
	Status          *int    `json:"status"`
}

func (h *ExamSourceHandler) PatchPaper(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUint64(r.PathValue("paperId"))
	if !ok || id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchPaperBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err := h.svc.PatchPaper(r.Context(), id, adminIDFromReq(r), adminexamsource.PatchPaperInput{
		PaperCode:       req.PaperCode,
		Title:           req.Title,
		SourceRegion:    req.SourceRegion,
		SourceSchool:    req.SourceSchool,
		ExamYear:        req.ExamYear,
		Term:            req.Term,
		GradeLabel:      req.GradeLabel,
		K12GradeID:      req.K12GradeID,
		K12SubjectID:    req.K12SubjectID,
		PaperType:       req.PaperType,
		TotalScore:      req.TotalScore,
		DurationMinutes: req.DurationMinutes,
		Remarks:         req.Remarks,
		Status:          req.Status,
	})
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, adminexamsource.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func (h *ExamSourceHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	pid, ok := parsePathUint64(r.PathValue("paperId"))
	if !ok || pid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	qs, err := h.svc.ListQuestions(r.Context(), pid)
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"items": toExamSourceQuestionRows(qs)})
	}
}

type createQuestionBody struct {
	QuestionNo    string  `json:"question_no"`
	QuestionOrder int     `json:"question_order"`
	SectionNo     string  `json:"section_no"`
	QuestionType  string  `json:"question_type"`
	Score         *string `json:"score"`
	StemText      string  `json:"stem_text"`
	AnswerText    string  `json:"answer_text"`
	Explanation   string  `json:"explanation_text"`
	PageFrom      *int    `json:"page_from"`
	PageTo        *int    `json:"page_to"`
	Status        int     `json:"status"`
}

func (h *ExamSourceHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	pid, ok := parsePathUint64(r.PathValue("paperId"))
	if !ok || pid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req createQuestionBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	nid, err := h.svc.CreateQuestion(r.Context(), pid, adminIDFromReq(r), adminexamsource.CreateQuestionInput{
		QuestionNo:    req.QuestionNo,
		QuestionOrder: req.QuestionOrder,
		SectionNo:     req.SectionNo,
		QuestionType:  req.QuestionType,
		Score:         req.Score,
		StemText:      req.StemText,
		AnswerText:    req.AnswerText,
		Explanation:   req.Explanation,
		PageFrom:      req.PageFrom,
		PageTo:        req.PageTo,
		Status:        req.Status,
	})
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "id": nid})
	}
}

type patchQuestionBody struct {
	QuestionNo    *string `json:"question_no"`
	QuestionOrder *int    `json:"question_order"`
	SectionNo     *string `json:"section_no"`
	QuestionType  *string `json:"question_type"`
	Score         *string `json:"score"`
	StemText      *string `json:"stem_text"`
	AnswerText    *string `json:"answer_text"`
	Explanation   *string `json:"explanation_text"`
	PageFrom      *int    `json:"page_from"`
	PageTo        *int    `json:"page_to"`
	Status        *int    `json:"status"`
}

func (h *ExamSourceHandler) PatchQuestion(w http.ResponseWriter, r *http.Request) {
	qid, ok := parsePathUint64(r.PathValue("questionId"))
	if !ok || qid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	var req patchQuestionBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}
	err := h.svc.PatchQuestion(r.Context(), qid, adminIDFromReq(r), adminexamsource.PatchQuestionInput{
		QuestionNo:    req.QuestionNo,
		QuestionOrder: req.QuestionOrder,
		SectionNo:     req.SectionNo,
		QuestionType:  req.QuestionType,
		Score:         req.Score,
		StemText:      req.StemText,
		AnswerText:    req.AnswerText,
		Explanation:   req.Explanation,
		PageFrom:      req.PageFrom,
		PageTo:        req.PageTo,
		Status:        req.Status,
	})
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"code": "NOT_FOUND"})
	case errors.Is(err, adminexamsource.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	}
}

func parsePathUint64(raw string) (uint64, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseCSVValues(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseOptionalInt(raw string) (*int, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(t)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseOptionalUint64Form(raw string) (*uint64, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return nil, nil
	}
	v, err := strconv.ParseUint(t, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func readUploadImage(hdr *multipart.FileHeader) (adminexamsource.UploadImage, error) {
	f, err := hdr.Open()
	if err != nil {
		return adminexamsource.UploadImage{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return adminexamsource.UploadImage{}, err
	}
	return adminexamsource.UploadImage{
		Filename: hdr.Filename,
		Bytes:    b,
	}, nil
}

func (h *ExamSourceHandler) CreatePaperWithUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_MULTIPART"})
		return
	}
	form := r.MultipartForm
	if form == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_MULTIPART"})
		return
	}
	get := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	subID, err := strconv.ParseUint(get("k12_subject_id"), 10, 64)
	if err != nil || subID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	gradeID, err := parseOptionalUint64Form(get("k12_grade_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	examYear, err := parseOptionalInt(get("exam_year"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	duration, err := parseOptionalInt(get("duration_minutes"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
		return
	}
	status := 1
	if stRaw := get("status"); stRaw != "" {
		v, pe := strconv.Atoi(stRaw)
		if pe != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
			return
		}
		status = v
	}
	var totalScore *string
	if ts := get("total_score"); ts != "" {
		totalScore = &ts
	}
	fhs := form.File["images"]
	if len(fhs) == 0 {
		// fallback: support "files" field
		fhs = form.File["files"]
	}
	if len(fhs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "IMAGES_REQUIRED"})
		return
	}
	images := make([]adminexamsource.UploadImage, 0, len(fhs))
	for _, fh := range fhs {
		img, re := readUploadImage(fh)
		if re != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_FILE"})
			return
		}
		images = append(images, img)
	}
	req := adminexamsource.CreatePaperWithUploadInput{
		CreatePaperInput: adminexamsource.CreatePaperInput{
			PaperCode:       get("paper_code"),
			Title:           get("title"),
			SourceRegion:    get("source_region"),
			SourceSchool:    get("source_school"),
			ExamYear:        examYear,
			Term:            get("term"),
			GradeLabel:      get("grade_label"),
			K12GradeID:      gradeID,
			K12SubjectID:    subID,
			PaperType:       get("paper_type"),
			TotalScore:      totalScore,
			DurationMinutes: duration,
			Remarks:         get("remarks"),
			Status:          status,
		},
		QuestionNos: parseCSVValues(get("question_nos")),
	}
	nid, err := h.svc.CreatePaperWithImages(r.Context(), adminIDFromReq(r), req, images)
	switch {
	case errors.Is(err, adminexamsource.ErrNoDatabase):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"code": "DATABASE_REQUIRED"})
	case errors.Is(err, adminexamsource.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_INPUT"})
	case errors.Is(err, adminexamsource.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"code": "CONFLICT"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"code": "INTERNAL_ERROR"})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "id": nid})
	}
}
