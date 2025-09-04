package middleware

import (
	"net/http"
	"strings"
)

func WithCORS(next http.Handler, allowAny bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := GetTenant(r)
		origin := r.Header.Get("Origin")

		allow := ""
		if allowAny && origin != "" {
			allow = origin
		} else {
			allow = "https://" + t.Portal
			// se quiser liberar subdomínios confiáveis do portal:
			if origin != "" && strings.HasSuffix(origin, t.Portal) {
				allow = origin
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
