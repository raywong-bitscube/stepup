package studentpaper

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// maxVisionImageBytes caps the image sent to chat-completions (base64 expands ~33%).
const maxVisionImageBytes = 10 << 20

func uniqueStoredFileName(original string) string {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin"
	}
	var rnd [4]byte
	_, _ = rand.Read(rnd[:])
	return fmt.Sprintf("%s_%s%s", slugOriginalFileStem(original), hex.EncodeToString(rnd[:]), ext)
}

func slugOriginalFileStem(original string) string {
	base := strings.TrimSuffix(filepath.Base(original), filepath.Ext(original))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "upload"
	}
	const maxStem = 80
	if len(s) > maxStem {
		s = s[:maxStem]
	}
	return s
}

// visionImageForModel returns non-empty mime + bytes when the upload should be passed as an OpenAI-style image_url part.
func visionImageForModel(fileName, declaredMIME string, data []byte) (mimeType string, image []byte) {
	if len(data) == 0 {
		return "", nil
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".pdf" {
		return "", nil
	}

	mt := ""
	if parsed, _, err := mime.ParseMediaType(declaredMIME); err == nil && parsed != "" {
		mt = strings.ToLower(parsed)
	}
	if mt == "" || mt == "application/octet-stream" {
		sniff := http.DetectContentType(data)
		if p, _, err := mime.ParseMediaType(sniff); err == nil && p != "" {
			mt = strings.ToLower(p)
		} else {
			mt = strings.TrimSpace(strings.ToLower(strings.Split(sniff, ";")[0]))
		}
	}

	if !strings.HasPrefix(mt, "image/") {
		switch ext {
		case ".jpg", ".jpeg":
			mt = "image/jpeg"
		case ".png":
			mt = "image/png"
		case ".gif":
			mt = "image/gif"
		case ".webp":
			mt = "image/webp"
		default:
			return "", nil
		}
	}

	if len(data) > maxVisionImageBytes {
		return "", nil
	}
	return mt, data
}
