# internal/config

Minimal configuration helpers.

- LoadDotEnv loads key=value pairs from .env if present.
- Does not overwrite existing environment variables.
- No writes to .env; only read.

Usage
- Called by cmd/chat, cmd/search, and cmd/chat-token before reading env vars.
