package studentpaper

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/raywong-bitscube/stepup/backend/internal/service/prompttemplate"
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

func isDashScopeLikeHost(host string) bool {
	h := strings.ToLower(host)
	return strings.Contains(h, "dashscope") || strings.Contains(h, "aliyuncs.com")
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
	trace.RequestBody = redactLogJSON(string(payload))

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
	respBodyBytes, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		trace.ErrorPhase = "read_body"
		trace.ErrorMessage = truncateRunes(rerr.Error(), 400)
		trace.ResponseBody = string(respBodyBytes)
		return a.mockFallback(input, trace)
	}
	trace.ResponseBody = string(respBodyBytes)

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
	if err := json.Unmarshal(respBodyBytes, &parsed); err != nil {
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

	userPrompt := strings.TrimSpace(input.ChatUserPrompt)
	if userPrompt == "" {
		userPrompt = prompttemplate.Expand(prompttemplate.DefaultPaperAnalyzeChatUser(), map[string]string{
			"subject":   input.Subject,
			"stage":     input.Stage,
			"file_name": input.FileName,
		})
	}

	var userMsg map[string]any
	if len(input.VisionImages) > 0 {
		content := make([]any, 0, len(input.VisionImages)+1)
		for _, im := range input.VisionImages {
			mime := strings.ToLower(strings.TrimSpace(im.MIME))
			if !strings.HasPrefix(mime, "image/") || len(im.Data) == 0 {
				continue
			}
			dataURL := fmt.Sprintf("data:%s;base64,%s", im.MIME, base64.StdEncoding.EncodeToString(im.Data))
			content = append(content, map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url": dataURL,
				},
			})
		}
		if len(content) == 0 {
			userMsg = map[string]any{"role": "user", "content": userPrompt}
		} else {
			content = append(content, map[string]any{"type": "text", "text": userPrompt})
			userMsg = map[string]any{"role": "user", "content": content}
		}
	} else {
		userMsg = map[string]any{
			"role":    "user",
			"content": userPrompt,
		}
	}

	reqBody := map[string]any{
		"model":    model,
		"messages": []any{userMsg},
	}
	if input.OptionalMaxOutputTokens > 0 {
		reqBody["max_tokens"] = input.OptionalMaxOutputTokens
	}
	// DashScope Qwen3.5+ may emit huge reasoning_content; prefer non-thinking for latency and logs.
	if isDashScopeLikeHost(host) {
		reqBody["enable_thinking"] = false
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		trace.LatencyMS = time.Since(start).Milliseconds()
		trace.ErrorPhase = "marshal"
		trace.ErrorMessage = truncateRunes(err.Error(), 400)
		return a.mockFallback(input, trace)
	}
	trace.RequestBody = redactLogJSON(string(payload))

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
	respBodyBytes, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		trace.ErrorPhase = "read_body"
		trace.ErrorMessage = truncateRunes(rerr.Error(), 400)
		trace.ResponseBody = string(respBodyBytes)
		return a.mockFallback(input, trace)
	}
	trace.ResponseBody = string(redactChatCompletionResponseForLog(respBodyBytes))

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
	if err := json.Unmarshal(respBodyBytes, &openAI); err != nil {
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
