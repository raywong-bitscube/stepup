package studentpaper

import (
	"encoding/json"
	"regexp"
	"strconv"
)

var dataImageBase64RE = regexp.MustCompile(`data:image/[^;]+;base64,[A-Za-z0-9+/=]+`)

// redactLogJSON replaces inline data-URL image payloads so AI logs stay readable and smaller.
func redactLogJSON(s string) string {
	return dataImageBase64RE.ReplaceAllStringFunc(s, func(m string) string {
		return "[image base64 omitted, " + strconv.Itoa(len(m)) + " chars]"
	})
}

// redactChatCompletionResponseForLog returns a compact copy of an OpenAI-style chat completion
// JSON with long model-only fields removed so ai_call_log stays readable (e.g. Qwen
// reasoning_content / thinking traces).
func redactChatCompletionResponseForLog(raw []byte) []byte {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return raw
	}
	if choices, ok := root["choices"].([]any); ok {
		for _, ch := range choices {
			cm, ok := ch.(map[string]any)
			if !ok {
				continue
			}
			msg, ok := cm["message"].(map[string]any)
			if !ok {
				continue
			}
			if s, has := msg["reasoning_content"]; has {
				n := 0
				if str, ok := s.(string); ok {
					n = len(str)
				}
				msg["reasoning_content"] = "[omitted " + strconv.Itoa(n) + " chars]"
			}
		}
	}
	out, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return out
}
