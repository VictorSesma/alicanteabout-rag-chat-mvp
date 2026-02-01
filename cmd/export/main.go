package main

import (
	"encoding/json"
	"flag"
	"fmt"
	htmlpkg "html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type wpItem struct {
	ID          int    `json:"id"`
	Slug        string `json:"slug"`
	Link        string `json:"link"`
	ModifiedGMT string `json:"modified_gmt"`
	Title       struct {
		Rendered string `json:"rendered"`
	} `json:"title"`
	Content struct {
		Rendered string `json:"rendered"`
	} `json:"content"`
}

type doc struct {
	ID          int    `json:"id"`
	Type        string `json:"type"` // "post" | "page"
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	ModifiedGMT string `json:"modified_gmt"`
	ContentText string `json:"content_text"`
}

var (
	reMultiNL   = regexp.MustCompile(`\n{3,}`)
	reMultiSp   = regexp.MustCompile(`[ \t]{2,}`)
	reTrimSpace = regexp.MustCompile(`\s+\n`)
)

func main() {
	defaultBaseURL := envString("BASE_URL", "https://alicanteabout.com")
	baseURL := flag.String("base", defaultBaseURL, "WordPress site base URL (e.g. https://example.com)")
	outDir := flag.String("out", "./out", "Output directory")
	perPage := flag.Int("per_page", 100, "WP REST per_page (max often 100)")
	timeout := flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	sleep := flag.Duration("sleep", 0*time.Millisecond, "Sleep between requests (e.g. 200ms)")

	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	docsDir := filepath.Join(*outDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		fatal(err)
	}

	client := &http.Client{Timeout: *timeout}

	var all []doc

	// WP endpoints: /wp-json/wp/v2/posts and /pages
	for _, typ := range []struct {
		endpoint string
		docType  string
	}{
		{endpoint: "posts", docType: "post"},
		{endpoint: "pages", docType: "page"},
	} {
		fmt.Printf("Fetching %s...\n", typ.endpoint)
		items, err := fetchAll(client, *baseURL, typ.endpoint, *perPage, *sleep)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("  -> %d items\n", len(items))

		for _, it := range items {
			title := htmlUnescape(strings.TrimSpace(it.Title.Rendered))
			txt := htmlToText(it.Content.Rendered)

			all = append(all, doc{
				ID:          it.ID,
				Type:        typ.docType,
				Slug:        it.Slug,
				Title:       title,
				URL:         it.Link,
				ModifiedGMT: it.ModifiedGMT,
				ContentText: txt,
			})
		}
	}

	// Sort by URL for stable output
	sort.Slice(all, func(i, j int) bool {
		return all[i].URL < all[j].URL
	})

	// Write corpus JSON
	corpusPath := filepath.Join(*outDir, "alicanteabout_corpus.json")
	if err := writeJSON(corpusPath, all); err != nil {
		fatal(err)
	}
	fmt.Printf("Saved: %s (%d docs)\n", corpusPath, len(all))

	// Write individual text files (debug)
	for _, d := range all {
		safe := safeFilename(fmt.Sprintf("%s_%s", d.Type, d.Slug))
		p := filepath.Join(docsDir, safe+".txt")
		body := fmt.Sprintf("%s\n\nURL: %s\n\n%s\n", d.Title, d.URL, d.ContentText)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			fatal(err)
		}
	}
	fmt.Printf("Saved text files: %s\n", docsDir)
}

func fetchAll(client *http.Client, baseURL, endpoint string, perPage int, sleep time.Duration) ([]wpItem, error) {
	var out []wpItem
	page := 1

	for {
		url := fmt.Sprintf("%s/wp-json/wp/v2/%s?per_page=%d&page=%d&_fields=id,slug,link,modified_gmt,title,content",
			strings.TrimRight(baseURL, "/"),
			endpoint,
			perPage,
			page,
		)

		items, status, err := fetchPage(client, url)
		if err != nil {
			return nil, err
		}
		if status == http.StatusNotFound {
			return nil, fmt.Errorf("endpoint not found: %s", url)
		}
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			return nil, fmt.Errorf("access denied (%d) fetching: %s", status, url)
		}

		if len(items) == 0 {
			break
		}
		out = append(out, items...)

		if len(items) < perPage {
			break
		}
		page++

		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
	return out, nil
}

func fetchPage(client *http.Client, url string) ([]wpItem, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "victorsesma-corpus-export/1.0")

	res, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// read some body for debugging
		b, _ := io.ReadAll(io.LimitReader(res.Body, 8_192))
		return nil, res.StatusCode, fmt.Errorf("HTTP %d: %s\n%s", res.StatusCode, url, string(b))
	}

	var items []wpItem
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&items); err != nil {
		return nil, res.StatusCode, err
	}
	return items, res.StatusCode, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ---- HTML cleaning ----

// htmlToText converts HTML to readable plain text.
// It removes scripts/styles and keeps basic structure with newlines.
func htmlToText(htmlStr string) string {
	if strings.TrimSpace(htmlStr) == "" {
		return ""
	}

	// Parse HTML fragment as a full doc for robustness
	node, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		// fallback: strip tags roughly
		return normalizeText(stripTagsRough(htmlStr))
	}

	var sb strings.Builder
	var walk func(n *html.Node)

	walk = func(n *html.Node) {
		// Skip script/style/nav/footer
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style", "nav", "footer":
				return
			case "br":
				sb.WriteString("\n")
			case "p", "div", "section", "article", "header", "li":
				// ensure separation
				sb.WriteString("\n")
			case "h1", "h2", "h3", "h4":
				sb.WriteString("\n")
			}
		}

		if n.Type == html.TextNode {
			txt := strings.TrimSpace(n.Data)
			if txt != "" {
				sb.WriteString(txt)
				sb.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}

		// close block tags with newline
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "p", "li", "h1", "h2", "h3", "h4":
				sb.WriteString("\n")
			}
		}
	}

	walk(node)

	return normalizeText(sb.String())
}

func normalizeText(s string) string {
	s = htmlUnescape(s)
	s = strings.ReplaceAll(s, "\u00a0", " ") // nbsp
	s = reMultiSp.ReplaceAllString(s, " ")
	s = reTrimSpace.ReplaceAllString(s, "\n")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func stripTagsRough(s string) string {
	// very rough fallback (should rarely be used)
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, " ")
}

func htmlUnescape(s string) string {
	// minimal entity unescape without pulling extra deps
	// The html package has UnescapeString in stdlib: html.UnescapeString
	// But name conflicts with x/net/html, so we call via stdlib alias:
	return stdlibHTMLUnescape(s)
}

func stdlibHTMLUnescape(s string) string {
	return htmlpkg.UnescapeString(s)
}

func safeFilename(s string) string {
	s = strings.ToLower(s)
	// keep letters, numbers, dash, underscore
	re := regexp.MustCompile(`[^a-z0-9\-_]+`)
	s = re.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "doc"
	}
	return s
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
