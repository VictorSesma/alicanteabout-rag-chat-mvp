# internal/DESIGN

Design principles for internal packages.

- Explicit data flow: inputs -> embeddings -> retrieval -> grounded answer.
- Safety first: always allow "I don't know" when sources are weak.
- Privacy: redact and hash user input before storage.
- Reliability: timeouts and rate limiting are mandatory on network calls.
- Streaming: prefer SSE for UI responsiveness; non-streaming must still be correct.
- Testability: keep pure helpers and small interfaces for easy unit tests.
