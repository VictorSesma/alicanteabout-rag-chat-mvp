package rag

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Chunk struct {
	ChunkID     string `json:"chunk_id"`
	DocID       int    `json:"doc_id"`
	DocType     string `json:"type"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	ModifiedGMT string `json:"modified_gmt"`
	IndexPage   bool   `json:"index_page"`
	Text        string `json:"text"`
	CharLen     int    `json:"char_len"`
}

// RawChunk represents the format in alicanteabout_chunks.json
type RawChunk struct {
	ID          int    `json:"id"`
	DocType     string `json:"type"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	ModifiedGMT string `json:"modified_gmt"`
	ContentText string `json:"content_text"`
}

type EmbedCacheItem struct {
	ID        string    `json:"id"` // chunk_id
	Hash      string    `json:"hash"`
	Dim       int       `json:"dim"`
	Vector    []float32 `json:"vector"`
	UpdatedAt string    `json:"updated_at"`
}

type EmbedCache struct {
	Model string                    `json:"model"`
	Items map[string]EmbedCacheItem `json:"items"`
}

type ScoredChunk struct {
	Chunk Chunk
	Score float32
}

type Entry struct {
	Chunk Chunk
	Vec   []float32
}

// ReadChunks loads chunks from JSON array (.json) or JSONL (.jsonl).
func ReadChunks(path string) ([]Chunk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		return readChunksJSON(f)
	}
	return readChunksJSONL(f)
}

// readChunksJSON reads a JSON array of RawChunk and converts to []Chunk.
func readChunksJSON(f *os.File) ([]Chunk, error) {
	var raw []RawChunk
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode JSON array: %w", err)
	}

	chunks := make([]Chunk, 0, len(raw))
	for _, r := range raw {
		ch := Chunk{
			ChunkID:     fmt.Sprintf("%s-%d", r.Slug, r.ID),
			DocID:       r.ID,
			DocType:     r.DocType,
			Slug:        r.Slug,
			Title:       r.Title,
			URL:         r.URL,
			ModifiedGMT: r.ModifiedGMT,
			IndexPage:   false,
			Text:        r.ContentText,
			CharLen:     len(r.ContentText),
		}
		chunks = append(chunks, ch)
	}
	return chunks, nil
}

// readChunksJSONL reads JSONL format (one Chunk per line).
func readChunksJSONL(f *os.File) ([]Chunk, error) {
	var chunks []Chunk
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ch Chunk
		if err := json.Unmarshal([]byte(line), &ch); err != nil {
			return nil, fmt.Errorf("bad jsonl line: %w", err)
		}
		chunks = append(chunks, ch)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return chunks, nil
}

// BuildIndex creates a normalized in-memory matrix from cached embeddings.
func BuildIndex(chunks []Chunk, cache *EmbedCache, model string) []Entry {
	entries := make([]Entry, 0, len(chunks))
	for _, ch := range chunks {
		item, ok := cache.Items[ch.ChunkID]
		if !ok || cache.Model != model {
			continue
		}
		v := make([]float32, len(item.Vector))
		copy(v, item.Vector)
		Normalize(v)
		entries = append(entries, Entry{Chunk: ch, Vec: v})
	}
	return entries
}

// ---------------- Embeddings (OpenAI) ----------------

type openAIEmbeddingsRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"` // string or []string
}

type openAIEmbeddingsResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func EmbedAll(ctx context.Context, client *http.Client, provider, apiKey, model string, chunks []Chunk, cache *EmbedCache, batchSize int, sleep time.Duration) error {
	type pending struct {
		ch   Chunk
		hash string
	}
	var todo []pending
	for _, ch := range chunks {
		h := TextHash(ch.Text)
		item, ok := cache.Items[ch.ChunkID]
		if ok && cache.Model == model && item.Hash == h && item.Dim > 0 && len(item.Vector) == item.Dim {
			continue
		}
		todo = append(todo, pending{ch: ch, hash: h})
	}

	for i := 0; i < len(todo); i += batchSize {
		end := i + batchSize
		if end > len(todo) {
			end = len(todo)
		}
		batch := todo[i:end]

		inputs := make([]string, 0, len(batch))
		for _, p := range batch {
			inputs = append(inputs, fmt.Sprintf("%s\n%s\n\n%s", p.ch.Title, p.ch.URL, p.ch.Text))
		}

		vecs, err := EmbedTexts(ctx, client, provider, apiKey, model, inputs)
		if err != nil {
			return err
		}
		if len(vecs) != len(batch) {
			return fmt.Errorf("embedding count mismatch: got=%d want=%d", len(vecs), len(batch))
		}

		now := time.Now().UTC().Format(time.RFC3339)
		for j, p := range batch {
			v := vecs[j]
			cache.Items[p.ch.ChunkID] = EmbedCacheItem{
				ID:        p.ch.ChunkID,
				Hash:      p.hash,
				Dim:       len(v),
				Vector:    v,
				UpdatedAt: now,
			}
		}

		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
	return nil
}

func EmbedQuery(ctx context.Context, client *http.Client, provider, apiKey, model, query string) ([]float32, error) {
	vecs, err := EmbedTexts(ctx, client, provider, apiKey, model, []string{query})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func EmbedTexts(ctx context.Context, client *http.Client, provider, apiKey, model string, inputs []string) ([][]float32, error) {
	switch provider {
	case "openai":
		return openAIEmbed(ctx, client, apiKey, model, inputs)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func openAIEmbed(ctx context.Context, client *http.Client, apiKey, model string, inputs []string) ([][]float32, error) {
	reqBody := openAIEmbeddingsRequest{
		Model: model,
		Input: inputs,
	}
	b, _ := json.Marshal(reqBody)

	reqID := RequestID(ctx)
	if reqID == "" {
		reqID = "unknown"
	}
	log.Printf("req_id=%s openai embeddings request_bytes=%d inputs=%d model=%s", reqID, len(b), len(inputs), model)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 50*1024*1024))
	log.Printf("req_id=%s openai embeddings response_status=%d response_bytes=%d took=%s", reqID, res.StatusCode, len(body), fmtDuration(time.Since(start)))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("openai embeddings http %d: %s", res.StatusCode, string(body))
	}

	var out openAIEmbeddingsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w (body: %q)", err, string(body))
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openai error: %s (%s)", out.Error.Message, out.Error.Type)
	}

	vecs := make([][]float32, len(inputs))
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(inputs) {
			continue
		}
		vecs[d.Index] = d.Embedding
	}
	for i := range vecs {
		if vecs[i] == nil {
			return nil, fmt.Errorf("missing embedding for index %d", i)
		}
	}
	return vecs, nil
}

// ---------------- Search ----------------

func TopKSearch(entries []Entry, q []float32, k int) []ScoredChunk {
	if k <= 0 {
		return nil
	}
	results := make([]ScoredChunk, 0, k)

	for _, e := range entries {
		s := Dot(q, e.Vec)
		results = append(results, ScoredChunk{Chunk: e.Chunk, Score: s})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > k {
		results = results[:k]
	}
	return results
}

func Dot(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var s float32
	for i := 0; i < n; i++ {
		s += a[i] * b[i]
	}
	return s
}

func Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	n := float32(math.Sqrt(sum))
	if n == 0 {
		return
	}
	for i := range v {
		v[i] = v[i] / n
	}
}

func fmtDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fs (%dms)", d.Seconds(), d.Milliseconds())
}

// ---------------- Prompt builder ----------------

func BuildPrompt(question string, hits []ScoredChunk) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant for a tourism website about Alicante.\n")
	sb.WriteString("Answer the user's question using ONLY the provided sources. If the answer is not in the sources, say you don't know and suggest the closest source.\n")
	sb.WriteString("Always include a short 'Sources' section with the URLs you used.\n\n")

	sb.WriteString("User question:\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")

	sb.WriteString("Sources (excerpts):\n")
	for i, h := range hits {
		sb.WriteString(fmt.Sprintf("\n[%d] %s\nURL: %s\nExcerpt:\n%s\n", i+1, h.Chunk.Title, h.Chunk.URL, h.Chunk.Text))
	}
	sb.WriteString("\nAnswer:\n")
	return sb.String()
}

// ---------------- Cache ----------------

func LoadCache(path string) (*EmbedCache, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EmbedCache{Items: map[string]EmbedCacheItem{}}, nil
		}
		return nil, err
	}
	var c EmbedCache
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Items == nil {
		c.Items = map[string]EmbedCacheItem{}
	}
	return &c, nil
}

func SaveCache(path string, c *EmbedCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func TextHash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
