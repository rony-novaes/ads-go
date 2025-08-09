package ads

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type mysqlRepo struct { db *sql.DB }

func NewMySQLRepo(db *sql.DB) Repository { return &mysqlRepo{db: db} }

// ActiveByTypes busca anúncios ativos do tenant, filtrando pelos tipos informados.
func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	if len(types) == 0 { return []Ad{}, nil }
	placeholders := make([]string, len(types))
	args := make([]any, 0, len(types)+1)
	for i, t := range types { placeholders[i] = "?"; args = append(args, t) }
	args = append(args, tenantID)

	q := fmt.Sprintf(`
		SELECT id, type, image_url, target_url, weight, active
		FROM ads
		WHERE deleted_at IS NULL
		  AND type IN (%s)
		  AND tenant_id = ?
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Ad
	for rows.Next() {
		var a Ad
		if err := rows.Scan(&a.ID, &a.Type, &a.ImageURL, &a.TargetURL, &a.Weight, &a.Active); err != nil { return nil, err }
		out = append(out, a)
	}
	return out, rows.Err()
}
