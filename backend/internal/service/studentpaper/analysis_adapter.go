package studentpaper

// VisionImage is one page sent as an OpenAI-style image_url part (raw bytes; adapter base64-encodes).
type VisionImage struct {
	MIME string
	Data []byte
}

type AnalyzeInput struct {
	Subject   string
	Stage     string
	FileName  string // label for prompts/logging (may summarize multiple uploads)
	// VisionImages is optional; order preserved (e.g. exam pages). Empty => text-only or mock-AI protocol.
	VisionImages []VisionImage
	// ChatUserPrompt is the final user message for chat/completions (from prompt_template + placeholders).
	ChatUserPrompt string
}

type AnalyzeOutput struct {
	ModelSnapshot   map[string]any
	Summary         string
	WeakPoints      []string
	ImprovementPlan []string
	RawContent      string
}

// AnalyzeTrace describes how the adapter produced AnalyzeOutput (for ai_call_log).
type AnalyzeTrace struct {
	AdapterKind    string
	ResultStatus   string // success | mock_only | fallback_mock
	HTTPStatus     int
	LatencyMS      int64
	ErrorPhase     string
	ErrorMessage   string
	EndpointHost   string
	ChatModel      string
	FallbackToMock bool
	RequestBody    string // e.g. outbound JSON (image base64 redacted in http adapter)
	ResponseBody   string // upstream response body when available
}

type AnalyzeResult struct {
	Out   AnalyzeOutput
	Trace AnalyzeTrace
}

type AnalysisAdapter interface {
	Analyze(input AnalyzeInput) AnalyzeResult
}

type MockAnalysisAdapter struct{}

func (m MockAnalysisAdapter) Analyze(input AnalyzeInput) AnalyzeResult {
	return AnalyzeResult{
		Out: AnalyzeOutput{
			ModelSnapshot: map[string]any{
				"name": "mock-model-v0.1",
				"url":  "https://mock-ai.local/analyze",
			},
			Summary:         "本次试卷基础题稳定，综合题存在建模与条件提取问题。",
			WeakPoints:      []string{"受力分析", "图像题条件提取", "单位换算"},
			ImprovementPlan: []string{"D1-D2: 力学基础与受力分析专项", "D3-D4: 图像题分层训练", "D5-D6: 限时综合训练", "D7: 错题复盘与回测"},
			RawContent:      "mock-o-cr-content",
		},
		Trace: AnalyzeTrace{
			AdapterKind:  "mock_builtin",
			ResultStatus: "mock_only",
		},
	}
}
