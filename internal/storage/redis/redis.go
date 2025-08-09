package redis

import (
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"ads-go/internal/config"
)

// New retorna um client Redis OU nil se REDIS_URL vazio.
func New(cfg config.Config) (*redis.Client, error) {
	url := strings.TrimSpace(cfg.RedisURL)
	if url == "" { return nil, nil }
	if strings.HasPrefix(url, "redis://") || strings.HasPrefix(url, "rediss://") {
		opt, err := redis.ParseURL(url)
		if err != nil { return nil, err }
		return redis.NewClient(opt), nil
	}
	addr := getenv("REDIS_ADDR", "127.0.0.1:6379")
	pass := getenv("REDIS_PASSWORD", "")
	db, _ := strconv.Atoi(getenv("REDIS_DB", "0"))
	return redis.NewClient(&redis.Options{Addr: addr, Password: pass, DB: db}), nil
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(getenvRaw(k)); v != "" { return v }
	return def
}

// pode ser substituído por os.LookupEnv no main; mantido assim para testes
var getenvRaw = func(key string) string { return "" }
