package httpserver

import (
	"net/http"
	"time"

	"search-golang/internal/config"
	"search-golang/internal/http/handler"
	"search-golang/internal/http/middleware"
	"search-golang/internal/search"
)

func New(cfg config.Config) *http.Server {
	mux := http.NewServeMux()

	// dependências
	osClient := search.NewClient(cfg.OSHost)

	// handlers
	mux.Handle("/", middleware.WithTenant(
		middleware.WithCORS(http.HandlerFunc(handler.Search(osClient)), cfg.CORSAllowAny),
	))

	return &http.Server{
		Addr:         cfg.BindAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}
