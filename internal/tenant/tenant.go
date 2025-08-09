package tenant

import "strings"

type Tenant struct {
	ID     int
	Portal string
	AdsURL string
	Static string
}

var tenants = map[string]Tenant{
	"conexaoguarulhos.com.br": {ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://conexaoguarulhos.com.br/ads", Static: "https://static.conexaoguarulhos.com.br"},
	"www.conexaoguarulhos.com.br": {ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://conexaoguarulhos.com.br/ads", Static: "https://static.conexaoguarulhos.com.br"},
	"gazetadeosasco.com.br": {ID: 2, Portal: "gazetadeosasco.com.br", AdsURL: "https://gazetadeosasco.com.br/ads", Static: "https://static.gazetadeosasco.com.br"},
	"www.gazetadeosasco.com.br": {ID: 2, Portal: "gazetadeosasco.com.br", AdsURL: "https://gazetadeosasco.com.br/ads", Static: "https://static.gazetadeosasco.com.br"},
}

var Default = Tenant{ID: 1, Portal: "conexaoguarulhos.com.br", AdsURL: "https://conexaoguarulhos.com.br/ads", Static: "https://static.conexaoguarulhos.com.br"}

func FromRequestHost(host, forwarded string) Tenant {
	h := strings.ToLower(strings.TrimSpace(forwarded))
	if h == "" { h = strings.ToLower(strings.TrimSpace(host)) }
	if t, ok := tenants[h]; ok { return t }
	return Default
}
