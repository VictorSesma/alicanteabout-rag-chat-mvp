package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestBuildToken(t *testing.T) {
	cfg := tokenConfig{
		Secret:   "test-secret",
		Issuer:   "alicanteabout.com",
		Audience: "alicanteabout-chat",
		TTL:      2 * time.Minute,
	}
	now := time.Unix(1700000000, 0)
	token, err := buildToken(cfg, now)
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	parts := splitToken(token)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	payload := map[string]any{}
	if err := json.Unmarshal(mustDecode(parts[1]), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["iss"] != cfg.Issuer {
		t.Fatalf("unexpected iss: %v", payload["iss"])
	}
	if payload["aud"] != cfg.Audience {
		t.Fatalf("unexpected aud: %v", payload["aud"])
	}
	if payload["iat"] != float64(now.Unix()) {
		t.Fatalf("unexpected iat: %v", payload["iat"])
	}
	if payload["exp"] != float64(now.Add(cfg.TTL).Unix()) {
		t.Fatalf("unexpected exp: %v", payload["exp"])
	}
}

func TestEnvDurationUsesEnv(t *testing.T) {
	t.Setenv("CHAT_JWT_TTL", "10m")
	got := envDuration("CHAT_JWT_TTL", 2*time.Minute)
	if got != 10*time.Minute {
		t.Fatalf("expected 10m, got %s", got)
	}
}

func splitToken(token string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	return append(parts, token[start:])
}

func mustDecode(raw string) []byte {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		panic(err)
	}
	return data
}
