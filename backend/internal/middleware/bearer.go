package middleware

import "strings"

// BearerToken extracts the token from an Authorization: Bearer <token> header value.
func BearerToken(authorization string) string {
	raw := strings.TrimSpace(authorization)
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
