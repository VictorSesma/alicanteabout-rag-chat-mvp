package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"content-rag-chat/internal/chat"
	appconfig "content-rag-chat/internal/config"
	"content-rag-chat/internal/rag"
	"content-rag-chat/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := appconfig.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	cfg := chat.LoadConfigFromEnv()
	chat.BindFlags(&cfg)
	flag.Parse()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if cfg.Provider == "openai" && apiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("CHAT_JWT_SECRET is not set")
	}

	var logger storage.Logger
	db, err := openChatDB()
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if db != nil {
		if envBool("RUN_MIGRATIONS", true) {
			if err := runMigrations(db); err != nil {
				log.Fatalf("run migrations: %v", err)
			}
		}
		if !cfg.DisableLogging {
			async := storage.NewAsyncLogger(db, storage.AsyncLoggerConfig{
				Buffer:      cfg.LogBuffer,
				BatchSize:   cfg.LogBatchSize,
				FlushEvery:  cfg.LogFlushEvery,
				ReportEvery: cfg.LogReportEvery,
			})
			async.Start(context.Background())
			logger = async
		} else {
			_ = db.Close()
		}
	}

	chunks, err := rag.ReadChunks(cfg.ChunksPath)
	if err != nil {
		log.Fatalf("load chunks: %v", err)
	}
	cache, err := rag.LoadCache(cfg.CachePath)
	if err != nil {
		log.Fatalf("load cache: %v", err)
	}
	if cache.Model != "" && cache.Model != cfg.EmbedModel {
		log.Printf("warning: cache model is %q but embed model is %q; only matching items will be used", cache.Model, cfg.EmbedModel)
	}

	entries := rag.BuildIndex(chunks, cache, cfg.EmbedModel)
	if len(entries) == 0 {
		log.Fatal("no embeddings loaded; ensure cache exists and matches embed model")
	}

	srv := chat.NewServer(cfg, entries, &http.Client{Timeout: cfg.Timeout}, logger)
	mux := chat.NewMux(srv)

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}

func openChatDB() (*sql.DB, error) {
	dsn := os.Getenv("CHAT_DB_DSN")
	if dsn == "" {
		return nil, nil
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func runMigrations(db *sql.DB) error {
	migrationsDir := envString("MIGRATIONS_DIR", "internal/storage/migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}
	return nil
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "y", "on":
			return true
		case "0", "false", "no", "n", "off":
			return false
		}
	}
	return def
}
