package tenant

import (
	"log"
	"strings"
)

type Tenant struct {
	ID     int
	Portal string
	AdsURL string
	Static string
}

var tenants = map[string]Tenant{
	"ads.conexao.gru.br":   {ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://ads.conexao.gru.br",   Static: "https://static.conexao.gru.br"},
	"conexao.gru.br":       {ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://ads.conexao.gru.br",   Static: "https://static.conexao.gru.br"},
	"ads.gazeta.osasco.br": {ID: 2, Portal: "gazetadeosasco.com.br",   AdsURL: "https://ads.gazeta.osasco.br", Static: "https://static.gazeta.osasco.br"},
	"gazeta.osasco.br":     {ID: 2, Portal: "gazetadeosasco.com.br",   AdsURL: "https://ads.gazeta.osasco.br", Static: "https://static.gazeta.osasco.br"},
	"ads.diario.osasco.br": {ID: 3, Portal: "diariodeosasco.com.br",   AdsURL: "https://ads.diario.osasco.br", Static: "https://static.diario.osasco.br"},
	"diario.osasco.br":     {ID: 3, Portal: "diariodeosasco.com.br",   AdsURL: "https://ads.diario.osasco.br", Static: "https://static.diario.osasco.br"},
}

var Default = Tenant{ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://ads.conexao.gru.br", Static: "https://static.conexao.gru.br"}

func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	// se vier múltiplos no X-Forwarded-Host: "a, b", pega o primeiro
	if i := strings.IndexByte(h, ','); i > -1 {
		h = h[:i]
	}
	// remove porta (ex.: host:443)
	if i := strings.IndexByte(h, ':'); i > -1 {
		h = h[:i]
	}
	// remove ponto final
	return strings.TrimSuffix(h, ".")
}

func FromRequestHost(host, forwarded string) Tenant {
	log.Printf("[tenant] CHEGAMOS AQUI %q", host)
	raw := strings.TrimSpace(forwarded)
	if raw == "" {
		raw = host
	}
	norm := normalizeHost(raw)
	log.Printf("[tenant] received host=%q forwarded=%q normalized=%q", host, forwarded, norm)

	// 1) match exato
	if t, ok := tenants[norm]; ok {
		log.Printf("[tenant] match=exact key=%q tenant_id=%d portal=%s", norm, t.ID, t.Portal)
		return t
	}

	// 2) fallback: remover o primeiro label até achar
	base := norm
	for {
		if dot := strings.IndexByte(base, '.'); dot > 0 {
			base = base[dot+1:]
			if t, ok := tenants[base]; ok {
				log.Printf("[tenant] match=fallback base=%q from=%q tenant_id=%d portal=%s", base, norm, t.ID, t.Portal)
				return t
			}
		} else {
			break
		}
	}

	// 3) default
	log.Printf("[tenant] match=default tenant_id=%d portal=%s", Default.ID, Default.Portal)
	return Default
}
