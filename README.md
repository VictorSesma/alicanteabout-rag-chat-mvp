# Content RAG Chat — Production RAG Backend for alicanteabout.com

Production-grade RAG backend for [alicanteabout.com](https://alicanteabout.com) (Alicante, Spain tourism guide). The system is backend-first and serves a lightweight JS widget on WordPress. Answers must be grounded in site content and include sources.

Read about the experience building this at my blog post [victorsesma.com](https://victorsesma.com/blog/grounded-rag-chatbot-wordpress-real-mvp/).

## Scope & Non-Goals

This repository is intentionally built for a single production website (alicanteabout.com).
Several defaults and environment variables are hardcoded to reflect that reality.

Non-goals:
- Multi-tenant support
- Automatic onboarding for other sites
- Generic SaaS abstractions
- Multilingual RAG (English only by design)

The goal of this project is to document and demonstrate a production-grade RAG
architecture applied to a real content-heavy site.

## Docs Map (per folder)

- cmd/ARCHITECTURE.md, cmd/DESIGN.md, cmd/AGENTS.md
- internal/ARCHITECTURE.md, internal/DESIGN.md, internal/AGENTS.md
- out/ARCHITECTURE.md, out/DESIGN.md, out/AGENTS.md
- wordpress/ARCHITECTURE.md, wordpress/DESIGN.md, wordpress/AGENTS.md
- 
## Design principles

This project is guided by a few core principles:

- **Grounding over fluency**  
  Wrong answers are worse than no answers.

- **Explicit fallbacks**  
  If the content does not support an answer, the system must say so.

- **Simplicity over abstraction**  
  No vector database or heavy infrastructure unless it is clearly needed.

- **Backend-first design**  
  WordPress acts only as a UI and token issuer.

- **Cost and predictability**  
  All expensive work is done offline whenever possible.

## Project Structure

```
cmd/
  export/  - WordPress content exporter
  search/  - CLI RAG search with embeddings
  chat/    - HTTP API for the chatbot
  chat-token/ - CLI for minting dev JWTs
internal/
  rag/     - Shared RAG helpers (chunks, embeddings, search)
wordpress/
  alicanteabout-chat-token/ - WordPress plugin that issues short-lived JWTs
  alicanteabout-chat-widget/ - WordPress chat widget (lazy-loaded modal)
bin/       - Compiled binaries
out/       - Output data (corpus, chunks, embeddings cache)
```

## How It Works

- Content is exported from WordPress, cleaned, chunked, and embedded.
- Embeddings are cached locally (`out/embeddings_cache.json`).
- Retrieval uses in-memory cosine similarity (no vector DB yet).
- The `/chat` API embeds the question, runs top-K search, gates on relevance, and then calls a chat model with retrieved sources.
- If not supported by content, the answer is: "I don't know based on AlicanteAbout content."

## Usage

### Export WordPress content

```bash
go run ./cmd/export -base https://alicanteabout.com -out ./out
```

### RAG Search

```bash
go run ./cmd/search -chunks ./out/alicanteabout_chunks.jsonl -cache ./out/embeddings_cache.json
```

### RAG Chat API

```bash
go run ./cmd/chat -chunks ./out/alicanteabout_chunks.json -cache ./out/embeddings_cache.json
```

Request:

```bash
curl -X POST http://localhost:8080/chat \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"question":"How do I get from the airport to the city center?","lang":"en"}'
```

Postman-importable curl (same request):

```bash
curl --location "http://localhost:8080/chat" \
  --header "Authorization: Bearer <jwt>" \
  --header "Content-Type: application/json" \
  --data-raw '{"question":"How do I get from the airport to the city center?","lang":"en"}'
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

Streaming (SSE):

```bash
curl -N "http://localhost:8080/chat?stream=1" \
  -H "Accept: text/event-stream" \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"question":"How do I get from the airport to the city center?","lang":"en"}'
```

SSE events:
- `delta` with `{ "delta": "..." }` chunks
- `result` with the final JSON response

Env vars (optional):

```bash
ADDR=:8080
CHUNKS_PATH=./out/alicanteabout_chunks.json
CACHE_PATH=./out/embeddings_cache.json
EMBED_PROVIDER=openai
EMBED_MODEL=text-embedding-3-small
CHAT_MODEL=gpt-4o-mini
TOP_K=3
MAX_SOURCES=2
MIN_SCORE=0.25
CORS_ALLOWED_ORIGIN=https://alicanteabout.com
RATE_LIMIT=30
RATE_WINDOW=1m
TIMEOUT=30s
EMBED_CACHE_MAX=256
CHAT_JWT_SECRET=your-shared-secret
CHAT_JWT_ISSUER=alicanteabout.com
CHAT_JWT_AUDIENCE=alicanteabout-chat
CHAT_JWT_LEEWAY=10s
CHAT_JWT_TTL=120s
CHAT_DB_DSN=postgres://user:pass@localhost:5432/alicanteabout?sslmode=disable
RUN_MIGRATIONS=true
CHAT_LOG_BUFFER=1000
CHAT_LOG_BATCH_SIZE=100
CHAT_LOG_FLUSH_EVERY=500ms
CHAT_LOG_REPORT_EVERY=30s
CHAT_LOG_DISABLE=false
MIGRATIONS_DIR=internal/storage/migrations
```

Chat logging behavior:

- Logging is enabled only when `CHAT_DB_DSN` is set and `CHAT_LOG_DISABLE=false`.
- Logs are queued and written asynchronously in batches.
- If the queue is full, logs are dropped (best effort).
- Dropped count is reported periodically (interval: `CHAT_LOG_REPORT_EVERY`).

## Local Secrets (.env)

Create a `.env` file in the project root for local development. It is ignored by git.

```bash
OPENAI_API_KEY=your-key-here
```

Use `.env.example` as a starting point for all supported env vars.

## Local Postgres (Docker)

For local development, use Docker Compose to run Postgres:

```bash
make db-up
```

Connection string (set in `.env`):

```bash
CHAT_DB_DSN=postgres://alicante:alicante@localhost:5432/alicanteabout?sslmode=disable
```

Run migrations (requires `goose` installed). Install with:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Ensure `MIGRATIONS_DIR` is set in your environment or `.env`:

```bash
make db-migrate
```

## CI Migrations (Railway)

There is a GitHub Actions workflow that runs migrations when files in
`internal/storage/migrations/` change. It also supports manual runs.

Setup:

1) Create a GitHub secret named `RAILWAY_DATABASE_URL` with your Railway Postgres URL.
2) Push a migration or trigger the workflow manually.

Stop the DB:

```bash
make db-down
```

Reset the DB (drops volume):

```bash
make db-reset
```

Open a psql shell:

```bash
make db-shell
```

## Behavior Guarantees

- English only (`lang: "en"`). Other languages are rejected.
- Sources are always returned when an answer is provided.
- If sources are low relevance, the API returns the fallback "I don't know..." answer.
- CORS is restricted to `https://alicanteabout.com`.
- Rate limiting is enforced per IP.
- JWT auth is required for `/chat`.
- On startup, the chat service runs database migrations when `CHAT_DB_DSN` is set. Set `RUN_MIGRATIONS=false` to skip.

## WordPress Chat Token Plugin

The WordPress plugin issues short-lived JWTs for the chat API.

Location in repo:

```
wordpress/alicanteabout-chat-token/
```

Install:

1) Copy the folder into `wp-content/plugins/alicanteabout-chat-token/`.
2) Activate "AlicanteAbout Chat Token" in WP Admin.
3) Go to Settings → Chat Token, set the JWT secret and rate limit.
4) Settings are stored in the `alicanteabout_chat_token_settings` option.

