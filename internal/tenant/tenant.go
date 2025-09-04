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

package tenant

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

type Tenant struct {
	ID     int
	Portal string
	AdsURL string
	Static string
}

// (mantém seu Default como já está declarado no arquivo)
var Default = Tenant{ID: 1, Portal: "https://conexaoguarulhos.com.br", Search: "https://pesquisa.conexaoguarulhos.com.br", Static: "https://static.conexao.gru.br"}

// --------- ALTERAÇÃO AQUI ---------
var tenants map[string]Tenant

func init() {
	tenants = loadTenantsFromJSON()
}

func loadTenantsFromJSON() map[string]Tenant {
	// fallback: seu mapa atual embutido
	fallback := map[string]Tenant{
		"pesquisa.conexaoguarulhos.com.br":	{ID: 1, Portal: "https://conexaoguarulhos.com.br", Search: "https://pesquisa.conexaoguarulhos.com.br",	Static: "https://static.conexaoguarulhos.com.br"},
		"pesquisa.gazetadeosasco.com.br":	{ID: 2, Portal: "https://gazetadeosasco.com.br",   Search: "https://pesquisa.gazetadeosasco.com.br",	Static: "https://static.gazetadeosasco.com.br"},
		"pesquisa.diariodeosasco.com.br":	{ID: 3, Portal: "https://diariodeosasco.com.br",   Search: "https://pesquisa.diariodeosasco.com.br",	Static: "https://static.diariodeosasco.com.br"},
	}

	path := strings.TrimSpace(os.Getenv("TENANT_FILE"))
	if path == "" {
		path = "tenant.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[tenant] %s não encontrado (%v); usando mapa embutido", path, err)
		return fallback
	}

	var m map[string]Tenant
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("[tenant] erro ao parsear %s (%v); usando mapa embutido", path, err)
		return fallback
	}

	if len(m) == 0 {
		log.Printf("[tenant] %s carregado, mas sem entradas; usando mapa embutido", path)
		return fallback
	}

	log.Printf("[tenant] carregados %d tenants de %s", len(m), path)
	return m
}

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
