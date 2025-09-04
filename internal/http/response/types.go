package response

type SearchItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Dek         string   `json:"dek,omitempty"`
	Canonical   string   `json:"canonical"`
	Section     string   `json:"section,omitempty"`
	Topics      []string `json:"topics,omitempty"`
	Author      string   `json:"author,omitempty"`
	Column      string   `json:"column,omitempty"`
	PublishedAt string   `json:"published_at,omitempty"`
	HasImage    bool     `json:"has_image"`
	Score       float64  `json:"score"`
	Snippet     string   `json:"snippet,omitempty"`
}

type SearchResponse struct {
	TookMS int                    `json:"took_ms"`
	Total  int                    `json:"total"`
	Items  []SearchItem           `json:"items"`
	Facets map[string]any         `json:"facets,omitempty"`
}