Endpoint:

```
/wp-json/alicanteabout/v1/chat-token
```

Example token request:

```bash
curl -s "https://alicanteabout.com/wp-json/alicanteabout/v1/chat-token"
```

Example URL:

```
https://alicanteabout.com/wp-json/alicanteabout/v1/chat-token
```

Frontend flow:

1) Fetch the JWT from the WordPress endpoint.
2) Call the Go `/chat` API with `Authorization: Bearer <jwt>`.

Token details:

- Signed with HS256 using the configured secret
- Short-lived (default 120s)
- Includes `iss` and `aud` that must match the API configuration

## Chat Token CLI (Dev Only)

The `chat-token` CLI mints a short-lived JWT locally for testing.

```bash
go run ./cmd/chat-token -secret "$CHAT_JWT_SECRET"
```

Flags:

- `-secret` (or `CHAT_JWT_SECRET`)
- `-issuer` (default `alicanteabout.com`)
- `-audience` (default `alicanteabout-chat`)
- `-ttl` (default `120s`, or `CHAT_JWT_TTL`)

Example dev request:

```bash
TOKEN="$(go run ./cmd/chat-token -secret "$CHAT_JWT_SECRET")"
curl -X POST http://localhost:8080/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question":"How do I get from the airport to the city center?","lang":"en"}'
```

## WordPress Chat Widget Plugin

The chat widget is a separate WordPress plugin that renders a floating button and lazy-loads a modal UI with streaming support.

Location in repo:

```
wordpress/alicanteabout-chat-widget/
```

Install:

1) Copy the folder into `wp-content/plugins/alicanteabout-chat-widget/`.
2) Activate "AlicanteAbout Chat Widget" in WP Admin.
3) Go to Settings → Chat Widget to configure API URL, token URL, and labels.

Defaults:

- Button label: "Ask Alicante"
- API URL: `https://api.alicanteabout.com/chat`
- Token URL: `/wp-json/alicanteabout/v1/chat-token`
- Allowed languages: `en` (uses page `lang` attribute)

Behavior:

- Floating button loads only a tiny bootstrap script.
- Full widget JS/CSS is loaded on first click.
- Modal UI with streaming responses.
- Chat resets on close.
- Disclaimers shown for content and GDPR.
- Widget only renders when the page `<html lang>` matches the allowed languages list (default `en`).

## Commands

**export**: Fetches posts and pages from WordPress REST API and exports to JSON and text files.

**search**: Interactive RAG search using OpenAI embeddings with caching for efficient retrieval.

**chat**: HTTP API that performs RAG retrieval and returns grounded answers with sources.

## Tests

```bash
go test ./cmd/chat
go test ./...
```

PHP (plugin) tests:

```bash
cd wordpress/alicanteabout-chat-token
phpunit
```

Requires a local `phpunit` installation (global or via Composer).

## Building (optional)

To compile binaries for faster startup:

```bash
go build -o bin/export ./cmd/export
go build -o bin/search ./cmd/search
go build -o bin/chat ./cmd/chat
```

## Requirements

- Go toolchain
- `OPENAI_API_KEY` set in the environment for embeddings and chat


## Why this repository is public

The goal of publishing this repository is to document and share the technical
decisions behind a real RAG system, not to provide a finished product.

Future iterations may explore privately.