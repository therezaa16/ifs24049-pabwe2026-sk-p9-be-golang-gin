package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrOpenAI dibungkus saat pemanggilan LLM gagal.
var ErrOpenAI = errors.New("gagal menghubungi layanan OpenAI")

var validCategories = map[string]string{
	"teknologi": "Teknologi", "pendidikan": "Pendidikan", "gaya hidup": "Gaya Hidup",
	"bisnis": "Bisnis", "opini": "Opini", "umum": "Umum",
}

const systemPrompt = "Kamu penulis blog berbahasa Indonesia. Balas HANYA dengan JSON valid " +
	`berbentuk {"title":"...","category":"...","excerpt":"...","content":"..."}. ` +
	"category salah satu dari: Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum. " +
	"excerpt 1-2 kalimat. content berupa artikel Markdown (judul, subjudul, paragraf) sekitar 300-450 kata."

// Service menyusun draf artikel dari topik (+ sumber Search Engine).
type Service struct {
	search Searcher
	chat   Completer
}

// NewService membuat service AI.
func NewService(search Searcher, chat Completer) *Service {
	return &Service{search: search, chat: chat}
}

func clipRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n])
	}
	return string(r)
}

func buildPrompt(topic, category string, sources []Source) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Topik: %s\n", topic)
	if category != "" {
		fmt.Fprintf(&b, "Kategori yang diinginkan: %s\n", category)
	}
	if len(sources) > 0 {
		b.WriteString("Gunakan fakta terkini dari sumber berikut (jangan mengarang di luar sumber):\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "[%d] %s - %s\n", i+1, s.Title, s.Snippet)
		}
	}
	return b.String()
}

func stripFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func appendSources(content string, sources []Source) string {
	if len(sources) == 0 {
		return content
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(content, "\n"))
	b.WriteString("\n\n## Sumber Rujukan\n\n")
	for _, s := range sources {
		title := s.Title
		if title == "" {
			title = s.URL
		}
		fmt.Fprintf(&b, "- [%s](%s)\n", title, s.URL)
	}
	return b.String()
}

func parseDraft(raw, topic, category string, sources []Source) *GenerateResponse {
	var d struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		Excerpt  string `json:"excerpt"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(stripFence(raw)), &d); err != nil || strings.TrimSpace(d.Content) == "" {
		d.Title, d.Category, d.Excerpt, d.Content = "", "", "", raw
	}
	if strings.TrimSpace(d.Title) == "" {
		d.Title = topic
	}
	cat := validCategories[strings.ToLower(strings.TrimSpace(d.Category))]
	if cat == "" {
		cat = validCategories[strings.ToLower(strings.TrimSpace(category))]
	}
	if cat == "" {
		cat = "Umum"
	}
	if strings.TrimSpace(d.Excerpt) == "" {
		d.Excerpt = clipRunes(strings.TrimLeft(strings.TrimSpace(d.Content), "# "), 160)
	}
	if sources == nil {
		sources = []Source{}
	}
	return &GenerateResponse{
		Title: strings.TrimSpace(d.Title), Category: cat, Excerpt: strings.TrimSpace(d.Excerpt),
		Content: appendSources(d.Content, sources), Sources: sources,
	}
}

// Generate menghasilkan draf artikel. Kegagalan Search Engine tidak menggagalkan proses.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	topic := clipRunes(req.Topic, 200)
	var sources []Source
	if (req.UseSearch == nil || *req.UseSearch) && s.search != nil {
		if found, err := s.search.Search(ctx, topic, 3); err == nil {
			sources = found
		}
	}
	raw, err := s.chat.Complete(ctx, systemPrompt, buildPrompt(topic, req.Category, sources))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenAI, err)
	}
	return parseDraft(raw, topic, req.Category, sources), nil
}
