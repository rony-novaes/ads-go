package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"ads-go/internal/config"
	appmw "ads-go/internal/http/middleware"
	"ads-go/internal/http/routes"
	mysqldb "ads-go/internal/storage/mysql"
	redisc "ads-go/internal/storage/redis"
)

func main() {
	// Carrega .env (best-effort)
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := config.Load()

	// MySQL obrigatório
	db, err := mysqldb.Open()
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	if err := mysqldb.Ping(db); err != nil {
		log.Fatalf("mysql ping: %v", err)
	}

	// Redis OPCIONAL (usa a assinatura do pacote do projeto)
	rdb, err := redisc.New(cfg)
	if err != nil {
		log.Printf("redis init: %v (seguindo sem redis)", err)
		rdb = nil
	} else if rdb != nil {
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Printf("redis indisponível: %v (seguindo sem redis)", err)
			rdb = nil
		}
	}

	// Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(appmw.CORS(cfg.AllowedOrigins)) // <- assinatura correta
	r.Use(appmw.OnlyGET())

	// Registro de rotas (assinatura correta do projeto)
	routes.Register(r, cfg, rdb, db)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shut down cleanly")
}
