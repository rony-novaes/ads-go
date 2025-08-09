package routes

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"ads-go/internal/ads"
	"ads-go/internal/config"
	"ads-go/internal/tenant"
)

type RecentStore interface {
	Get(r *http.Request, tenantID int, userKey string, n int) ([]string, error)
	Push(r *http.Request, tenantID int, userKey string, ids []string, n int) error
}

func Register(mux *chi.Mux, cfg config.Config, rdb *redis.Client, db *sql.DB) {
	// Repo (MySQL)
	repo := ads.NewMySQLRepo(db)

	// Cache em memória e refresher a cada 5 minutos
	cache := ads.NewCache()
	tenantIDs := []int{tenant.Default.ID, 2} // adicione aqui os tenants que você tem
	ctx := context.Background()
	cache.StartRefresher(ctx, repo, tenantIDs, 5*time.Minute, func(string, ...any) {})

	// Recent: Redis se houver, senão memória
	var recent RecentStore
	if rdb != nil {
		recent = NewRedisRecent(rdb)
	} else {
		recent = NewMemoryRecent()
	}

	ad := adsDeps{Cfg: cfg, Recent: recent, Repo: repo, Cache: cache}
	s := shortDeps{Cfg: cfg, Rdb: rdb, DB: db} // <- adicionamos DB aqui

	// Raiz: redireciona para o portal do tenant
	// mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
	// 	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))
	// 	http.Redirect(w, r, "https://"+t.Portal, http.StatusMovedPermanently)
	// })

	mux.Get("/", ad.AdsHandler)
	// mux.Get("/cache-ads-clear-rdo-sm", ad.CacheClear) // compatível mesmo sem Redis

	// REMOVIDO: rota /c de clique (não é necessária)
	// mux.Get("/c", c.Click)

	// Shortlink por último
	mux.Get("/{short}", s.Short)
}
