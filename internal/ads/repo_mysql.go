package ads

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type mysqlRepo struct{ db *sql.DB }

func NewMySQLRepo(db *sql.DB) Repository { return &mysqlRepo{db: db} }

// ActiveItems retorna os anúncios ativos do tenant já no formato esperado pelo front.
func (r *mysqlRepo) ActiveItems(ctx context.Context, tenantID int, pageTypes []int) ([]Item, error) {
	// Se quiser filtrar por "tipo de página" (1..4), use pageTypes; senão ignoramos e devolvemos todos ativos
	const q = `
		SELECT
			code,           -- 0
			COALESCE(description,''), -- 1
			breackpoint,    -- 2
			types,          -- 3 (JSON: {"3":{"file":"...","extension":"png"}})
			status,         -- 4
			started_at,     -- 5
			validate_at,    -- 6
			deleted_at      -- 7
		FROM ads
		WHERE tenant_id = ?
		  AND deleted_at IS NULL
		  AND status = 1
		  AND (started_at IS NULL OR started_at <= NOW())
		  AND (validate_at IS NULL OR validate_at > NOW())
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()

	now := time.Now()
	out := make([]Item, 0, 128)

	for rows.Next() {
		var (
			code, desc string
			bp         int
			typesJSON  sql.NullString
			status     int8
			startedAt, validateAt, deletedAt sql.NullTime
		)
		if err := rows.Scan(&code, &desc, &bp, &typesJSON, &status, &startedAt, &validateAt, &deletedAt); err != nil {
			return nil, err
		}
		// guard rails (já filtrados no WHERE)
		if status != 1 || deletedAt.Valid || (startedAt.Valid && startedAt.Time.After(now)) || (validateAt.Valid && !validateAt.Time.After(now)) {
			continue
		}
		if !typesJSON.Valid || strings.TrimSpace(typesJSON.String) == "" {
			continue
		}

		// Parse do objeto por tipo {"3":{"file":"...","extension":"png"}}
		var raw map[string]struct {
			File      string `json:"file"`
			Extension string `json:"extension"`
		}
		if err := json.Unmarshal([]byte(typesJSON.String), &raw); err != nil {
			continue
		}
		typed := make(map[int]TypeVariant, len(raw))
		for k, v := range raw {
			tp, err := strconv.Atoi(k); if err != nil { continue }
			typed[tp] = TypeVariant{File: strings.TrimSpace(v.File), Extension: strings.TrimSpace(v.Extension)}
		}

		out = append(out, Item{
			Code:        code,
			Description: desc,
			Breackpoint: bp,
			Types:       typed,
		})
	}
	return out, rows.Err()
}
