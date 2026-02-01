# out/ARCHITECTURE

Expected artifacts

- alicanteabout_corpus.json
  - Raw export output from cmd/export (array of docs).
- docs/
  - One text file per WP page/post for inspection.
- alicanteabout_chunks.json
  - Chunked corpus consumed by cmd/search and cmd/chat.
  - This file is produced by a separate chunking step (not in this repo).
- embeddings_cache.json
  - Embeddings cache keyed by chunk_id.
  - Generated/updated by cmd/search.

Data dependencies
- cmd/chat requires alicanteabout_chunks.json + embeddings_cache.json.
- cmd/search can create/update embeddings_cache.json if chunks exist.
