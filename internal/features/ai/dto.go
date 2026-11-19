package ai

// GenerateRequest payload AI generator. use_search default true.
type GenerateRequest struct {
	Topic     string `json:"topic" binding:"required"`
	Category  string `json:"category"`
	UseSearch *bool  `json:"use_search"`
}

// Source adalah satu sumber rujukan dari Search Engine.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

// GenerateResponse adalah draf artikel hasil AI.
type GenerateResponse struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}
