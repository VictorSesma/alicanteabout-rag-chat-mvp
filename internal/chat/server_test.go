package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"content-rag-chat/internal/rag"
)

func TestWithCORSPreflight(t *testing.T) {
	h := withCORS("https://alicanteabout.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for preflight")
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://example.com/chat", nil)
	req.Header.Set("Origin", "https://alicanteabout.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://alicanteabout.com" {
		t.Fatalf("missing allow-origin header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("expected allow-headers to include Authorization")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)
	h := withRateLimit(limiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "http://example.com/chat", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec2.Code)
	}
}

func TestHandleChatValidation(t *testing.T) {
	srv := &Server{
		cfg: Config{
			TopK:     3,
			MinScore: 0.1,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/chat", nil)
	srv.handleChat(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "http://example.com/chat", bytes.NewBufferString("{bad"))
	srv.handleChat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "http://example.com/chat", bytes.NewBufferString(`{"question":"hi","lang":"es"}`))
	srv.handleChat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleChatNonEnglishFallbackJSON(t *testing.T) {
	srv := &Server{
		cfg: Config{
			TopK:     3,
			MinScore: 0.1,
		},
	}

	cases := []struct {
		name     string
		question string
	}{
		{name: "spanish", question: "Hola, como llego al aeropuerto?"},
		{name: "french", question: "Bonjour, comment aller a la plage?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			body := `{"question":"` + tc.question + `","lang":"en"}`
			req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", bytes.NewBufferString(body))
			srv.handleChat(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			var out chatResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Answer != langFallback {
				t.Fatalf("expected language fallback message")
			}
			if out.Sources != nil {
				t.Fatalf("expected no sources")
			}
		})
	}
}

func TestHandleChatNonEnglishFallbackStream(t *testing.T) {
	srv := &Server{
		cfg: Config{
			TopK:     3,
			MinScore: 0.1,
		},
	}

	cases := []struct {
		name     string
		question string
	}{
		{name: "spanish", question: "Hola, como llego al aeropuerto?"},
		{name: "french", question: "Bonjour, comment aller a la plage?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			body := `{"question":"` + tc.question + `","lang":"en"}`
			req := httptest.NewRequest(http.MethodPost, "http://example.com/chat?stream=1", bytes.NewBufferString(body))
			req.Header.Set("Accept", "text/event-stream")
			srv.handleChat(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			out := parseSSEData(t, rec.Body.String())
			if out.Answer != langFallback {
				t.Fatalf("expected language fallback message")
			}
			if out.Sources != nil {
				t.Fatalf("expected no sources")
			}
		})
	}
}

func TestHandleChatLowScore(t *testing.T) {
	srv := &Server{
		cfg: Config{
			TopK:     3,
			MinScore: 0.5,
		},
		embedFunc: func(ctx context.Context, question string) ([]float32, error) {
			return []float32{1, 0, 0}, nil
		},
		searchFunc: func(entries []rag.Entry, q []float32, k int) []rag.ScoredChunk {
			return []rag.ScoredChunk{
				{Chunk: rag.Chunk{Title: "A", URL: "https://a"}, Score: 0.1},
			}
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", bytes.NewBufferString(`{"question":"hi","lang":"en"}`))
	srv.handleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var out chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Answer == "" || out.Answer == " " {
		t.Fatalf("expected answer to be set")
	}
	if len(out.Sources) != 0 {
		t.Fatalf("expected no sources")
	}
}

func TestHandleChatSuccess(t *testing.T) {
	srv := &Server{
		cfg: Config{
			TopK:     3,
			MinScore: 0.1,
		},
		embedFunc: func(ctx context.Context, question string) ([]float32, error) {
			return []float32{1, 0, 0}, nil
		},
		searchFunc: func(entries []rag.Entry, q []float32, k int) []rag.ScoredChunk {
			return []rag.ScoredChunk{
				{Chunk: rag.Chunk{Title: "Post A", URL: "https://a"}, Score: 0.9},
			}
		},
		answerFunc: func(ctx context.Context, question string, hits []rag.ScoredChunk) (string, []sourceItem, error) {
			return "Answer", []sourceItem{{Title: "Post A", URL: "https://a"}}, nil
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/chat", bytes.NewBufferString(`{"question":"hi","lang":"en"}`))
	srv.handleChat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var out chatResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Answer != "Answer" {
		t.Fatalf("unexpected answer: %q", out.Answer)
	}
	if len(out.Sources) != 1 || out.Sources[0].URL != "https://a" {
		t.Fatalf("unexpected sources")
	}
}

func parseSSEData(t *testing.T, body string) chatResponse {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var out chatResponse
		if err := json.Unmarshal([]byte(payload), &out); err != nil {
			t.Fatalf("decode sse data: %v", err)
		}
		return out
	}
	t.Fatalf("missing sse data")
	return chatResponse{}
}

func TestEmbedCache(t *testing.T) {
	c := newEmbedCache(2)
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}
	c.Add("a", a)
	c.Add("b", b)

	if v, ok := c.Get("a"); !ok || len(v) != len(a) {
		t.Fatalf("expected cache hit for a")
	}

	c.Add("c", []float32{7})
	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected b to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected a to remain")
	}

	v, _ := c.Get("a")
	v[0] = 999
	v2, _ := c.Get("a")
	if v2[0] == 999 {
		t.Fatalf("expected cached vector to be copied")
	}
}
