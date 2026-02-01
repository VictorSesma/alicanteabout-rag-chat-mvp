package chat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateJWTValid(t *testing.T) {
	cfg := Config{
		JWTSecret:   "test-secret",
		JWTIssuer:   "alicanteabout.com",
		JWTAudience: "alicanteabout-chat",
		JWTLeeway:   1 * time.Second,
	}
	now := time.Now().Unix()
	token := buildTestJWT(cfg.JWTSecret, map[string]any{
		"iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience,
		"iat": now,
		"exp": now + 60,
	})

	if err := validateJWT(token, cfg); err != nil {
		t.Fatalf("expected valid token, got %v", err)
	}
}

func TestValidateJWTExpired(t *testing.T) {
	cfg := Config{
		JWTSecret: "test-secret",
		JWTLeeway: 0,
	}
	now := time.Now().Unix()
	token := buildTestJWT(cfg.JWTSecret, map[string]any{
		"exp": now - 10,
	})

	if err := validateJWT(token, cfg); err != errJWTExpired {
		t.Fatalf("expected errJWTExpired, got %v", err)
	}
}

func TestWithJWTAuthMissing(t *testing.T) {
	cfg := Config{JWTSecret: "test-secret"}
	handler := withJWTAuth(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestWithJWTAuthValid(t *testing.T) {
	cfg := Config{JWTSecret: "test-secret"}
	now := time.Now().Unix()
	token := buildTestJWT(cfg.JWTSecret, map[string]any{
		"iat": now,
		"exp": now + 60,
	})
	handler := withJWTAuth(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func buildTestJWT(secret string, payload map[string]any) string {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	segments := []string{
		base64.RawURLEncoding.EncodeToString(headerJSON),
		base64.RawURLEncoding.EncodeToString(payloadJSON),
	}
	signingInput := segments[0] + "." + segments[1]
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return signingInput + "." + signature
}
