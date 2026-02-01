package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"content-rag-chat/internal/config"
	"content-rag-chat/internal/rag"
)

func main() {
	// Inputs
	chunksPath := flag.String("chunks", "./out/alicanteabout_chunks.json", "Path to chunks JSON or JSONL")
	cachePath := flag.String("cache", "./out/embeddings_cache.json", "Path to embeddings cache JSON")
	outPrompt := flag.Bool("prompt", true, "Print a ready-to-use prompt with sources after ranking")
	topK := flag.Int("k", 5, "Top K chunks to retrieve")

	// Embeddings config
	provider := flag.String("provider", "openai", "Embeddings provider: openai (default)")
	model := flag.String("model", "text-embedding-3-small", "Embeddings model (provider-specific)")
	batchSize := flag.Int("batch", 64, "Batch size for embedding requests")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP timeout for embedding requests")
	sleep := flag.Duration("sleep", 150*time.Millisecond, "Sleep between embedding requests (rate-limit friendly)")

	flag.Parse()

	if err := config.LoadDotEnv(".env"); err != nil {
		fatal(err)
	}

	// Load chunks
	chunks, err := rag.ReadChunks(*chunksPath)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Loaded %d chunks\n", len(chunks))

	// Load cache (or create)
	cache, err := rag.LoadCache(*cachePath)
	if err != nil {
		fatal(err)
	}
	if cache.Items == nil {
		cache.Items = map[string]rag.EmbedCacheItem{}
	}
	if cache.Model != "" && cache.Model != *model {
		fmt.Printf("⚠️ Cache model is %q but you selected %q. We'll keep cache but only reuse matching items.\n", cache.Model, *model)
	}

	// Prepare embedding client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if *provider == "openai" && apiKey == "" {
		fatal(fmt.Errorf("OPENAI_API_KEY is not set"))
	}
	client := &http.Client{Timeout: *timeout}

	// Ensure embeddings exist for all chunks
	ctx := context.Background()
	needCount := 0
	for _, ch := range chunks {
		h := rag.TextHash(ch.Text)
		item, ok := cache.Items[ch.ChunkID]
		if ok && item.Hash == h && item.Dim > 0 && len(item.Vector) == item.Dim && cache.Model == *model {
			continue
		}
		needCount++
	}
	fmt.Printf("Embeddings missing/outdated: %d\n", needCount)

	if needCount > 0 {
		fmt.Println("Generating embeddings (cached)…")
		if err := rag.EmbedAll(ctx, client, *provider, apiKey, *model, chunks, cache, *batchSize, *sleep); err != nil {
			fatal(err)
		}
		cache.Model = *model
		if err := rag.SaveCache(*cachePath, cache); err != nil {
			fatal(err)
		}
		fmt.Printf("Saved cache: %s\n", *cachePath)
	}

	// Build in-memory embedding matrix (normalized)
	entries := rag.BuildIndex(chunks, cache, *model)
	fmt.Printf("Index ready: %d vectors (normalized)\n", len(entries))

	// Interactive search loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nAsk a question (or 'exit'): ")
		q, _ := reader.ReadString('\n')
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		if q == "exit" || q == "quit" {
			fmt.Println("bye")
			return
		}

		qVec, err := rag.EmbedQuery(ctx, client, *provider, apiKey, *model, q)
		if err != nil {
			fmt.Println("Embedding error:", err)
			continue
		}
		rag.Normalize(qVec)

		results := rag.TopKSearch(entries, qVec, *topK)

		fmt.Printf("\nTop %d results:\n", len(results))
		for i, r := range results {
			fmt.Printf("\n#%d  score=%.4f\n", i+1, r.Score)
			fmt.Printf("Title: %s\n", r.Chunk.Title)
			fmt.Printf("URL:   %s\n", r.Chunk.URL)
			fmt.Printf("Slug:  %s\n", r.Chunk.Slug)
			preview := r.Chunk.Text
			if len(preview) > 420 {
				preview = preview[:420] + "…"
			}
			fmt.Printf("Text:  %s\n", strings.ReplaceAll(preview, "\n", " "))
		}

		if *outPrompt {
			fmt.Println("\n--- Prompt (copy/paste) ---")
			fmt.Println(rag.BuildPrompt(q, results))
			fmt.Println("--- End prompt ---")
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
