package routes

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ads-go/internal/ads"
	"ads-go/internal/config"
	"ads-go/internal/tenant"
)

type adsDeps struct {
	Cfg    config.Config
	Recent RecentStore
	Repo   ads.Repository
	Cache  *ads.Cache
	DB     *sql.DB // <— adicionar DB para logar views
}

var adTypeConfig = map[string]map[int]int{
	"news": {1: 1, 2: 1, 3: 2, 4: -1},
	"home": {1: 1, 2: 1, 3: 1, 4: -1},
}

func (d adsDeps) AdsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ptype := r.URL.Query().Get("type")
	rulesBase, ok := adTypeConfig[ptype]
	if !ok {
		http.Error(w, `{"error":"type inválido"}`, http.StatusBadRequest)
		return
	}
	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))

	// monta regras finais (permite sobrescrever quando lim = -1)
	final := map[int]int{}
	for tp, lim := range rulesBase {
		if lim == -1 {
			if v, _ := strconv.Atoi(r.URL.Query().Get(fmt.Sprintf("ad_type_%d", tp))); v > 0 {
				final[tp] = v
			}
		} else {
			final[tp] = lim
		}
	}

	// pool base (cache)
	all := d.Cache.Get(t.ID)

	// fallback: se cache vier vazio, busca direto no MySQL e popula cache
	if len(all) == 0 {
		types := make([]int, 0, len(final))
		for tp := range final {
			types = append(types, tp)
		}
		log.Printf("[ads handler] cache vazio tenant=%d -> buscando MySQL types=%v", t.ID, types)
		list, err := d.Repo.ActiveByTypes(r.Context(), t.ID, types)
		if err == nil && len(list) > 0 {
			all = list
			d.Cache.Set(t.ID, list)
		} else if err != nil {
			log.Printf("[ads handler] erro MySQL tenant=%d: %v", t.ID, err)
		} else {
			log.Printf("[ads handler] MySQL retornou vazio tenant=%d", t.ID)
		}
	}

	uk := userKey(r)
	recent, _ := d.Recent.Get(r, t.ID, uk, d.Cfg.RecentN)
	recentSet := make(map[string]struct{}, len(recent))
	for _, code := range recent {
		recentSet[code] = struct{}{}
	}

	pool := filterByTypes(all, final)                 // filtra por presença de tipos solicitados
	pool = avoidRecent(pool, recentSet)               // evita recentemente exibidos (por code)
	sel := ads.FilterByRules(ads.Shuffle(pool), final) // aplica cotas por tipo

	log.Printf("[ads handler] tenant=%d type=%s pool=%d recent=%d selected=%d", t.ID, ptype, len(pool), len(recent), len(sel))

	// Salvar LOG de view (TYPE = 1) para cada anúncio entregue
	// Salvar LOG de view (TYPE = 1) para cada anúncio entregue
	if d.DB != nil && len(sel) > 0 {
		ip := userIP(r)
		ua := r.UserAgent()
		ref := r.Referer()
		for _, a := range sel {
			if a.UUID == "" {
				log.Printf("[ads view] skip sem UUID code=%s tenant=%d", a.Code, t.ID)
				continue
			}
			if err := salvarView(d.DB, a.UUID, t.ID, 1, ip, ua, ref); err != nil {
				log.Printf("[ads view] erro salvar uuid=%s tenant=%d: %v", a.UUID, t.ID, err)
			}
		}
	}

	if len(sel) > 0 {
		// guarda os "codes" exibidos para o usuário
		ids := make([]string, 0, len(sel))
		for _, a := range sel {
			ids = append(ids, a.Code)
		}
		_ = d.Recent.Push(r, t.ID, uk, ids, d.Cfg.RecentN)
	}

	_ = json.NewEncoder(w).Encode(ads.AdsResponse{
		Ads:      sel,
		Redirect: t.AdsURL,
		Static:   t.Static,
	})
}

// ---- helpers (com ResponseAd) ----

func filterByTypes(list []ads.ResponseAd, rules map[int]int) []ads.ResponseAd {
	if len(rules) == 0 {
		return list
	}
	set := map[int]struct{}{}
	for tp := range rules {
		set[tp] = struct{}{}
	}
	out := make([]ads.ResponseAd, 0, len(list))
	for _, a := range list {
		if len(a.Types) == 0 {
			continue
		}
		keep := false
		for tp := range a.Types {
			if _, ok := set[tp]; ok {
				keep = true
				break
			}
		}
		if keep {
			out = append(out, a)
		}
	}
	return out
}

func avoidRecent(in []ads.ResponseAd, recent map[string]struct{}) []ads.ResponseAd {
	if len(recent) == 0 {
		return in
	}
	keep := make([]ads.ResponseAd, 0, len(in))
	for _, a := range in {
		if _, ok := recent[a.Code]; !ok {
			keep = append(keep, a)
		}
	}
	if len(keep) == 0 {
		return in
	}
	return keep
}

func userIP(r *http.Request) string {
	if v := r.Header.Get("cf-connecting-ip"); v != "" {
		return v
	}
	if v := r.Header.Get("x-forwarded-for"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func userKey(r *http.Request) string {
	ip := strings.TrimSpace(userIP(r))
	ua := strings.TrimSpace(r.UserAgent())
	sum := sha256.Sum256([]byte(ip + "|" + ua))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ---- persistência de logs ----

// salvarView: insere uma linha no ads_logs com TYPE = 1 (view de JSON).
// Aqui uso a coluna `code`; se sua tabela não tiver `code`, troque para `uuid` conforme seu schema.
func salvarView(db *sql.DB, uuid string, tenantID int, typ int, ip, ua, ref string) error {
	const q = `
		INSERT INTO ads_logs (uuid, tenant_id, type, ip, user_agent, referer, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(q, uuid, tenantID, typ, ip, ua, ref, time.Now())
	return err
}

