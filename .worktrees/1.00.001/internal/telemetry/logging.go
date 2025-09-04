package telemetry

import (
	"log"
	"time"
)

type Fields map[string]any

func Info(msg string, f Fields) { log.Printf("level=info msg=%q fields=%v", msg, f) }
func Error(msg string, f Fields) { log.Printf("level=error msg=%q fields=%v", msg, f) }
func Now() time.Time { return time.Now() }
