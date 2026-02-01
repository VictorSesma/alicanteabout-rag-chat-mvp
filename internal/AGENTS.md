# internal/AGENTS

Guidance for changes in internal/ packages.

- Keep packages focused: chat, rag, storage, config.
- Prefer dependency-free helpers over new external libraries.
- Preserve public function behavior; update tests when behavior changes.
- Avoid leaking secrets or PII; sanitize before logging or storing.
- Keep timeouts, limits, and safety defaults conservative.
- Do not mutate .env or assume it exists; LoadDotEnv is best-effort only.
