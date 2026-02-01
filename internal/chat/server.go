package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"content-rag-chat/internal/rag"
	"content-rag-chat/internal/storage"
)

type Server struct {
	cfg        Config
	client     *http.Client
	entries    []rag.Entry
	embedCache *embedCache
	logger     storage.Logger

	embedFunc  func(ctx context.Context, question string) ([]float32, error)
	searchFunc func(entries []rag.Entry, q []float32, k int) []rag.ScoredChunk
	answerFunc func(ctx context.Context, question string, hits []rag.ScoredChunk) (string, []sourceItem, error)
	streamFunc func(ctx context.Context, question string, hits []rag.ScoredChunk, w http.ResponseWriter) (string, error)
}

const (
	fallbackAnswer = "I don't know based on AlicanteAbout content."
	langFallback   = "Sorry, English only for now."
)

func NewServer(cfg Config, entries []rag.Entry, client *http.Client, logger storage.Logger) *Server {
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	srv := &Server{
		cfg:     cfg,
		client:  client,
		entries: entries,
		logger:  logger,
	}
	if cfg.EmbedCacheMax > 0 {
		srv.embedCache = newEmbedCache(cfg.EmbedCacheMax)
	}
	return srv
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	body := http.MaxBytesReader(w, r.Body, 64*1024)
	defer body.Close()

	var req chatRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}
	if req.Lang == "" {
		req.Lang = "en"
	}
	if req.Lang != "en" {
		http.Error(w, "only English is supported", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	reqID := rag.RequestID(ctx)
	if reqID == "" {
		reqID = newReqID()
		ctx = rag.WithRequestID(ctx, reqID)
	}
	if !isEnglishQuestion(req.Question) {
		if wantsStream(r) {
			writeStreamFallback(w, langFallback)
		} else {
			writeJSON(w, chatResponse{
				Answer:  langFallback,
				Sources: nil,
			})
		}
		s.logChat(ctx, req.Question, "no_answer", nil, start)
		log.Printf("req_id=%s chat done=%s fallback=true reason=non_english", reqID, fmtDuration(time.Since(start)))
		return
	}
	log.Printf("req_id=%s chat start question_len=%d", reqID, len(req.Question))
	embed := s.embedFunc
	if embed == nil {
		embed = func(ctx context.Context, question string) ([]float32, error) {
			return rag.EmbedQuery(ctx, s.client, s.cfg.Provider, os.Getenv("OPENAI_API_KEY"), s.cfg.EmbedModel, question)
		}
	}
	tEmbed := time.Now()
	var qVec []float32
	var err error
	cacheKey := s.cfg.Provider + ":" + s.cfg.EmbedModel + ":" + req.Lang + ":" + req.Question
	if s.embedCache != nil {
		if v, ok := s.embedCache.Get(cacheKey); ok {
			qVec = v
			log.Printf("req_id=%s chat embed_cache_hit=true", reqID)
		}
	}
	if qVec == nil {
		qVec, err = embed(ctx, req.Question)
		if err != nil {
			http.Error(w, "embedding error", http.StatusInternalServerError)
			return
		}
		if s.embedCache != nil {
			s.embedCache.Add(cacheKey, qVec)
			log.Printf("req_id=%s chat embed_cache_hit=false", reqID)
		}
	}
	rag.Normalize(qVec)
	log.Printf("req_id=%s chat embed=%s", reqID, fmtDuration(time.Since(tEmbed)))

	search := s.searchFunc
	if search == nil {
		search = rag.TopKSearch
	}
	tSearch := time.Now()
	results := search(s.entries, qVec, s.cfg.TopK)
	log.Printf("req_id=%s chat search=%s results=%d top_score=%.4f", reqID, fmtDuration(time.Since(tSearch)), len(results), topScore(results))
	if len(results) == 0 || results[0].Score < s.cfg.MinScore {
		writeJSON(w, chatResponse{
			Answer:  fallbackAnswer,
			Sources: nil,
		})
		s.logChat(ctx, req.Question, "no_answer", results, start)
		log.Printf("req_id=%s chat done=%s fallback=true", reqID, fmtDuration(time.Since(start)))
		return
	}

	if wantsStream(r) {
		stream := s.streamFunc
		if stream == nil {
			stream = s.generateAnswerStream
		}
		answerType, err := stream(ctx, req.Question, results, w)
		if err != nil {
			http.Error(w, "streaming error", http.StatusInternalServerError)
			return
		}
		s.logChat(ctx, req.Question, answerType, results, start)
		log.Printf("req_id=%s chat done=%s streamed=true", reqID, fmtDuration(time.Since(start)))
		return
	}

	answerFn := s.answerFunc
	if answerFn == nil {
		answerFn = s.generateAnswer
	}
	tAnswer := time.Now()
	answer, sources, err := answerFn(ctx, req.Question, results)
	if err != nil {
		http.Error(w, "generation error", http.StatusInternalServerError)
		return
	}
	log.Printf("req_id=%s chat answer=%s sources=%d", reqID, fmtDuration(time.Since(tAnswer)), len(sources))

	writeJSON(w, chatResponse{
		Answer:  answer,
		Sources: sources,
	})
	answerType := "grounded"
	if isFallbackAnswer(answer, sources) {
		answerType = "no_answer"
	}
	s.logChat(ctx, req.Question, answerType, results, start)
	log.Printf("req_id=%s chat done=%s fallback=%t", reqID, fmtDuration(time.Since(start)), answerType == "no_answer")
}

func (s *Server) generateAnswer(ctx context.Context, question string, hits []rag.ScoredChunk) (string, []sourceItem, error) {
	prompt, ordered := buildPrompt(question, hits, s.cfg.TopK)
	req := chatCompletionRequest{
		Model: s.cfg.ChatModel,
		Messages: []chatMessage{
			{Role: "system", Content: "You must follow the instructions. Output JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		ResponseFormat: &chatResponseFormat{
			Type: "json_object",
		},
	}

	raw, err := s.callChatCompletion(ctx, req)
	if err != nil {
		return "", nil, err
	}

	var out struct {
		Answer  string       `json:"answer"`
		Sources []sourceItem `json:"sources"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return "", nil, fmt.Errorf("invalid model json: %w", err)
	}
	out.Answer = strings.TrimSpace(out.Answer)

	cleanSources := filterSources(ordered, out.Sources, s.cfg.MaxSources)
	if out.Answer == "" {
		out.Answer = fallbackAnswer
		cleanSources = nil
	}

	return out.Answer, cleanSources, nil
}

func (s *Server) generateAnswerStream(ctx context.Context, question string, hits []rag.ScoredChunk, w http.ResponseWriter) (string, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return "grounded", fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	prompt, ordered := buildPrompt(question, hits, s.cfg.TopK)
	req := chatCompletionRequest{
		Model: s.cfg.ChatModel,
		Messages: []chatMessage{
			{Role: "system", Content: "You must follow the instructions. Output JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		ResponseFormat: &chatResponseFormat{
			Type: "json_object",
		},
		Stream: true,
	}

	var full strings.Builder
	err := s.callChatCompletionStream(ctx, req, func(delta string) {
		if delta == "" {
			return
		}
		full.WriteString(delta)
		_ = writeSSEEvent(w, "delta", map[string]string{"delta": delta})
		flusher.Flush()
	})
	if err != nil {
		_ = writeSSEEvent(w, "error", map[string]string{"error": err.Error()})
		flusher.Flush()
		return "grounded", err
	}

	var out struct {
		Answer  string       `json:"answer"`
		Sources []sourceItem `json:"sources"`
	}
	if err := json.Unmarshal([]byte(full.String()), &out); err != nil {
		_ = writeSSEEvent(w, "error", map[string]string{"error": "invalid model json"})
		flusher.Flush()
		return "grounded", fmt.Errorf("invalid model json: %w", err)
	}
	out.Answer = strings.TrimSpace(out.Answer)
	cleanSources := filterSources(ordered, out.Sources, s.cfg.MaxSources)
	if out.Answer == "" {
		out.Answer = fallbackAnswer
		cleanSources = nil
	}

	if err := writeSSEEvent(w, "result", chatResponse{
		Answer:  out.Answer,
		Sources: cleanSources,
	}); err != nil {
		return "grounded", err
	}
	if isFallbackAnswer(out.Answer, cleanSources) {
		return "no_answer", nil
	}
	return "grounded", nil
}

func isFallbackAnswer(answer string, sources []sourceItem) bool {
	return strings.TrimSpace(answer) == fallbackAnswer && len(sources) == 0
}

func writeStreamFallback(w http.ResponseWriter, answer string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, chatResponse{Answer: answer})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_ = writeSSEEvent(w, "result", chatResponse{
		Answer:  answer,
		Sources: nil,
	})
	flusher.Flush()
}
