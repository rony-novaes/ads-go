package ads

import (
	"context"
)

type memoryRepo struct{}

func NewMemoryRepo() Repository {
	return &memoryRepo{}
}

func (r *memoryRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]ResponseAd, error) {
	// Simulação de retorno como se viesse do MySQL
	out := []ResponseAd{
		{
			Breackpoint: 1,
			Code:        "uuid-1-1",
			Description: "Texto opcional",
			Types: map[int]AdTypeVariant{
				1: {File: "file-1-1", Extension: "jpg"},
				2: {File: "file-1-2", Extension: "png"},
			},
		},
		{
			Breackpoint: 2,
			Code:        "uuid-2-1",
			Description: "Banner secundário",
			Types: map[int]AdTypeVariant{
				1: {File: "file-2-1", Extension: "webp"},
			},
		},
	}

	return out, nil
}
