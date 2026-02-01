# cmd/AGENTS

Notes for contributors working in cmd/. These are entry-point binaries; keep them thin and deterministic.

- Keep each command single-purpose and minimal; prefer calling internal/* packages for logic.
- Fail fast on missing required env or inputs (e.g., OPENAI_API_KEY, CHAT_JWT_SECRET).
- Keep default paths aligned with repo outputs (e.g., ./out/alicanteabout_chunks.json).
- Flags should mirror env var names where possible and keep backward compatibility.
- Avoid writing to .env (only users change env files).
- If you change CLI behavior, update README.md usage examples.

Commands
- cmd/export: export WP content to ./out (corpus JSON + docs/* text).
- cmd/search: local interactive retrieval and embeddings cache generation.
- cmd/chat: HTTP API server for /chat and /healthz.
- cmd/chat-token: dev-only JWT minting CLI.
