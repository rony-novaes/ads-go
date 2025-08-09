package ads

// Estrutura no formato esperado pelo JS do Node
type TypeVariant struct {
	File      string `json:"file"`
	Extension string `json:"extension"`
}

type Item struct {
	Code        string                 `json:"code"`
	Description string                 `json:"description,omitempty"`
	Breackpoint int                    `json:"breackpoint"`
	Types       map[int]TypeVariant    `json:"types"` // ex.: {"3":{"file":"...","extension":"png"}}
}

// Repositório agora retorna os itens já no formato "Item"
type Repository interface {
	ActiveItems(ctx context.Context, tenantID int, pageTypes []int) ([]Item, error)
}
