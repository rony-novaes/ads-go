package ads

// Ad entregue no /ads.
// - ID        : uuid do anúncio (para clique)
// - Type      : tipo expandido a partir do JSON "types"
// - TargetURL : coluna redirect
// - File      : identificador do arquivo (do objeto por tipo, ex.: "389d...b917cd")
// - FileExt   : extensão do arquivo (do objeto por tipo, ex.: "png")
// - Active    : sempre true nos selecionados
type Ad struct {
	ID        string `json:"id"`         // uuid (char(36)) do anúncio
	Type      int    `json:"type"`       // 1..4
	TargetURL string `json:"target_url"` // redirect
	File      string `json:"file"`       // do JSON types[<type>].file
	FileExt   string `json:"file_ext"`   // do JSON types[<type>].extension
	Active    bool   `json:"active"`
}
