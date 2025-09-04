package search

import (
	"net/url"
	"strings"
	"time"
)

func getFirst(qs url.Values, k, def string) string {
	if v := qs.Get(k); strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func mustInt(qs url.Values, k string, def, min, max int) int {
	v := def
	if s := strings.TrimSpace(qs.Get(k)); s != "" {
		// parse manual
		var n int
		sign := 1
		for i, r := range s {
			if i == 0 && r == '-' {
				sign = -1
				continue
			}
			if r < '0' || r > '9' {
				n = def
				break
			}
			n = n*10 + int(r-'0')
		}
		v = n * sign
	}
	if v < min { v = min }
	if v > max { v = max }
	return v
}

// parseIntString: igual ao mustInt, mas para uma string solta (tenant)
func parseIntString(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" { return def }
	var n int
	sign := 1
	for i, r := range s {
		if i == 0 && r == '-' {
			sign = -1
			continue
		}
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return n * sign
}

func BuildQuery(q string, tenant string, qs url.Values) map[string]any {
	// paginação
	size := mustInt(qs, "size", 20, 1, 50)
	from := mustInt(qs, "from", 0, 0, 10000)

	// ---- filtros base ----
	tenantID := parseIntString(tenant, 0) // seu mapping é numérico
	filters := []map[string]any{
		{"term": map[string]any{"tenant_id": tenantID}},
	}

	// ---- filtros de campo ----
	// topics: aceitar ?topics=A,B,C OU ?topic=A
	topics := []string{}
	if v := getFirst(qs, "topics", ""); v != "" {
		for _, t := range strings.Split(v, ",") {
			if tt := strings.TrimSpace(t); tt != "" {
				topics = append(topics, tt)
			}
		}
	} else if v := getFirst(qs, "topic", ""); v != "" {
		topics = append(topics, strings.TrimSpace(v))
	}
	if len(topics) > 0 {
		should := make([]any, 0, len(topics)*2)
		for _, t := range topics {
			should = append(should,
				map[string]any{"term": map[string]any{"topic.name.keyword": t}},
				map[string]any{"match_phrase": map[string]any{"topic.name": t}},
			)
		}
		filters = append(filters, map[string]any{
			"bool": map[string]any{
				"should":               should,
				"minimum_should_match": 1,
			},
		})
	}

	// book.name
	if v := getFirst(qs, "book", ""); v != "" {
		filters = append(filters, map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"term": map[string]any{"book.name.keyword": v}},
					map[string]any{"match_phrase": map[string]any{"book.name": v}},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// author.name (array de objetos)
	if v := getFirst(qs, "author", ""); v != "" {
		filters = append(filters, map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"term": map[string]any{"author.name.keyword": v}},
					map[string]any{"match_phrase": map[string]any{"author.name": v}},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// column.name (array de objetos; às vezes vazio)
	if v := getFirst(qs, "column", ""); v != "" {
		filters = append(filters, map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"term": map[string]any{"column.name.keyword": v}},
					map[string]any{"match_phrase": map[string]any{"column.name": v}},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// ---- filtro por data: (date OR date_update) ----
	// aceita date_from/date_to e também from/to como fallback
	df := getFirst(qs, "date_from", getFirst(qs, "from", ""))
	dt := getFirst(qs, "date_to", getFirst(qs, "to", ""))
	if df != "" || dt != "" {
		r := map[string]any{}
		if df != "" { r["gte"] = df }
		if dt != "" { r["lte"] = dt }
		filters = append(filters, map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"range": map[string]any{"date": r}},
					map[string]any{"range": map[string]any{"date_update": r}},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// ---- scoring principal (campos e pesos) ----
	fields := []string{
		"title^8",
		"line^5",
		"author.name^4",
		"book.name^3",
		"column.name^3",
		"body^1",
	}

	must := []any{
		map[string]any{
			"dis_max": map[string]any{
				"tie_breaker": 0.1,
				"queries": []any{
					map[string]any{"multi_match": map[string]any{
						"query":   q,
						"type":    "phrase",
						"fields":  []string{"title^9", "line^6"},
						"slop":    1,
						"lenient": true,
					}},
					map[string]any{"multi_match": map[string]any{
						"query":     q,
						"type":      "best_fields",
						"fields":    fields,
						"operator":  "and",
						"fuzziness": "AUTO",
						"lenient":   true,
					}},
				},
			},
		},
	}

	boolQ := map[string]any{
		"filter": filters,
		"must":   must,
	}

	// recência: favorece date_update e depois date
	now := time.Now().UTC().Format(time.RFC3339)
	query := map[string]any{
		"function_score": map[string]any{
			"query": map[string]any{"bool": boolQ},
			"functions": []any{
				map[string]any{"gauss": map[string]any{"date_update": map[string]any{
					"origin": now, "scale": "10d", "decay": 0.6,
				}}},
				map[string]any{"gauss": map[string]any{"date": map[string]any{
					"origin": now, "scale": "21d", "decay": 0.5,
				}}},
			},
			"score_mode": "sum",
			"boost_mode": "multiply",
		},
	}

	// highlight
	hl := map[string]any{
		"pre_tags":            []string{"<mark>"},
		"post_tags":           []string{"</mark>"},
		"require_field_match": false,
		"fields": map[string]any{
			"title":       map[string]any{},
			"line":        map[string]any{},
			"author.name": map[string]any{},
			"book.name":   map[string]any{},
			"column.name": map[string]any{},
			"body":        map[string]any{"fragment_size": 140, "number_of_fragments": 1},
		},
	}

	// _source
	src := []string{
		"tenant_id",
		"title", "line", "body",
		"url",
		"show_date", "show_date_update",
		"date", "date_update",
		"topic",   // []{name,link}
		"author",  // []{name,link}
		"column",  // []{name,link} ou []
		"book",    // {name,link}
	}

	// aggs para facetas
	aggs := map[string]any{
		"books":   map[string]any{"terms": map[string]any{"field": "book.name.keyword", "size": 20}},
		"authors": map[string]any{"terms": map[string]any{"field": "author.name.keyword", "size": 30}},
		"columns": map[string]any{"terms": map[string]any{"field": "column.name.keyword", "size": 30}},
		"topics":  map[string]any{"terms": map[string]any{"field": "topic.name.keyword", "size": 30}},
	}

	return map[string]any{
		"track_total_hits": false,
		"size":             size,
		"from":             from,
		"_source":          src,
		"query":            query,
		"highlight":        hl,
		"sort": []any{
			map[string]any{"_score": map[string]any{"order": "desc"}},
			map[string]any{"date_update": map[string]any{"order": "desc"}},
			map[string]any{"date": map[string]any{"order": "desc"}},
		},
		"aggs": aggs,
	}
}
