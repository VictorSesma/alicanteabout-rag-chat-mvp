package chat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	errJWTMissing       = errors.New("missing token")
	errJWTInvalid       = errors.New("invalid token")
	errJWTExpired       = errors.New("token expired")
	errJWTNotYetValid   = errors.New("token not yet valid")
	errJWTIssuer        = errors.New("invalid issuer")
	errJWTAudience      = errors.New("invalid audience")
	errJWTAlg           = errors.New("invalid algorithm")
	errJWTNotConfigured = errors.New("jwt not configured")
)

func withJWTAuth(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.JWTSecret == "" {
			http.Error(w, "auth not configured", http.StatusInternalServerError)
			return
		}
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "missing auth token", http.StatusUnauthorized)
			return
		}
		if err := validateJWT(token, cfg); err != nil {
			http.Error(w, "invalid auth token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func validateJWT(token string, cfg Config) error {
	if cfg.JWTSecret == "" {
		return errJWTNotConfigured
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errJWTInvalid
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return errJWTInvalid
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return errJWTInvalid
	}
	if header.Alg != "HS256" {
		return errJWTAlg
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return errJWTInvalid
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return errJWTInvalid
	}
	signingInput := parts[0] + "." + parts[1]
	expectedSig := signJWT(signingInput, cfg.JWTSecret)
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return errJWTInvalid
	}
	if !hmac.Equal(gotSig, expectedSig) {
		return errJWTInvalid
	}

	now := time.Now().Unix()
	leeway := int64(cfg.JWTLeeway.Seconds())
	if expRaw, ok := payload["exp"]; ok {
		exp, ok := numberToInt64(expRaw)
		if !ok {
			return errJWTInvalid
		}
		if now > exp+leeway {
			return errJWTExpired
		}
	}
	if iatRaw, ok := payload["iat"]; ok {
		iat, ok := numberToInt64(iatRaw)
		if !ok {
			return errJWTInvalid
		}
		if iat > now+leeway {
			return errJWTNotYetValid
		}
	}
	if cfg.JWTIssuer != "" {
		if iss, _ := payload["iss"].(string); iss != cfg.JWTIssuer {
			return errJWTIssuer
		}
	}
	if cfg.JWTAudience != "" {
		if !audienceMatches(payload["aud"], cfg.JWTAudience) {
			return errJWTAudience
		}
	}
	return nil
}

func BuildJWT(secret, issuer, audience string, now time.Time, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errJWTNotConfigured
	}
	if ttl <= 0 {
		return "", errors.New("invalid ttl")
	}
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	issuedAt := now.Unix()
	payload := map[string]any{
		"iss": issuer,
		"aud": audience,
		"iat": issuedAt,
		"exp": issuedAt + int64(ttl.Seconds()),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	segments := []string{
		base64.RawURLEncoding.EncodeToString(headerJSON),
		base64.RawURLEncoding.EncodeToString(payloadJSON),
	}
	signingInput := segments[0] + "." + segments[1]
	signature := base64.RawURLEncoding.EncodeToString(signJWT(signingInput, secret))
	return signingInput + "." + signature, nil
}

func signJWT(input, secret string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(input))
	return h.Sum(nil)
}

func numberToInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		out, err := n.Int64()
		return out, err == nil
	default:
		return 0, false
	}
}

func audienceMatches(raw any, expected string) bool {
	switch v := raw.(type) {
	case string:
		return v == expected
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
		return false
	default:
		return false
	}
}
