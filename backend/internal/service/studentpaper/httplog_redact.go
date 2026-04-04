package studentpaper

import (
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
