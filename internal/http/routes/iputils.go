package routes

import (
	"net"
	"net/http"
	"strings"
)

func preferIP(r *http.Request) (string, string) {
	// 0) Se Cloudflare Pseudo-IPv4 estiver ativo, priorize esse cabeçalho
	if p4 := strings.TrimSpace(r.Header.Get("CF-Pseudo-IPv4")); p4 != "" {
		if ip := net.ParseIP(p4); ip != nil && ip.To4() != nil {
			// ip preferido = Pseudo IPv4, raw = CF-Connecting-IP ou RemoteAddr
			raw := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
			if raw == "" { raw = r.RemoteAddr }
			return ip.String(), raw
		}
	}

	// 1) CF-Connecting-IP como base (pode ser v6)
	raw := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
	if raw == "" {
		// 2) Primeiro da X-Forwarded-For
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				raw = strings.TrimSpace(parts[0])
			}
		}
	}
	// 3) Fallback RemoteAddr
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
	// se não achou, tenta converter raw para v4
	if v4 == "" {
		if ip := net.ParseIP(raw); ip != nil && ip.To4() != nil {
			v4 = ip.String()
		}
	}

	if v4 != "" {
		return v4, raw // preferido = IPv4, raw = original (pode ser v6)
	}
	return raw, raw // só IPv6 disponível
}
