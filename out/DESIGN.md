# out/DESIGN

File format notes

- alicanteabout_chunks.json
  - JSON array of RawChunk items (id, type, slug, title, url, modified_gmt, content_text).
  - JSONL variant is also supported by internal/rag.
- embeddings_cache.json
  - {"model": "...", "items": {chunk_id: {hash, dim, vector, updated_at}}}
  - Vectors must match the embedding model in use.

Guidelines
- Keep outputs deterministic (sorted, stable ordering).
- Regenerate instead of editing by hand.
