package chat

import (
	"strings"
	"testing"
)

func TestSanitizeQuestion(t *testing.T) {
	input := "Email me at test.user+1@example.com or call +1 (555) 123-4567. See https://example.com."
	got := SanitizeQuestion(input)
	if got == input {
		t.Fatal("expected redaction")
	}
	for _, needle := range []string{"[redacted_email]", "[redacted_phone]", "[redacted_url]"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("missing %s in %s", needle, got)
		}
	}
}
