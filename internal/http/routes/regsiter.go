package routes

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"ads-go/internal/ads"
	"ads-go/internal/config"
)

func Register(r *chi.Mux, cfg config.Config, rdb *redis.Client, db *sql.DB) {
	repo := ads.NewMySQLRepo(db)

	ad := adsDeps{Cfg: cfg, Repo: repo}
	r.Get("/", ad.AdsRoot)

	sd := shortDeps{Cfg: cfg, Rdb: rdb, DB: db}
	r.Get("/{short}", sd.Short)
}
