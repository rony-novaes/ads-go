package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	BindAddr       string
	OSHost         string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	CORSAllowAny   bool
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func atoiDef(v string, def int) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func Load() Config {
	return Config{
		BindAddr:     getenv("BIND_ADDR", ":8080"),
		OSHost:       getenv("OS_HOST", "http://opensearch:9200"),
		ReadTimeout:  time.Duration(atoiDef(getenv("READ_TIMEOUT_MS", "3000"), 3000)) * time.Millisecond,
		WriteTimeout: time.Duration(atoiDef(getenv("WRITE_TIMEOUT_MS", "3000"), 3000)) * time.Millisecond,
		CORSAllowAny: getenv("CORS_ALLOW_ANY", "0") == "1",
	}
}
