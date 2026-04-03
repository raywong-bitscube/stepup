package studentpaper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HTTPAnalysisAdapter struct {
	endpoint string
	timeout  time.Duration
	client   *http.Client
}

func NewHTTPAnalysisAdapter(endpoint string, timeout time.Duration) *HTTPAnalysisAdapter {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPAnalysisAdapter{
		endpoint: strings.TrimSpace(endpoint),
		timeout:  timeout,
		client:   &http.Client{Timeout: timeout},
	}
}

func (a *HTTPAnalysisAdapter) Analyze(input AnalyzeInput) AnalyzeOutput {
	if a.endpoint == "" {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	reqBody := map[string]any{
		"subject":   input.Subject,
		"stage":     input.Stage,
		"file_name": input.FileName,
	}
	payload, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	var parsed struct {
		ModelName       string   `json:"model_name"`
		ModelURL        string   `json:"model_url"`
		Summary         string   `json:"summary"`
		WeakPoints      []string `json:"weak_points"`
		ImprovementPlan []string `json:"improvement_plan"`
		RawContent      string   `json:"raw_content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	if parsed.Summary == "" {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	return AnalyzeOutput{
		ModelSnapshot: map[string]any{
			"name": coalesce(parsed.ModelName, "http-adapter-model"),
			"url":  coalesce(parsed.ModelURL, a.endpoint),
		},
		Summary:         parsed.Summary,
		WeakPoints:      parsed.WeakPoints,
		ImprovementPlan: parsed.ImprovementPlan,
		RawContent:      parsed.RawContent,
	}
}

func coalesce(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (a *HTTPAnalysisAdapter) String() string {
	return fmt.Sprintf("http-adapter(endpoint=%s)", a.endpoint)
}
