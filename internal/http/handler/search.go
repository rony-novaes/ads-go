package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"search-golang/internal/http/middleware"
	"search-golang/internal/http/response"
	"search-golang/internal/search"
	"search-golang/internal/util"
)

func Search(os *search.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("query"))
		if q == "" {
			http.Error(w, "query é obrigatória", http.StatusBadRequest)
			return
		}

		t := middleware.GetTenant(r)
		index := fmt.Sprintf("news_t%d_read", t.ID)

		body := search.BuildQuery(q, strconv.Itoa(t.ID), r.URL.Query())

		ctx, cancel := context.WithTimeout(r.Context(), 1100*time.Millisecond)
		defer cancel()

		es, err := os.Search(ctx, index, body)
		if err != nil {
			http.Error(w, "erro na consulta ao mecanismo de busca", http.StatusBadGateway)
			return
		}

		out := response.SearchResponse{
			TookMS: es.Took,
			Total:  es.Hits.Total.Value,
			Items:  make([]response.SearchItem, 0, len(es.Hits.Hits)),
			Facets: map[string]any{},
		}

		for _, h := range es.Hits.Hits {
			src := h.Source
			item := response.SearchItem{
				ID:          util.Str(src["id"]),
				Title:       util.Str(src["title"]),
				Dek:         util.Str(src["dek"]),
				Canonical:   util.Str(src["canonical"]),
				Section:     util.NestedStr(src, "section.slug"),
				Topics:      util.NestedSliceStr(src, "topics.slug"),
				Author:      util.NestedStr(src, "author.slug"),
				Column:      util.NestedStr(src, "column.slug"),
				PublishedAt: util.Str(src["published_at"]),
				HasImage:    util.NestedBool(src, "image.has"),
				Score:       h.Score,
				Snippet:     util.FirstOrEmpty(h.Highlight["body_text"]),
			}
			out.Items = append(out.Items, item)
		}
		if es.Aggregations != nil {
			out.Facets = es.Aggregations
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		// cache leve (a borda/CF pode melhorar)
		w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=60")
		_ = json.NewEncoder(w).Encode(out)
	}
}
