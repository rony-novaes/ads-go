package ads

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type mysqlRepo struct{ db *sql.DB }

func NewMySQLRepo(db *sql.DB) Repository { return &mysqlRepo{db: db} }

// ActiveByTypes busca anúncios ativos do tenant, respeitando janela e status.
// Suporta "types" no formato objeto por tipo:
//   { "3": { "file": "389d...b917cd", "extension": "png" }, "1": {...} }
// (Se vier um array de ints por compat, ignora porque precisamos do file/extension por tipo.)
func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	if len(types) == 0 {
		return []Ad{}, nil
	}
	want := make(map[int]struct{}, len(types))
	for _, t := range types { want[t] = struct{}{} }

	const q = `
		SELECT
			uuid,         -- 0
			redirect,     -- 1
			types,        -- 2 (JSON objeto por tipo)
			status,       -- 3 (1 = ativo)
			started_at,   -- 4
			validate_at,  -- 5
			deleted_at    -- 6
		FROM ads
		WHERE tenant_id = ?
		  AND deleted_at IS NULL
			AND status = 1
		  AND (started_at IS NULL OR started_at <= NOW())
		  AND (validate_at IS NULL OR validate_at > NOW())
	`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()

	now := time.Now()
	out := make([]Ad, 0, 128)

	for rows.Next() {
		var (
			uuid, redirect string
			typesJSON      sql.NullString
			status         int8
			startedAt      sql.NullTime
			validateAt     sql.NullTime
			deletedAt      sql.NullTime
		)
		if err := rows.Scan(&uuid, &redirect, &typesJSON, &status, &startedAt, &validateAt, &deletedAt); err != nil {
			return nil, err
		}

		// safety extra
		if status != 1 || deletedAt.Valid || (startedAt.Valid && startedAt.Time.After(now)) || (validateAt.Valid && !validateAt.Time.After(now)) {
			continue
		}
		if !typesJSON.Valid || strings.TrimSpace(typesJSON.String) == "" {
			continue
		}

		// Esperado: objeto por tipo
		type perType struct {
			File      string `json:"file"`
			Extension string `json:"extension"`
		}
		var obj map[string]perType
		if err := json.Unmarshal([]byte(typesJSON.String), &obj); err != nil {
			// Se vier array antigo, não usamos (faltam file/extension por tipo)
			continue
		}

		for k, v := range obj {
			tp, err := strconv.Atoi(k)
			if err != nil { continue }
			if _, ok := want[tp]; !ok { continue }

			// File e Extension são obrigatórios para montar a imagem no front
			file := strings.TrimSpace(v.File)
			ext  := strings.TrimSpace(v.Extension)
			// se faltar algo, ainda assim entregamos o resto (front pode tratar)
			out = append(out, Ad{
				ID:        uuid,
				Type:      tp,
				TargetURL: redirect,
				File:      file,
				FileExt:   ext,
				Active:    true,
			})
		}
	}
	return out, rows.Err()
}

// (opcional) debug
func debugAds(list []Ad) string {
	var b strings.Builder
	for i, a := range list {
		if i > 0 { b.WriteString(", ") }
		fmt.Fprintf(&b, "{uuid:%s type:%d file:%s ext:%s}", a.ID, a.Type, a.File, a.FileExt)
	}
	return b.String()
}
