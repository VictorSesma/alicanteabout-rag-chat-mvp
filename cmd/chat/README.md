# cmd/chat

HTTP API entry point for the RAG chatbot.

Purpose
- Orchestrates config, DB logging, embeddings index, and HTTP server startup.

Inputs
- Chunks: ./out/alicanteabout_chunks.json (JSON or JSONL)
- Cache: ./out/embeddings_cache.json
- Env: OPENAI_API_KEY, CHAT_JWT_SECRET, optional CHAT_DB_DSN

Flow
- Load .env (best-effort).
- Read config from env + flags.
- Optionally open DB and run migrations.
- Load chunks + embeddings cache and build index.
- Start HTTP server with /chat and /healthz.

Tests
- go test ./cmd/chat
