package ads

// Mantém o tipo Ad usado internamente pelo pacote ads (cache/service).
// NÃO redefina Repository aqui (ele já existe em repo.go).
type Ad struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	TargetURL string `json:"target_url"`
	File      string `json:"file"`
	FileExt   string `json:"file_ext"`
	Active    bool   `json:"active"`
}
