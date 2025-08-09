package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ads-go/internal/ads"
	"ads-go/internal/config"
	"ads-go/internal/tenant"
)

type adsDeps struct {
	Cfg    config.Config
	Recent RecentStore
	Repo   ads.Repository
	Cache  *ads.Cache
}

type adsResponse struct { Ads []ads.Ad `json:"ads"`; Redirect string `json:"redirect"`; Static string `json:"static"` }

var adTypeConfig = map[string]map[int]int{"news": {1:1,2:1,3:2,4:-1}, "home": {1:1,2:1,3:1,4:-1}}

func (d adsDeps) AdsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","application/json")
	ptype := r.URL.Query().Get("type")
	rulesBase, ok := adTypeConfig[ptype]
	if !ok { http.Error(w, `{"error":"type inválido"}`, 400); return }
	t := tenant.FromRequestHost(r.Host, r.Header.Get("X-Forwarded-Host"))

	final := map[int]int{}
	for tp, lim := range rulesBase {
		if lim == -1 { if v, _ := strconv.Atoi(r.URL.Query().Get(fmt.Sprintf("ad_type_%d", tp))); v>0 { final[tp]=v } } else { final[tp]=lim }
	}

	all := d.Cache.Get(t.ID)

	uk := userKey(r)
	recent, _ := d.Recent.Get(r, t.ID, uk, d.Cfg.RecentN)
	recentSet := make(map[string]struct{}, len(recent)); for _, id := range recent { recentSet[id]=struct{}{} }

	pool := filterByTypes(all, final)
	pool = avoidRecent(pool, recentSet)
	sel := ads.FilterByRules(ads.Shuffle(pool), final)

	if len(sel) > 0 {
		ids := make([]string,0,len(sel)); for _, a := range sel { ids = append(ids, a.ID) }
		_ = d.Recent.Push(r, t.ID, uk, ids, d.Cfg.RecentN)
	}
	json.NewEncoder(w).Encode(adsResponse{Ads: sel, Redirect: t.AdsURL, Static: t.Static})
}

func (d adsDeps) CacheClear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type","application/json")
	w.Write([]byte(`{"success":true}`))
}

// helpers
func filterByTypes(list []ads.Ad, rules map[int]int) []ads.Ad {
	if len(rules)==0 { return list }
	set := map[int]struct{}{}
	for tp := range rules { set[tp]=struct{}{} }
	out := make([]ads.Ad,0,len(list))
	for _, a := range list { if a.Active { if _, ok := set[a.Type]; ok { out = append(out, a) } } }
	return out
}

func userIP(r *http.Request) string { if v:=r.Header.Get("cf-connecting-ip"); v!=""{return v}; if v:=r.Header.Get("x-forwarded-for"); v!=""{return strings.TrimSpace(strings.Split(v,",")[0])}; return strings.Split(r.RemoteAddr,":")[0] }
func userKey(r *http.Request) string { ip:=strings.TrimSpace(userIP(r)); ua:=strings.TrimSpace(r.UserAgent()); sum:=sha256.Sum256([]byte(ip+"|"+ua)); return base64.RawURLEncoding.EncodeToString(sum[:]) }
func avoidRecent(in []ads.Ad, recent map[string]struct{}) []ads.Ad { if len(recent)==0 {return in}; keep:=make([]ads.Ad,0,len(in)); for _,a:=range in{ if _,ok:=recent[a.ID];!ok{keep=append(keep,a)} }; if len(keep)==0 {return in}; return keep }
