package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type analyzeRequest struct {
	Subject  string `json:"subject"`
	Stage    string `json:"stage"`
	FileName string `json:"file_name"`
}

func main() {
	port := getenv("MOCK_AI_PORT", "8090")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("POST /analyze", handleAnalyze)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("mock-ai listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_JSON"})
		return
	}

	summary := "本次试卷基础题表现稳定，综合题在条件提取与建模上有提升空间。"
	if req.Subject == "语文" {
		summary = "本次试卷阅读理解与作文结构需加强，基础题整体较稳。"
	}

	resp := map[string]any{
		"model_name": "local-mock-ai",
		"model_url":  "http://mock-ai:8090/analyze",
		"summary":    summary,
		"weak_points": []string{
			"关键条件提取",
			"解题结构化表达",
			"细节审题与复核",
		},
		"improvement_plan": []string{
			"D1-D2: 基础题型分组训练",
			"D3-D4: 中档题限时训练",
			"D5-D6: 综合题拆解与复盘",
			"D7: 全量错题回看与再测",
		},
		"raw_content": "mock raw content for " + req.FileName,
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
