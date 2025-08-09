package routes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ads-go/internal/config"
)

type clickDeps struct { Cfg config.Config }

type clickToken struct { A string `json:"a"`; E time.Time `json:"e"`; T int `json:"t"` }

func sign(secret []byte, p clickToken) (string, error) { b,_ := json.Marshal(p); h:=hmac.New(sha256.New,secret); h.Write(b); sig:=h.Sum(nil); msg:=append(b,sig...); return base64.RawURLEncoding.EncodeToString(msg), nil }
func verify(secret []byte, tok string) (clickToken, error) {
	var zero clickToken
	data, err := base64.RawURLEncoding.DecodeString(tok); if err!=nil || len(data)<sha256.Size { return zero, errors.New("bad") }
	b := data[:len(data)-sha256.Size]; sig := data[len(data)-sha256.Size:]
	h:=hmac.New(sha256.New,secret); h.Write(b); if !hmac.Equal(h.Sum(nil), sig) { return zero, errors.New("sig") }
	if err := json.Unmarshal(b,&zero); err!=nil { return zero, err }
	if time.Now().After(zero.E) { return zero, errors.New("exp") }
	return zero, nil
}

func (d clickDeps) Click(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("k"); if tok=="" { http.NotFound(w,r); return }
	p, err := verify([]byte(d.Cfg.HMACSecret), tok); if err!=nil { http.NotFound(w,r); return }
	_ = p // placeholder até conectar com lookup real
	w.Header().Set("Cache-Control","no-store")
	http.Redirect(w,r, "https://conexaoguarulhos.com.br", http.StatusFound)
}
