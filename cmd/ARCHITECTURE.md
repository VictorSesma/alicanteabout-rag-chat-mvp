# cmd/ARCHITECTURE

Entry points and how they compose with internal packages.

cmd/export
- Inputs: WordPress REST API (posts + pages).
- Outputs: ./out/alicanteabout_corpus.json and ./out/docs/*.txt.
- Purpose: create a clean text corpus for later chunking/embedding.

cmd/search
- Inputs: chunk file (JSON array or JSONL) and embeddings cache JSON.
- Outputs: updates embeddings cache JSON (when missing/outdated).
- Purpose: interactive retrieval, plus prompt preview for manual checks.

cmd/chat
- Inputs: chunk file + embeddings cache JSON; optional Postgres DSN for logging.
- Flow: load config -> connect DB -> run migrations -> load chunks/cache -> build index -> serve HTTP.
- HTTP: /chat (POST) and /healthz (GET).

cmd/chat-token
- Inputs: JWT secret + issuer/audience/ttl.
- Output: JWT printed to stdout (plus expiry lines).

Shared dependencies
- internal/config: .env loader (best-effort).
- internal/rag: chunk loading, embeddings, search index.
- internal/chat: HTTP server, auth, rate limit, OpenAI calls.
- internal/storage: async DB logger.
