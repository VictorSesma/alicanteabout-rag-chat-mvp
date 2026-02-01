package chat

import (
	"context"
	"time"

	"content-rag-chat/internal/rag"
	"content-rag-chat/internal/storage"
)

func (s *Server) logChat(ctx context.Context, question string, answerType string, results []rag.ScoredChunk, start time.Time) {
	if s.logger == nil {
		return
	}
	redacted := SanitizeQuestion(question)
	sources, scores := topSources(results, s.cfg.MaxSources)
	rec := storage.ChatLog{
		QuestionRedacted: redacted,
		QuestionHash:     HashQuestion(redacted),
		AnswerType:       answerType,
		TopSources:       sources,
		TopScores:        scores,
		LatencyMs:        int(time.Since(start).Milliseconds()),
	}
	s.logger.Log(ctx, rec)
}

func topSources(results []rag.ScoredChunk, max int) ([]string, []float32) {
	if max <= 0 {
		return nil, nil
	}
	limit := max
	if len(results) < limit {
		limit = len(results)
	}
	sources := make([]string, 0, limit)
	scores := make([]float32, 0, limit)
	for i := 0; i < limit; i++ {
		sources = append(sources, results[i].Chunk.URL)
		scores = append(scores, results[i].Score)
	}
	return sources, scores
}
