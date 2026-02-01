package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	emailPattern = regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.-]+`)
	phonePattern = regexp.MustCompile(`\+?\d[\d\s\-().]{7,}`)
	urlPattern   = regexp.MustCompile(`https?://\S+|www\.\S+`)
)

func SanitizeQuestion(input string) string {
	out := strings.TrimSpace(input)
	out = emailPattern.ReplaceAllString(out, "[redacted_email]")
	out = phonePattern.ReplaceAllString(out, "[redacted_phone]")
	out = urlPattern.ReplaceAllString(out, "[redacted_url]")
	out = strings.TrimSpace(out)
	return out
}

func HashQuestion(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
