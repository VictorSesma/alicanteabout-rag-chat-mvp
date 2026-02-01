package chat

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr              string
	ChunksPath        string
	CachePath         string
	Provider          string
	EmbedModel        string
	ChatModel         string
	TopK              int
	MaxSources        int
	MinScore          float32
	CORSAllowedOrigin string
	RateLimit         int
	RateWindow        time.Duration
	Timeout           time.Duration
	EmbedCacheMax     int
	JWTSecret         string
	JWTIssuer         string
	JWTAudience       string
	JWTLeeway         time.Duration
	LogBuffer         int
	LogBatchSize      int
	LogFlushEvery     time.Duration
	LogReportEvery    time.Duration
	DisableLogging    bool
}

func DefaultConfig() Config {
	return Config{
		Addr:              ":8080",
		ChunksPath:        "./out/alicanteabout_chunks.json",
		CachePath:         "./out/embeddings_cache.json",
		Provider:          "openai",
		EmbedModel:        "text-embedding-3-small",
		ChatModel:         "gpt-4o-mini",
		TopK:              3,
		MaxSources:        2,
		MinScore:          0.25,
		CORSAllowedOrigin: envString("CORS_ALLOWED_ORIGIN", "https://alicanteabout.com"),
		RateLimit:         30,
		RateWindow:        1 * time.Minute,
		Timeout:           30 * time.Second,
		EmbedCacheMax:     256,
		JWTIssuer:         "alicanteabout.com",
		JWTAudience:       "alicanteabout-chat",
		JWTLeeway:         10 * time.Second,
		LogBuffer:         1000,
		LogBatchSize:      100,
		LogFlushEvery:     500 * time.Millisecond,
		LogReportEvery:    30 * time.Second,
	}
}

func LoadConfigFromEnv() Config {
	def := DefaultConfig()
	return Config{
		Addr:              envString("ADDR", def.Addr),
		ChunksPath:        envString("CHUNKS_PATH", def.ChunksPath),
		CachePath:         envString("CACHE_PATH", def.CachePath),
		Provider:          envString("EMBED_PROVIDER", def.Provider),
		EmbedModel:        envString("EMBED_MODEL", def.EmbedModel),
		ChatModel:         envString("CHAT_MODEL", def.ChatModel),
		TopK:              envInt("TOP_K", def.TopK),
		MaxSources:        envInt("MAX_SOURCES", def.MaxSources),
		MinScore:          envFloat32("MIN_SCORE", def.MinScore),
		CORSAllowedOrigin: envString("CORS_ALLOWED_ORIGIN", def.CORSAllowedOrigin),
		RateLimit:         envInt("RATE_LIMIT", def.RateLimit),
		RateWindow:        envDuration("RATE_WINDOW", def.RateWindow),
		Timeout:           envDuration("TIMEOUT", def.Timeout),
		EmbedCacheMax:     envInt("EMBED_CACHE_MAX", def.EmbedCacheMax),
		JWTSecret:         envString("CHAT_JWT_SECRET", def.JWTSecret),
		JWTIssuer:         envString("CHAT_JWT_ISSUER", def.JWTIssuer),
		JWTAudience:       envString("CHAT_JWT_AUDIENCE", def.JWTAudience),
		JWTLeeway:         envDuration("CHAT_JWT_LEEWAY", def.JWTLeeway),
		LogBuffer:         envInt("CHAT_LOG_BUFFER", def.LogBuffer),
		LogBatchSize:      envInt("CHAT_LOG_BATCH_SIZE", def.LogBatchSize),
		LogFlushEvery:     envDuration("CHAT_LOG_FLUSH_EVERY", def.LogFlushEvery),
		LogReportEvery:    envDuration("CHAT_LOG_REPORT_EVERY", def.LogReportEvery),
		DisableLogging:    envBool("CHAT_LOG_DISABLE", def.DisableLogging),
	}
}

func BindFlags(cfg *Config) {
	flag.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listen address")
	flag.StringVar(&cfg.ChunksPath, "chunks", cfg.ChunksPath, "Path to chunks JSON or JSONL")
	flag.StringVar(&cfg.CachePath, "cache", cfg.CachePath, "Path to embeddings cache JSON")
	flag.StringVar(&cfg.Provider, "provider", cfg.Provider, "Embeddings provider: openai")
	flag.StringVar(&cfg.EmbedModel, "embed-model", cfg.EmbedModel, "Embeddings model")
	flag.StringVar(&cfg.ChatModel, "chat-model", cfg.ChatModel, "Chat model")
	flag.IntVar(&cfg.TopK, "k", cfg.TopK, "Top K chunks to retrieve")
	flag.IntVar(&cfg.MaxSources, "max-sources", cfg.MaxSources, "Max sources to return")
	flag.Var(float32Value{v: &cfg.MinScore}, "min-score", "Min cosine score to answer")
	flag.StringVar(&cfg.CORSAllowedOrigin, "cors-origin", cfg.CORSAllowedOrigin, "Allowed CORS origin")
	flag.IntVar(&cfg.RateLimit, "rate", cfg.RateLimit, "Requests per window per IP")
	flag.DurationVar(&cfg.RateWindow, "window", cfg.RateWindow, "Rate limit window")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "HTTP timeout for OpenAI calls")
	flag.IntVar(&cfg.EmbedCacheMax, "embed-cache-max", cfg.EmbedCacheMax, "Max question embedding cache entries (in-memory)")
	flag.StringVar(&cfg.JWTSecret, "jwt-secret", cfg.JWTSecret, "JWT secret for chat auth")
	flag.StringVar(&cfg.JWTIssuer, "jwt-issuer", cfg.JWTIssuer, "JWT issuer")
	flag.StringVar(&cfg.JWTAudience, "jwt-audience", cfg.JWTAudience, "JWT audience")
	flag.DurationVar(&cfg.JWTLeeway, "jwt-leeway", cfg.JWTLeeway, "JWT leeway for exp/iat checks")
	flag.IntVar(&cfg.LogBuffer, "log-buffer", cfg.LogBuffer, "Chat log buffer size")
	flag.IntVar(&cfg.LogBatchSize, "log-batch", cfg.LogBatchSize, "Chat log batch size")
	flag.DurationVar(&cfg.LogFlushEvery, "log-flush", cfg.LogFlushEvery, "Chat log flush interval")
	flag.DurationVar(&cfg.LogReportEvery, "log-report", cfg.LogReportEvery, "Chat log report interval")
	flag.BoolVar(&cfg.DisableLogging, "log-disable", cfg.DisableLogging, "Disable chat logging")
}

type float32Value struct {
	v *float32
}

func (f float32Value) String() string {
	if f.v == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*f.v), 'f', -1, 32)
}

func (f float32Value) Set(value string) error {
	n, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return err
	}
	*f.v = float32(n)
	return nil
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func envFloat32(key string, def float32) float32 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseFloat(v, 32)
		if err == nil {
			return float32(n)
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
