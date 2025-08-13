package tenant

import "strings"

type Tenant struct {
	ID     int
	Portal string
	AdsURL string
	Static string
}

var tenants = map[string]Tenant{
	"ads.conexao.gru.br": {ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://ads.conexao.gru.br", Static: "https://static.conexao.gru.br"},
	"gazeta.osasco.br": {ID: 2, Portal: "gazetadeosasco.com.br", AdsURL: "https://ads.gazeta.osasco.br", Static: "https://static.gazeta.osasco.br"},
	"diario.osasco.br": {ID: 2, Portal: "diariodeosasco.com.br", AdsURL: "https://ads.diario.osasco.br", Static: "https://static.diario.osasco.br"}
}

var Default = Tenant{ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://ads.conexao.gru.br", Static: "https://static.conexao.gru.br"}

func FromRequestHost(host, forwarded string) Tenant {
	h := strings.ToLower(strings.TrimSpace(forwarded))
	if h == "" { h = strings.ToLower(strings.TrimSpace(host)) }
	if t, ok := tenants[h]; ok { return t }
	return Default
}
