package ads

// Ad é a unidade que entregamos para o /ads.
// ID usa o UUID do anúncio (string de 36 chars).
// Type vem do array JSON "types" da tabela.
// TargetURL mapeia a coluna "redirect".
// FileExt guarda a extensão do arquivo para o front montar a URL da imagem.
// Active fica true nos itens selecionados.
type Ad struct {
	ID        string `json:"id"`         // uuid
	Type      int    `json:"type"`       // 1..4
	TargetURL string `json:"target_url"` // redirect
	FileExt   string `json:"file_ext"`   // file_extension (ex.: "jpg", "png")
	Active    bool   `json:"active"`
}
