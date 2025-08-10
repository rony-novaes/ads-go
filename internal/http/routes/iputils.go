package routes

import (
	"net"
	"net/http"
	"strings"
)

// preferIP retorna (ipPreferido, ipBruto).
func preferIP(r *http.Request) (string, string) {
	raw := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
	if raw == "" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				raw = strings.TrimSpace(parts[0])
			}
		}
	}
	if raw == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil && host != "" {
			raw = host
		} else {
			raw = r.RemoteAddr
		}
	}

	// tenta IPv4 no XFF
	var v4 string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, p := range strings.Split(xff, ",") {
			ip := net.ParseIP(strings.TrimSpace(p))
			if ip != nil && ip.To4() != nil {
				v4 = ip.String()
				break
			}
		}
	}
	if v4 == "" {
		if ip := net.ParseIP(raw); ip != nil && ip.To4() != nil {
			v4 = ip.String()
		}
	}

	if v4 != "" {
		return v4, raw
	}
	return raw, raw
}
