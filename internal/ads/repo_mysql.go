package ads

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
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
func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, pageTypes []int) ([]ResponseAd, error) {
	want := make(map[int]struct{}, len(pageTypes))
	for _, t := range pageTypes {
		want[t] = struct{}{}
	}

	const q = `
		SELECT
			code,                   -- 0
			COALESCE(description,''), -- 1
			breackpoint,            -- 2
			types,                  -- 3 (JSON objeto por tipo)
			status,                 -- 4 (1 = ativo)
			started_at,             -- 5
			validate_at,            -- 6
			deleted_at              -- 7
		FROM ads
		WHERE tenant_id = ?
		  AND deleted_at IS NULL
		  AND status = 1
		  AND (started_at IS NULL OR started_at <= NOW())
		  AND (validate_at IS NULL OR validate_at > NOW())
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		log.Printf("[ads repo] tenant=%d query err: %v", tenantID, err)
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	out := make([]ResponseAd, 0, 128)

	for rows.Next() {
		var (
			code, desc  string
			bp          int
			typesJSON   sql.NullString
			status      int8
			startedAt   sql.NullTime
			validateAt  sql.NullTime
			deletedAt   sql.NullTime
		)
		if err := rows.Scan(&code, &desc, &bp, &typesJSON, &status, &startedAt, &validateAt, &deletedAt); err != nil {
			log.Printf("[ads repo] tenant=%d scan err: %v", tenantID, err)
			return nil, err
		}

		// guard rails (redundante com WHERE, mas seguro)
		if status != 1 || deletedAt.Valid || (startedAt.Valid && startedAt.Time.After(now)) || (validateAt.Valid && !validateAt.Time.After(now)) {
			continue
		}
		if !typesJSON.Valid || strings.TrimSpace(typesJSON.String) == "" {
			continue
	}

		// Parse objeto {"3":{"file":"...","extension":"png"}}
		var raw map[string]struct {
			File      string `json:"file"`
			Extension string `json:"extension"`
		}
		if err := json.Unmarshal([]byte(typesJSON.String), &raw); err != nil {
			// JSON inválido → ignora este anúncio
			continue
		}

		// Converte para map[int]AdTypeVariant aplicando filtro de tipos (se fornecido)
		typed := make(map[int]AdTypeVariant, len(raw))
		for k, v := range raw {
			tp, err := strconv.Atoi(k)
			if err != nil {
				continue
			}
			// Se pageTypes foi passado, filtra; se vazio, aceita todos
			if len(want) > 0 {
				if _, ok := want[tp]; !ok {
					continue
				}
			}
			typed[tp] = AdTypeVariant{
				File:      strings.TrimSpace(v.File),
				Extension: strings.TrimSpace(v.Extension),
			}
		}

		// Se após o filtro não sobrou tipo, pula
		if len(typed) == 0 {
			continue
		}

		out = append(out, ResponseAd{
			Breackpoint: bp,
			Code:        code,
			Description: desc,
			Types:       typed,
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ads repo] tenant=%d rows err: %v", tenantID, err)
		return nil, err
	}

	return out, nil
}
