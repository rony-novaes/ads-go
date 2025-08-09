package ads

import (
	"context"
	"fmt"
)

// memoryRepo: fonte fake só para desenvolvimento/testes.
// Gera alguns anúncios por tipo solicitado, com extensões de imagem variadas.
type memoryRepo struct{}

func NewMemoryRepo() Repository { return &memoryRepo{} }

func (r *memoryRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	exts := []string{"jpg", "png", "webp"}
	out := make([]Ad, 0, len(types)*len(exts))

	for _, tp := range types {
		for i, ext := range exts {
			out = append(out, Ad{
				ID:        fmt.Sprintf("uuid-%d-%d", tp, i+1),
				Type:      tp,
				TargetURL: "https://conexaoguarulhos.com.br",
				FileExt:   ext,
				Active:    true,
			})
		}
	}

	return out, nil
}
