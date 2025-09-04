package main

import (
	"log"

	"search-golang/internal/config"
	"search-golang/internal/httpserver"
)

func main() {
	cfg := config.Load()
	srv := httpserver.New(cfg)
	log.Printf("Search API listening on %s", cfg.BindAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
