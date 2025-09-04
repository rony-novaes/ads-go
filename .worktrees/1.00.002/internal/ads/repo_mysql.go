package ads

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type mysqlRepo struct{ db *sql.DB }

func NewMySQLRepo(db *sql.DB) Repository { return &mysqlRepo{db: db} }

// ActiveByTypes busca no MySQL os anúncios ativos do tenant, respeitando janela de tempo,
// status=1 e deleted_at IS NULL. A coluna "types" (JSON) é expandida e filtrada
// pelos types solicitados.
func (r *mysqlRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	if len(types) == 0 {
		return []Ad{}, nil
	}

	// Conjunto de filtro para O(1)
	want := make(map[int]struct{}, len(types))
	for _, t := range types {
		want[t] = struct{}{}
	}

	// Seleciona apenas o necessário do banco
	// Observação: validate_at pode ser NULL (sem expiração).
	// started_at padrão "0000-00-00 00:00:00" no schema — tratamos como <= now.
	const q = `
		SELECT
			uuid,               -- 0
			redirect,           -- 1
			file_extension,     -- 2
			types,              -- 3 (JSON, ex.: [1,2,4])
			status,             -- 4 (tinyint; 1 = ativo)
			started_at,         -- 5
			validate_at,        -- 6 (NULL = sem expiração)
			deleted_at          -- 7 (NULL = não deletado)
		FROM ads
		WHERE tenant_id = ?
		  AND (deleted_at IS NULL)
		  AND (status = 1)
		  AND (started_at IS NULL OR started_at <= NOW())
		  AND (validate_at IS NULL OR validate_at > NOW())
	`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	out := make([]Ad, 0, 128)

	for rows.Next() {
		var (
			uuid, redirect, fileExt string
			typesJSON               sql.NullString
			status                  int8
			startedAt, validateAt   sql.NullTime
			deletedAt               sql.NullTime
		)

		if err := rows.Scan(
			&uuid,
			&redirect,
			&fileExt,
			&typesJSON,
			&status,
			&startedAt,
			&validateAt,
			&deletedAt,
		); err != nil {
			return nil, err
		}

		// Safety extra (já filtramos no WHERE)
		if status != 1 {
			continue
		}
		if deletedAt.Valid {
			continue
		}
		if startedAt.Valid && startedAt.Time.After(now) {
			continue
		}
		if validateAt.Valid && !validateAt.Time.After(now) {
			continue
		}

		// Parse do JSON "types" (aceita array de ints)
		var itemTypes []int
		if typesJSON.Valid && strings.TrimSpace(typesJSON.String) != "" {
			if err := json.Unmarshal([]byte(typesJSON.String), &itemTypes); err != nil {
				// Se o JSON estiver inválido, ignore este anúncio
				continue
			}
		} else {
			// Sem types definidos → não entrega em lugar nenhum
			continue
		}

		// Cria itens por tipo solicitado
		for _, t := range itemTypes {
			if _, ok := want[t]; !ok {
				continue
			}
			out = append(out, Ad{
				ID:        uuid,
				Type:      t,
				TargetURL: redirect,
				FileExt:   fileExt,
				Active:    true,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Debug helper opcional (não usado): imprime a lista para inspeção
func debugAds(list []Ad) string {
	var b strings.Builder
	for i, a := range list {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "{uuid:%s type:%d ext:%s}", a.ID, a.Type, a.FileExt)
	}
	return b.String()
}
