package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Searcher mencari fakta terkini di Search Engine.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]Source, error)
}

// SearchClient memanggil SEARCH_ENGINE_ENDPOINT (GET ?q=...).
type SearchClient struct {
	endpoint string
	key      string
	http     *http.Client
}

// NewSearchClient membuat client Search Engine.
func NewSearchClient(endpoint, key string) *SearchClient {
	return &SearchClient{endpoint: endpoint, key: key, http: &http.Client{Timeout: 15 * time.Second}}
}

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}

// findList mencari array hasil pada beberapa bentuk respons umum.
func findList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		for _, k := range []string{"results", "data", "items", "organic", "web", "organic_results"} {
			if inner, ok := t[k]; ok {
				if l := findList(inner); l != nil {
					return l
				}
			}
		}
	}
	return nil
}

// Search mengembalikan maksimal `limit` sumber. Bentuk respons dibaca secara toleran.
func (s *SearchClient) Search(ctx context.Context, query string, limit int) ([]Source, error) {
	u, err := url.Parse(s.endpoint)
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("q", query)
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if s.key != "" {
		req.Header.Set("Authorization", "Bearer "+s.key)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("search engine status %d", resp.StatusCode)
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := []Source{}
	for _, item := range findList(raw) {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		link := str(m, "url", "link", "href")
		if link == "" {
			continue
		}
		out = append(out, Source{
			Title:   clip(str(m, "title", "name"), 150),
			URL:     link,
			Snippet: clip(str(m, "snippet", "content", "description", "text"), 300),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
