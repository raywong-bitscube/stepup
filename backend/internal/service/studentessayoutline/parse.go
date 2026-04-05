package studentessayoutline

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reStarMatch     = regexp.MustCompile(`匹配度[：:．.]?\s*(\d)\s*(?:星|颗)`)
	reStarStructure = regexp.MustCompile(`结构[合理]?[性]?[：:．.]?\s*(\d)\s*(?:星|颗)`)
	reStarMaterial  = regexp.MustCompile(`素材[：:．.]?\s*(\d)\s*(?:星|颗)`)
)

func clampStar(n int) int {
	if n < 1 {
		return 1
	}
	if n > 5 {
		return 5
	}
	return n
}

// ParseTopicFromAI splits "题目 | 标签" from model output.
func ParseTopicFromAI(raw string, genre, taskType string) (topic, label string) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "`\"'")
	if s == "" {
		return "", ""
	}
	parts := strings.SplitN(s, "|", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(genre + " · " + taskType)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func splitReviewParts(s string) (summary, scores, detail string) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"'")
	if s == "" {
		return "", "", ""
	}
	idx0 := strings.Index(s, "|")
	if idx0 < 0 {
		return s, "", ""
	}
	rest := s[idx0+1:]
	idx1 := strings.Index(rest, "|")
	if idx1 < 0 {
		return strings.TrimSpace(s[:idx0]), strings.TrimSpace(rest), ""
	}
	return strings.TrimSpace(s[:idx0]), strings.TrimSpace(rest[:idx1]), strings.TrimSpace(rest[idx1+1:])
}

func parseStarsLine(line string) (match, structure, material int) {
	match, structure, material = 3, 3, 3
	if m := reStarMatch.FindStringSubmatch(line); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			match = clampStar(v)
		}
	}
	if m := reStarStructure.FindStringSubmatch(line); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			structure = clampStar(v)
		}
	}
	if m := reStarMaterial.FindStringSubmatch(line); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			material = clampStar(v)
		}
	}
	return match, structure, material
}

func parseSuggestionBullets(detail string) []string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return nil
	}
	for _, sep := range []string{"详细建议：", "详细建议:", "建议：", "建议:"} {
		if strings.HasPrefix(detail, sep) {
			detail = strings.TrimSpace(strings.TrimPrefix(detail, sep))
			break
		}
	}
	var parts []string
	for _, chunk := range strings.FieldsFunc(detail, func(r rune) bool {
		return r == '；' || r == ';' || r == '\n'
	}) {
		t := strings.TrimSpace(chunk)
		if t == "" {
			continue
		}
		// trim leading "1." "2、" 
		t = regexp.MustCompile(`^\d+[\.\、．]\s*`).ReplaceAllString(t, "")
		if t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 && detail != "" {
		return []string{detail}
	}
	return parts
}

// BuildReviewJSON returns a JSON-serializable map for DB and API.
func BuildReviewJSON(rawAI string) map[string]any {
	summary, scoresLine, detail := splitReviewParts(rawAI)
	m, st, mat := parseStarsLine(scoresLine)
	sugs := parseSuggestionBullets(detail)
	highlights := make([]string, 0, 2)
	for i := range sugs {
		if i < 2 {
			highlights = append(highlights, sugs[i])
		}
	}
	return map[string]any{
		"summary": strings.TrimSpace(summary),
		"stars": map[string]any{
			"match":     m,
			"structure": st,
			"material":  mat,
		},
		"suggestions": sugs,
		"highlights":  highlights,
	}
}
