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

// ActiveByTypes retorna anúncios ativos no formato esperado pelo front (ResponseAd).
// Lê diretamente das colunas: code, description, breackpoint, types(JSON).
// O campo "types" no banco é um objeto do tipo:
//   { "3": { "file": "389d...b917cd", "extension": "png" }, "1": {...} }
// ... imports e struct iguais ...

func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, pageTypes []int) ([]ResponseAd, error) {
	want := make(map[int]struct{}, len(pageTypes))
	for _, t := range pageTypes { want[t] = struct{}{} }

	const q = `
		SELECT
			uuid,                  -- 0  <<<<<<<<<<
			code,                  -- 1
			COALESCE(description,''), -- 2
			breackpoint,           -- 3
			types,                 -- 4 (JSON)
			status,                -- 5
			started_at,            -- 6
			validate_at,           -- 7
			deleted_at             -- 8
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
	out := make([]ResponseAd, 0, 128)

	for rows.Next() {
		var (
			uuid, code, desc string
			bp               int
			typesJSON        sql.NullString
			status           int8
			startedAt, validateAt, deletedAt sql.NullTime
		)
		if err := rows.Scan(&uuid, &code, &desc, &bp, &typesJSON, &status, &startedAt, &validateAt, &deletedAt); err != nil {
			return nil, err
		}
		if status != 1 || deletedAt.Valid || (startedAt.Valid && startedAt.Time.After(now)) || (validateAt.Valid && !validateAt.Time.After(now)) {
			continue
		}
		if !typesJSON.Valid || strings.TrimSpace(typesJSON.String) == "" {
			continue
		}

		var raw map[string]struct {
			File      string `json:"file"`
			Extension string `json:"extension"`
		}
		if err := json.Unmarshal([]byte(typesJSON.String), &raw); err != nil {
			continue
		}

		typed := make(map[int]AdTypeVariant, len(raw))
		for k, v := range raw {
			tp, err := strconv.Atoi(k); if err != nil { continue }
			if len(want) > 0 {
				if _, ok := want[tp]; !ok { continue }
			}
			typed[tp] = AdTypeVariant{
				File: strings.TrimSpace(v.File), Extension: strings.TrimSpace(v.Extension),
			}
		}
		if len(typed) == 0 { continue }

		out = append(out, ResponseAd{
			Breackpoint: bp,
			Code:        code,
			Description: desc,
			Types:       typed,
			UUID:        uuid, // <<<<<<<<<< usado para log
		})
	}
	return out, rows.Err()
}
