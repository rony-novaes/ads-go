package ads

import (
	"context"
	"fmt"
)

// memoryRepo: fonte fake só para desenvolvimento/testes.
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
				TargetURL: "",
				File:      fmt.Sprintf("file-%d-%d", tp, i+1),
				FileExt:   ext,
				Active:    true,
			})
		}
	}
	return out, nil
}
