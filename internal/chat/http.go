package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"content-rag-chat/internal/rag"
)

type chatRequest struct {
	Question string `json:"question"`
	Lang     string `json:"lang"`
}

type sourceItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type chatResponse struct {
	Answer  string       `json:"answer"`
	Sources []sourceItem `json:"sources"`
}

func NewMux(s *Server) *http.ServeMux {
	limiter := newRateLimiter(s.cfg.RateLimit, s.cfg.RateWindow)
	mux := http.NewServeMux()
	chatHandler := withRateLimit(limiter, withJWTAuth(s.cfg, http.HandlerFunc(s.handleChat)))
	mux.Handle("/chat", withCORS(s.cfg.CORSAllowedOrigin, chatHandler))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func writeSSEEvent(w http.ResponseWriter, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	return nil
}

func wantsStream(r *http.Request) bool {
	if r.URL.Query().Get("stream") == "1" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/event-stream")
}

func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRateLimit(l *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := rag.RequestID(ctx)
		if reqID == "" {
			reqID = newReqID()
			ctx = rag.WithRequestID(ctx, reqID)
			r = r.WithContext(ctx)
		}
		ip := clientIP(r)
		if ip == "" {
			log.Printf("req_id=%s rate_limit blocked ip=unknown reason=missing_ip", reqID)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		allowed, remaining, reset := l.AllowWithStatus(ip)
		resetIn := time.Until(reset).Round(time.Second)
		if !allowed {
			log.Printf("req_id=%s rate_limit blocked ip=%s reset_in=%s", reqID, ip, resetIn)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		log.Printf("req_id=%s rate_limit allowed ip=%s remaining=%d reset_in=%s", reqID, ip, remaining, resetIn)
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts := strings.Split(v, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return strings.TrimSpace(v)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
