package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"ads-go/internal/ads"
	"ads-go/internal/config"
	"ads-go/internal/tenant"
)

type adsDeps struct {
	Cfg   config.Config
	Repo  ads.Repository
	Cache *ads.Cache // pode manter para futuro; aqui vamos direto do MySQL para manter compatibilidade
}

type nodeResponse struct {
	Ads      []ads.Item `json:"ads"`
	Redirect string     `json:"redirect"`
	Static   string     `json:"static"`
}

// Handler compatível com o Node na raiz "/"
func (d adsDeps) AdsRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))

	// Se quiser, use o query param "type=" para alterar lógica; por enquanto entregamos todos ativos
	items, err := d.Repo.ActiveItems(r.Context(), t.ID, []int{1,2,3,4})
	if err != nil {
		log.Printf("[ads root] mysql err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error":"internal"})
		return
	}
	_ = json.NewEncoder(w).Encode(nodeResponse{
		Ads: items, Redirect: t.AdsURL, Static: t.Static,
	})
}
