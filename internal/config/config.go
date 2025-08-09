package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	APIKey         string
	AllowedOrigins []string
	HMACSecret     string
	RedisURL       string
	RecentN        int
}

func get(key, def string) string { v := strings.TrimSpace(os.Getenv(key)); if v == "" { return def }; return v }

func parseOrigins() []string {
	v := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if v == "" { return []string{"https://conexaoguarulhos.com.br", "https://www.conexaoguarulhos.com.br", "https://gazetadeosasco.com.br", "https://www.gazetadeosasco.com.br"} }
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts { p = strings.TrimSpace(p); if p != "" { out = append(out, p) } }
	return out
}

func Load() Config {
	recentN, _ := strconv.Atoi(get("RECENT_N", "5"))
	return Config{
		Port:           get("PORT", "8080"),
		APIKey:         get("API_KEY", "changeme"),
		AllowedOrigins: parseOrigins(),
		HMACSecret:     get("HMAC_SECRET", "super-secret"),
		RedisURL:       os.Getenv("REDIS_URL"),
		RecentN:        recentN,
	}
}
