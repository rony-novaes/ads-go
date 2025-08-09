package ads

import "context"

type Repository interface {
	ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]ResponseAd, error)
}
