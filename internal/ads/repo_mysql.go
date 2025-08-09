package ads

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type mysqlRepo struct{ db *sql.DB }

func NewMySQLRepo(db *sql.DB) Repository { return &mysqlRepo{db: db} }

// ActiveByTypes busca anúncios ativos do tenant, respeitando janela e status.
// Suporta "types" no formato objeto por tipo:
//   { "3": { "file": "389d...b917cd", "extension": "png" }, "1": {...} }
func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	start := time.Now()
	if len(types) == 0 {
		log.Printf("[ads repo] tenant=%d types=[] -> vazio (0ms)", tenantID)
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
	if err != nil {
		log.Printf("[ads repo] tenant=%d query err: %v", tenantID, err)
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	out := make([]Ad, 0, 128)
	var scanned, expanded int

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
			log.Printf("[ads repo] tenant=%d scan err: %v", tenantID, err)
			return nil, err
		}
		scanned++

		// safety extra
		if status != 1 || deletedAt.Valid || (startedAt.Valid && startedAt.Time.After(now)) || (validateAt.Valid && !validateAt.Time.After(now)) {
			continue
		}
		if !typesJSON.Valid || strings.TrimSpace(typesJSON.String) == "" {
			continue
		}

		// objeto por tipo
		type perType struct {
			File      string `json:"file"`
			Extension string `json:"extension"`
		}
		var obj map[string]perType
		if err := json.Unmarshal([]byte(typesJSON.String), &obj); err != nil {
			// se não for objeto (ex.: array antigo), ignore
			continue
		}

		for k, v := range obj {
			tp, err := strconv.Atoi(k)
			if err != nil { continue }
			if _, ok := want[tp]; !ok { continue }

			out = append(out, Ad{
				ID:        uuid,
				Type:      tp,
				TargetURL: redirect,
				File:      strings.TrimSpace(v.File),
				FileExt:   strings.TrimSpace(v.Extension),
				Active:    true,
			})
			expanded++
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ads repo] tenant=%d rows err: %v", tenantID, err)
		return nil, err
	}

	dur := time.Since(start).Milliseconds()
	log.Printf("[ads repo] tenant=%d scanned=%d expanded=%d types=%v dur=%dms", tenantID, scanned, expanded, mapKeys(want), dur)
	return out, nil
}

// util para logar os types pedidos
func mapKeys(m map[int]struct{}) []int {
	ks := make([]int, 0, len(m))
	for k := range m { ks = append(ks, k) }
	return ks
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
