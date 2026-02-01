package chat

import (
	"fmt"
	"strings"

	"content-rag-chat/internal/rag"
)

type promptSource struct {
	Title   string
	URL     string
	Excerpt string
}

func buildPrompt(question string, hits []rag.ScoredChunk, topK int) (string, []promptSource) {
	unique := map[string]promptSource{}
	ordered := make([]promptSource, 0, len(hits))
	for _, h := range hits {
		if _, ok := unique[h.Chunk.URL]; ok {
			continue
		}
		ps := promptSource{
			Title:   h.Chunk.Title,
			URL:     h.Chunk.URL,
			Excerpt: h.Chunk.Text,
		}
		unique[h.Chunk.URL] = ps
		ordered = append(ordered, ps)
		if len(ordered) >= topK {
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("You are a helpful assistant for AlicanteAbout.com, a tourism guide for Alicante, Spain.\n")
	sb.WriteString("Use ONLY the provided sources to answer. If the answer is not in the sources, say \"I don't know based on AlicanteAbout content.\".\n")
	sb.WriteString("Respond in JSON with keys: answer (string) and sources (array of {title,url}).\n")
	sb.WriteString("Only include sources you actually used. Do not invent sources.\n\n")
	sb.WriteString("Question:\n")
	sb.WriteString(question)
	sb.WriteString("\n\nSources:\n")
	for i, src := range ordered {
		fmt.Fprintf(&sb, "\n[%d] %s\nURL: %s\nExcerpt:\n%s\n", i+1, src.Title, src.URL, src.Excerpt)
	}

	return sb.String(), ordered
}

func filterSources(ordered []promptSource, picked []sourceItem, max int) []sourceItem {
	sourceMap := map[string]string{}
	for _, src := range ordered {
		sourceMap[src.URL] = src.Title
	}
	cleanSources := make([]sourceItem, 0, len(picked))
	for _, src := range picked {
		if src.URL == "" {
			continue
		}
		title, ok := sourceMap[src.URL]
		if !ok {
			continue
		}
		if src.Title == "" {
			src.Title = title
		}
		cleanSources = append(cleanSources, src)
		if len(cleanSources) >= max {
			break
		}
	}
	return cleanSources
}
