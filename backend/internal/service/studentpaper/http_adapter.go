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
	endpoint    string
	timeout     time.Duration
	client      *http.Client
	bearerToken string
	chatModel   string // OpenAI-compatible model id (e.g. deepseek-chat); used when bearerToken is set
}

// NewHTTPAnalysisAdapter posts to endpoint. If bearerToken is non-empty, the adapter uses the
// OpenAI-compatible chat completions protocol (DeepSeek, OpenAI, etc.). Otherwise it uses the
// small StepUp mock-ai JSON shape: {subject, stage, file_name}.
func NewHTTPAnalysisAdapter(endpoint string, timeout time.Duration, bearerToken, chatModel string) *HTTPAnalysisAdapter {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPAnalysisAdapter{
		endpoint:    strings.TrimSpace(endpoint),
		timeout:     timeout,
		client:      &http.Client{Timeout: timeout},
		bearerToken: strings.TrimSpace(bearerToken),
		chatModel:   strings.TrimSpace(chatModel),
	}
}

func (a *HTTPAnalysisAdapter) Analyze(input AnalyzeInput) AnalyzeOutput {
	if a.endpoint == "" {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	if a.bearerToken != "" {
		return a.analyzeChatCompletions(input)
	}
	return a.analyzeMockAIProtocol(input)
}

func (a *HTTPAnalysisAdapter) analyzeMockAIProtocol(input AnalyzeInput) AnalyzeOutput {
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

func (a *HTTPAnalysisAdapter) analyzeChatCompletions(input AnalyzeInput) AnalyzeOutput {
	model := a.chatModel
	if model == "" {
		if strings.Contains(strings.ToLower(a.endpoint), "deepseek") {
			model = "deepseek-chat"
		} else {
			model = "gpt-4o-mini"
		}
	}

	userPrompt := fmt.Sprintf(
		`试卷上传元信息：科目=%s，阶段=%s，原始文件名=%s。
请只输出一段合法 JSON（不要用 markdown 代码围栏），严格符合下列键：summary (string)、weak_points (string 数组)、improvement_plan (string 数组)、raw_content (string，可为试卷要点摘录或空字符串)。
内容针对中国学生试卷分析场景，用语简洁专业。`,
		input.Subject, input.Stage, input.FileName,
	)

	reqBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.bearerToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	var openAI struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openAI); err != nil {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	if len(openAI.Choices) == 0 {
		return MockAnalysisAdapter{}.Analyze(input)
	}
	content := strings.TrimSpace(openAI.Choices[0].Message.Content)
	if content == "" {
		return MockAnalysisAdapter{}.Analyze(input)
	}

	out := parseModelJSONBlob(content)
	if out.Summary == "" {
		out.Summary = content
	}
	out.ModelSnapshot = map[string]any{
		"name": model,
		"url":  a.endpoint,
	}
	if out.RawContent == "" {
		out.RawContent = content
	}
	return out
}

type parsedAIFields struct {
	Summary         string   `json:"summary"`
	WeakPoints      []string `json:"weak_points"`
	ImprovementPlan []string `json:"improvement_plan"`
	RawContent      string   `json:"raw_content"`
}

func parseModelJSONBlob(raw string) AnalyzeOutput {
	s := stripCodeFence(raw)
	var fields parsedAIFields
	if err := json.Unmarshal([]byte(s), &fields); err != nil {
		return AnalyzeOutput{}
	}
	return AnalyzeOutput{
		Summary:         strings.TrimSpace(fields.Summary),
		WeakPoints:      fields.WeakPoints,
		ImprovementPlan: fields.ImprovementPlan,
		RawContent:      fields.RawContent,
	}
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := strings.TrimPrefix(s, "```")
	rest = strings.TrimPrefix(rest, "json")
	rest = strings.TrimSpace(rest)
	if idx := strings.LastIndex(rest, "```"); idx >= 0 {
		rest = rest[:idx]
	}
	return strings.TrimSpace(rest)
}

func coalesce(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (a *HTTPAnalysisAdapter) String() string {
	return fmt.Sprintf("http-adapter(endpoint=%s,bearer=%v)", a.endpoint, a.bearerToken != "")
}
