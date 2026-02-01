# cmd/DESIGN

Design goals for command binaries.

- Thin entry points: orchestration only; business logic lives in internal/*.
- Fast failure: missing env or critical inputs should exit with clear errors.
- Deterministic outputs: stable ordering and reproducible files when possible.
- Respect defaults: CLI flags should have sane defaults and align with README.
- Minimal side effects: only write to intended output files and directories.
- Logging: use stdout/stderr with simple, parseable messages.
