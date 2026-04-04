package studentpaper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type HTTPAnalysisAdapter struct {
	endpoint    string
	timeout     time.Duration
	client      *http.Client
	bearerToken string
	chatModel   string
}

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

func endpointHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…"
}

func (a *HTTPAnalysisAdapter) mockFallback(input AnalyzeInput, trace AnalyzeTrace) AnalyzeResult {
	m := MockAnalysisAdapter{}.Analyze(input)
	if trace.ResultStatus == "" {
		trace.ResultStatus = "fallback_mock"
	}
	trace.FallbackToMock = true
	return AnalyzeResult{Out: m.Out, Trace: trace}
}

func (a *HTTPAnalysisAdapter) Analyze(input AnalyzeInput) AnalyzeResult {
	host := endpointHost(a.endpoint)
	if a.endpoint == "" {
		m := MockAnalysisAdapter{}.Analyze(input)
		m.Trace = AnalyzeTrace{
			AdapterKind:  "http_unconfigured",
			ResultStatus: "mock_only",
			ErrorPhase:   "config",
			ErrorMessage: "empty endpoint URL",
		}
		m.Out = m.Out
		return AnalyzeResult{Out: m.Out, Trace: m.Trace}
	}
	if a.bearerToken != "" {
		return a.analyzeChatCompletions(input, host)
	}
	return a.analyzeMockAIProtocol(input, host)
}

func (a *HTTPAnalysisAdapter) analyzeMockAIProtocol(input AnalyzeInput, host string) AnalyzeResult {
	trace := AnalyzeTrace{
		AdapterKind:  "http_mock_ai_protocol",
		EndpointHost: host,
	}
	start := time.Now()

	reqBody := map[string]any{
		"subject":   input.Subject,
		"stage":     input.Stage,
		"file_name": input.FileName,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		trace.LatencyMS = time.Since(start).Milliseconds()
		trace.ErrorPhase = "marshal"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		trace.LatencyMS = time.Since(start).Milliseconds()
		trace.ErrorPhase = "request_build"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	trace.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		trace.ErrorPhase = errorPhaseFromErr(err)
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	defer resp.Body.Close()

	trace.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 400 {
		trace.ErrorPhase = "http_status"
		trace.ErrorMessage = truncateRunes(fmt.Sprintf("HTTP %d", resp.StatusCode), 400)
		return a.mockFallback(input, trace)
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
		trace.ErrorPhase = "decode"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}

	if parsed.Summary == "" {
		trace.ErrorPhase = "empty_summary"
		trace.ErrorMessage = "upstream returned empty summary"
		return a.mockFallback(input, trace)
	}

	trace.ResultStatus = "success"
	return AnalyzeResult{
		Out: AnalyzeOutput{
			ModelSnapshot: map[string]any{
				"name": coalesce(parsed.ModelName, "http-adapter-model"),
				"url":  coalesce(parsed.ModelURL, a.endpoint),
			},
			Summary:         parsed.Summary,
			WeakPoints:      parsed.WeakPoints,
			ImprovementPlan: parsed.ImprovementPlan,
			RawContent:      parsed.RawContent,
		},
		Trace: trace,
	}
}

func (a *HTTPAnalysisAdapter) analyzeChatCompletions(input AnalyzeInput, host string) AnalyzeResult {
	model := a.chatModel
	if model == "" {
		if strings.Contains(strings.ToLower(a.endpoint), "deepseek") {
			model = "deepseek-chat"
		} else {
			model = "gpt-4o-mini"
		}
	}

	trace := AnalyzeTrace{
		AdapterKind:  "http_chat_completions",
		EndpointHost: host,
		ChatModel:    model,
	}
	start := time.Now()

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
		trace.LatencyMS = time.Since(start).Milliseconds()
		trace.ErrorPhase = "marshal"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		trace.LatencyMS = time.Since(start).Milliseconds()
		trace.ErrorPhase = "request_build"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.bearerToken)

	resp, err := a.client.Do(req)
	trace.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		trace.ErrorPhase = errorPhaseFromErr(err)
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	defer resp.Body.Close()

	trace.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= 400 {
		trace.ErrorPhase = "http_status"
		trace.ErrorMessage = truncateRunes(fmt.Sprintf("HTTP %d", resp.StatusCode), 400)
		return a.mockFallback(input, trace)
	}

	var openAI struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openAI); err != nil {
		trace.ErrorPhase = "decode"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	if len(openAI.Choices) == 0 {
		trace.ErrorPhase = "empty_choices"
		trace.ErrorMessage = "no choices in completion response"
		return a.mockFallback(input, trace)
	}
	content := strings.TrimSpace(openAI.Choices[0].Message.Content)
	if content == "" {
		trace.ErrorPhase = "empty_body"
		trace.ErrorMessage = "empty assistant content"
		return a.mockFallback(input, trace)
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
	trace.ResultStatus = "success"
	return AnalyzeResult{Out: out, Trace: trace}
}

func errorPhaseFromErr(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return "timeout"
	}
	return "network"
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
