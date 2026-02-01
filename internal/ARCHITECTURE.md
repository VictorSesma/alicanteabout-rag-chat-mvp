# internal/ARCHITECTURE

Packages

internal/config
- LoadDotEnv: loads key=value from .env if present, without overwriting existing env.

internal/rag
- Chunk loading: JSON array (RawChunk) or JSONL (Chunk) formats.
- Embeddings: OpenAI embeddings API only (provider=openai).
- Cache: embeddings_cache.json keyed by chunk_id; includes model metadata.
- Search: cosine similarity over normalized vectors; TopK results.
- Prompt: BuildPrompt for CLI usage.

internal/chat
- HTTP server: /chat + /healthz, CORS, rate limiting, JWT auth.
- Language gate: English-only heuristic.
- Embeddings: question embeddings cached in-memory (LRU).
- Retrieval: TopK search + MIN_SCORE gate.
- Generation: OpenAI chat completions, JSON-only output.
- Streaming: SSE "delta" and "result" events.
- Logging: sanitized + hashed questions and top sources/scores.

internal/storage
- Async logger writes chat logs to Postgres.
- Migration: 001_create_chat_logs.sql (schema for chat_logs).

Request flow (/chat)
- JWT auth -> rate limit -> parse request -> language gate.
- Embed question -> search index -> score gate.
- Generate answer (streaming or non-streaming).
- Log sanitized question and top sources.
