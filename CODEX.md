# CODEX

Notes for future Codex sessions. This file is not user-facing.

## Project Intent

- Production-grade RAG backend for alicanteabout.com (tourism guide).
- Must answer using only site content and always provide sources.
- If not supported by content, reply: "I don't know based on AlicanteAbout content."
- English-only (lang must be "en").

## Docs Map (per folder)

- cmd/ARCHITECTURE.md, cmd/DESIGN.md, cmd/AGENTS.md
- internal/ARCHITECTURE.md, internal/DESIGN.md, internal/AGENTS.md
- out/ARCHITECTURE.md, out/DESIGN.md, out/AGENTS.md
- wordpress/ARCHITECTURE.md, wordpress/DESIGN.md, wordpress/AGENTS.md

## Entry Points

- `cmd/export` fetches WordPress content and writes to `out/`.
- `cmd/search` is an interactive CLI for retrieval (embeddings + cosine).
- `cmd/chat` is the HTTP API (`/chat`, `/healthz`).

## Languages by Folder

- `cmd/` (Go)
- `internal/` (Go)
- `wordpress/alicanteabout-chat-token/` (PHP)
- `wordpress/alicanteabout-chat-widget/` (PHP + JS + CSS)

## Shared RAG Logic

- `internal/rag` contains chunk loading, embeddings cache, index build, and search.
- Cache file: `out/embeddings_cache.json` (model-specific).
- Chunk file: `out/alicanteabout_chunks.json` (or JSONL).

## API Contract

Request:

```json
{ "question": "string", "lang": "en" }
```

Response:

```json
{
  "answer": "string",
  "sources": [
    { "title": "string", "url": "string" }
  ]
}
```

## Runtime Defaults

- Models: embeddings `text-embedding-3-small`, chat `gpt-4o-mini`.
- Retrieval: `TOP_K=5`, `MAX_SOURCES=3`, `MIN_SCORE=0.25`.
- CORS: `https://alicanteabout.com`.
- Rate limiting: 30 req/min per IP.
- JWT auth: HS256 with `CHAT_JWT_SECRET`, issuer/audience defaults in `internal/chat/config.go`.

## Tests

- `go test ./cmd/chat` (handler and middleware tests)
- `go test ./...`
- PHP plugin tests: `phpunit` from `wordpress/alicanteabout-chat-token/`

## Tooling Notes

- Go tools are installed via Snap in this environment (`/snap/bin/go`, `/snap/bin/gofmt`).
- Running `gofmt` or `go test` may require elevated permissions due to Snap confinement.

## WordPress Chat Widget Notes

- The widget appears only when the page `<html lang>` matches allowed languages (default `en`).
- Settings are in WP Admin → Settings → Chat Widget.

## Legal Implications

Let's keep in mind GDPR rules when building this project and let's not store identifiable personal data
