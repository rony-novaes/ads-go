package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ads-go/internal/tenant"
)

// Estruturas EXATAMENTE como o front espera (Node)
type nodeTypeVariant struct {
	File      string `json:"file"`
	Extension string `json:"extension"`
}
type nodeItem struct {
	Code        string                       `json:"code"`
	Description string                       `json:"description,omitempty"`
	Breackpoint int                          `json:"breackpoint"`
	Types       map[int]nodeTypeVariant      `json:"types"`
}
type nodeResp struct {
	Ads      []nodeItem `json:"ads"`
	Redirect string     `json:"redirect"`
	Static   string     `json:"static"`
}

type adsNodeDeps struct {
	DB *sql.DB
}

// GET "/"  → JSON idêntico ao Node
func (d adsNodeDeps) AdsRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","application/json")
	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))

	items, err := fetchActiveItems(r.Context(), d.DB, t.ID)
	if err != nil {
		log.Printf("[ads root] mysql err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error":"internal"})
		return
	}
	_ = json.NewEncoder(w).Encode(nodeResp{
		Ads: items, Redirect: t.AdsURL, Static: t.Static,
	})
}

// Consulta simples (status=1, janela válida, não deletado)
func fetchActiveItems(ctx context.Context, db *sql.DB, tenantID int) ([]nodeItem, error) {
	const q = `
		SELECT
			code,                   -- 0
			COALESCE(description,''), -- 1
			breackpoint,            -- 2
			types,                  -- 3 (JSON: {"3":{"file":"...","extension":"png"}})
			status,                 -- 4
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
	rows, err := db.QueryContext(ctx, q, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()

	now := time.Now()
	out := make([]nodeItem, 0, 128)

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
			continue
		}
		typed := make(map[int]nodeTypeVariant, len(raw))
		for k, v := range raw {
			tp, err := strconv.Atoi(k)
			if err != nil { continue }
			typed[tp] = nodeTypeVariant{
				File: strings.TrimSpace(v.File), Extension: strings.TrimSpace(v.Extension),
			}
		}

		out = append(out, nodeItem{
			Code: code, Description: desc, Breackpoint: bp, Types: typed,
		})
	}
	return out, rows.Err()
}
