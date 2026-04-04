package ailog

const maxStoredBodyBytes = 400 * 1024

// TruncateBody caps UTF-8 byte length stored in ai_call_log.
func TruncateBody(s string) string {
	if len(s) <= maxStoredBodyBytes {
		return s
	}
	return s[:maxStoredBodyBytes] + "\n…[truncated]"
}
