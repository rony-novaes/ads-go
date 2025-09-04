package search

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

func NewClient(base string) *Client {
	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		DialContext:         (&net.Dialer{Timeout: 500 * time.Millisecond}).DialContext,
		TLSHandshakeTimeout: 500 * time.Millisecond,
	}
	return &Client{
		base: base,
		http: &http.Client{Transport: tr, Timeout: 1200 * time.Millisecond},
	}
}

type esHit struct {
	Score     float64              `json:"_score"`
	Source    map[string]any       `json:"_source"`
	Highlight map[string][]string  `json:"highlight,omitempty"`
}
type esHits struct {
	Total struct {
		Value int `json:"value"`
	} `json:"total"`
	Hits []esHit `json:"hits"`
}
type ESResponse struct {
	Took         int             `json:"took"`
	Hits         esHits          `json:"hits"`
	Aggregations map[string]any  `json:"aggregations,omitempty"`
}

func (c *Client) Search(ctx context.Context, index string, body map[string]any) (*ESResponse, error) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/"+index+"/_search", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out ESResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
