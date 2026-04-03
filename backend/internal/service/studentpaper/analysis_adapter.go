package studentpaper

type AnalyzeInput struct {
	Subject  string
	Stage    string
	FileName string
}

type AnalyzeOutput struct {
	ModelSnapshot   map[string]any
	Summary         string
	WeakPoints      []string
	ImprovementPlan []string
	RawContent      string
}

type AnalysisAdapter interface {
	Analyze(input AnalyzeInput) AnalyzeOutput
}

type MockAnalysisAdapter struct{}

func (m MockAnalysisAdapter) Analyze(input AnalyzeInput) AnalyzeOutput {
	return AnalyzeOutput{
		ModelSnapshot: map[string]any{
			"name": "mock-model-v0.1",
			"url":  "https://mock-ai.local/analyze",
		},
		Summary:         "本次试卷基础题稳定，综合题存在建模与条件提取问题。",
		WeakPoints:      []string{"受力分析", "图像题条件提取", "单位换算"},
		ImprovementPlan: []string{"D1-D2: 力学基础与受力分析专项", "D3-D4: 图像题分层训练", "D5-D6: 限时综合训练", "D7: 错题复盘与回测"},
		RawContent:      "mock-o-cr-content",
	}
}
