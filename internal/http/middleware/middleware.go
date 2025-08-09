package middleware

import (
	"net/http"
)

func CORS(allowed []string) func(http.Handler) http.Handler {
	allow := map[string]struct{}{}
	for _, o := range allowed { allow[o] = struct{}{} }
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if o := r.Header.Get("Origin"); o != "" {
				if _, ok := allow[o]; ok { w.Header().Set("Access-Control-Allow-Origin", o); w.Header().Set("Vary","Origin"); w.Header().Set("Access-Control-Allow-Credentials","true"); w.Header().Set("Access-Control-Allow-Headers","Content-Type, Authorization") }
			}
			if r.Method == http.MethodOptions { w.WriteHeader(204); return }
			next.ServeHTTP(w, r)
		})
	}
}

func OnlyGET() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet { http.Error(w, "Método não permitido", 405); return }
			next.ServeHTTP(w, r)
		})
	}
}
