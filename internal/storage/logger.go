package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type ChatLog struct {
	QuestionRedacted string
	QuestionHash     string
	AnswerType       string
	TopSources       []string
	TopScores        []float32
	LatencyMs        int
}

type Logger interface {
	Log(ctx context.Context, record ChatLog)
}

type AsyncLoggerConfig struct {
	Buffer      int
	BatchSize   int
	FlushEvery  time.Duration
	ReportEvery time.Duration
}

func DefaultAsyncLoggerConfig() AsyncLoggerConfig {
	return AsyncLoggerConfig{
		Buffer:      1000,
		BatchSize:   100,
		FlushEvery:  500 * time.Millisecond,
		ReportEvery: 30 * time.Second,
	}
}

type AsyncLogger struct {
	db      *sql.DB
	cfg     AsyncLoggerConfig
	ch      chan ChatLog
	dropped uint64
	started uint32
}

func NewAsyncLogger(db *sql.DB, cfg AsyncLoggerConfig) *AsyncLogger {
	if cfg.Buffer <= 0 || cfg.BatchSize <= 0 || cfg.FlushEvery <= 0 {
		cfg = DefaultAsyncLoggerConfig()
	}
	return &AsyncLogger{
		db:  db,
		cfg: cfg,
		ch:  make(chan ChatLog, cfg.Buffer),
	}
}

func (l *AsyncLogger) Start(ctx context.Context) {
	if !atomic.CompareAndSwapUint32(&l.started, 0, 1) {
		return
	}
	go l.loop(ctx)
}

func (l *AsyncLogger) Log(_ context.Context, record ChatLog) {
	select {
	case l.ch <- record:
	default:
		atomic.AddUint64(&l.dropped, 1)
	}
}

func (l *AsyncLogger) Dropped() uint64 {
	return atomic.LoadUint64(&l.dropped)
}

func (l *AsyncLogger) loop(ctx context.Context) {
	ticker := time.NewTicker(l.cfg.FlushEvery)
	defer ticker.Stop()
	reporter := time.NewTicker(l.cfg.ReportEvery)
	defer reporter.Stop()
	var lastDropped uint64

	batch := make([]ChatLog, 0, l.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		_ = l.insertBatch(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case rec := <-l.ch:
			batch = append(batch, rec)
			if len(batch) >= l.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-reporter.C:
			dropped := l.Dropped()
			if dropped != lastDropped {
				lastDropped = dropped
				logDropped(dropped)
			}
		}
	}
}

func (l *AsyncLogger) insertBatch(ctx context.Context, records []ChatLog) error {
	if len(records) == 0 {
		return nil
	}
	query, args := buildInsert(records)
	if query == "" {
		return errors.New("empty insert query")
	}
	_, err := l.db.ExecContext(ctx, query, args...)
	return err
}

func buildInsert(records []ChatLog) (string, []any) {
	values := make([]string, 0, len(records))
	args := make([]any, 0, len(records)*6)
	for i, rec := range records {
		base := i*6 + 1
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)", base, base+1, base+2, base+3, base+4, base+5))
		args = append(args, rec.QuestionRedacted, rec.QuestionHash, rec.AnswerType, rec.TopSources, rec.TopScores, rec.LatencyMs)
	}
	query := "INSERT INTO chat_logs (question_redacted, question_hash, answer_type, top_sources, top_scores, latency_ms) VALUES " + join(values, ",")
	return query, args
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	n += len(sep) * (len(parts) - 1)
	out := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, p...)
	}
	return string(out)
}

func logDropped(dropped uint64) {
	if dropped == 0 {
		return
	}
	// TODO: hook into structured logger if/when added.
	// Using fmt here to avoid pulling log into this package.
	fmt.Printf("chat_logger dropped=%d\n", dropped)
}
