package routes

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"ads-go/internal/ads"
	"ads-go/internal/config"
)

func Register(r *chi.Mux, cfg config.Config, rdb *redis.Client, db *sql.DB) {
	// repo/caches originais seguem intocados (usados por outras rotas internas)
	_ = ads.NewMySQLRepo  // garante link do pacote ads, se usar em outros pontos

	// Raiz "/" no formato do Node
	node := adsNodeDeps{DB: db}
	r.Get("/", node.AdsRoot)

	// Shortlink
	sd := shortDeps{Cfg: cfg, Rdb: rdb, DB: db}
	r.Get("/{short}", sd.Short)
}
