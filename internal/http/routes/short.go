package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"ads-go/internal/config"
	"ads-go/internal/tenant"
)

func preferIP(r *http.Request) (string, string) {
	// 1) CF-Connecting-IP vence
	raw := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
	if raw == "" {
		// 2) Primeiro da X-Forwarded-For
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				raw = strings.TrimSpace(parts[0])
			}
		}
	}
	// 3) Fallback para RemoteAddr
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil && host != "" {
			raw = host
		} else {
			raw = r.RemoteAddr
		}
	}

	// ipPreferido: tenta achar um IPv4 em XFF
	var v4 string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, p := range strings.Split(xff, ",") {
			ip := net.ParseIP(strings.TrimSpace(p))
			if ip != nil && ip.To4() != nil {
				v4 = ip.String()
				break
			}
		}
	}
	// Se não achou v4 no XFF, tenta converter o raw para v4-mapeado
	if v4 == "" {
		if ip := net.ParseIP(raw); ip != nil && ip.To4() != nil {
			v4 = ip.String()
		}
	}

	// ipPreferido = v4 se existir; senão o raw (pode ser IPv6)
	if v4 != "" {
		return v4, raw
	}
	return raw, raw
}

type shortDeps struct {
	Cfg config.Config
	Rdb *redis.Client
	DB  *sql.DB
}

func (d shortDeps) getKey(t tenant.Tenant, short string) string {
	return "cg_" + strconv.Itoa(t.ID) + "_" + strings.ToLower(short)
}

func (d shortDeps) nfKey(cacheKey string) string {
	return "nf:" + cacheKey // negative cache key
}

func (d shortDeps) lookupShort(w http.ResponseWriter, r *http.Request) (string, string, error) {
	short := chi.URLParam(r, "short")
	if short == "" {
		return "", "", errors.New("empty")
	}
	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))
	cacheKey := d.getKey(t, short)

	// 0) Negative cache (evita bater no MySQL repetidamente por 4 minutos)
	if d.Rdb != nil {
		if _, err := d.Rdb.Get(r.Context(), d.nfKey(cacheKey)).Result(); err == nil {
			// marcado como não encontrado recentemente
			return "", "", errors.New("not found")
		}
	}

	// 1) Tenta Redis (cache positivo)
	if d.Rdb != nil {
		if raw, err := d.Rdb.Get(r.Context(), cacheKey).Result(); err == nil {
			var v struct{ URL, UUID string }
			if json.Unmarshal([]byte(raw), &v) == nil {
				return v.URL, v.UUID, nil
			}
		}
	}

	// 2) Busca no MySQL
	if d.DB != nil {
		if url, uuid, id, ok, err := fetchShortFromMySQL(r.Context(), d.DB, t.ID, short); err == nil && ok {
			_ = id // evitar "declared and not used" até você usar o id
			// 2.a) Achou: coloca no Redis (cache positivo) e retorna
			if d.Rdb != nil {
				b, _ := json.Marshal(map[string]string{"URL": url, "UUID": uuid})
				_ = d.Rdb.Set(r.Context(), cacheKey, string(b), 24*time.Hour).Err()
				// garante que a flag negativa não atrapalhe um hit recém inserido
				_ = d.Rdb.Del(r.Context(), d.nfKey(cacheKey)).Err()
			}
			return url, uuid, nil
		} else if err != nil {
			// erro real de MySQL — loga e não seta negative cache (para não esconder problema)
			log.Printf("short mysql error: %v", err)
			return "", "", err
		}
	}

	// 3) Não achou em lugar nenhum: negative cache por 4 minutos e retorna 404
	if d.Rdb != nil {
		_ = d.Rdb.Set(r.Context(), d.nfKey(cacheKey), "1", 4*time.Minute).Err()
	}
	return "", "", errors.New("not found")
}

func (d shortDeps) Short(w http.ResponseWriter, r *http.Request) {
    t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))

    redir, uuid, err := d.lookupShort(w, r)
    if err != nil {
        http.Redirect(w, r, "https://"+t.Portal+"?short_error=404", http.StatusFound)
        return
    }

    // pega IP preferido e raw
    ipPref, ipRaw := preferIP(r)

    // Salva o clique aqui mesmo — TYPE = 2
    if d.DB != nil {
        if err := salvarClick(d.DB, uuid, t.ID, 2, ipPref, ipRaw, r.UserAgent(), r.Referer()); err != nil {
            log.Printf("short click save error: %v", err)
        }
    }

    w.Header().Set("Cache-Control", "no-store")
    http.Redirect(w, r, redir, http.StatusFound)
}

// --- helpers ---

func fetchShortFromMySQL(ctx context.Context, db *sql.DB, tenantID int, short string) (url string, uuid string, id int, ok bool, err error) {
	const q = `
		SELECT redirect, uuid, id
		FROM ads
		WHERE tenant_id = ? AND code = ? AND deleted_at IS NULL
		LIMIT 1
	`
	err = db.QueryRowContext(ctx, q, tenantID, short).Scan(&url, &uuid, &id)
	if err == sql.ErrNoRows {
		return "", "", 0, false, nil
	}
	if err != nil {
		return "", "", 0, false, err
	}
	return url, uuid, id, true, nil
}

func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("cf-connecting-ip")); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("x-forwarded-for"); fwd != "" {
		ip := strings.TrimSpace(strings.Split(fwd, ",")[0])
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func salvarClick(db *sql.DB, uuid string, tenantID int, typ int, ip, ipRaw, ua, ref string) error {
    const q = `
        INSERT INTO ads_logs (uuid, tenant_id, type, ip, ip_raw, user_agent, referer, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `
    _, err := db.Exec(q, uuid, tenantID, typ, ip, ipRaw, ua, ref, time.Now())
    return err
}