package routes

import (
	"container/list"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis

type redisRecent struct { rdb *redis.Client }

func NewRedisRecent(rdb *redis.Client) *redisRecent { return &redisRecent{rdb: rdb} }

func (s *redisRecent) Get(r *http.Request, tenantID int, userKey string, n int) ([]string, error) {
	key := recentKey(tenantID, userKey)
	vals, err := s.rdb.LRange(r.Context(), key, 0, int64(n-1)).Result()
	if err != nil { return []string{}, nil }
	return vals, nil
}

func (s *redisRecent) Push(r *http.Request, tenantID int, userKey string, ids []string, n int) error {
	key := recentKey(tenantID, userKey)
	args := make([]interface{}, len(ids))
	for i, id := range ids { args[i] = id }
	pipe := s.rdb.TxPipeline()
	pipe.LPush(r.Context(), key, args...)
	pipe.LTrim(r.Context(), key, 0, int64(n-1))
	pipe.Expire(r.Context(), key, 6*time.Hour)
	_, _ = pipe.Exec(r.Context())
	return nil
}

func recentKey(tenantID int, userKey string) string { return fmt.Sprintf("recent:%d:%s", tenantID, userKey) }

// Memória (para quando não houver Redis)

type memoryRecent struct { m map[string]*list.List }

func NewMemoryRecent() *memoryRecent { return &memoryRecent{m: map[string]*list.List{}} }

func (s *memoryRecent) Get(_ *http.Request, tenantID int, userKey string, n int) ([]string, error) {
	k := recentKey(tenantID, userKey)
	lst, ok := s.m[k]
	if !ok { return []string{}, nil }
	out := []string{}
	for e, i := lst.Front(), 0; e != nil && i < n; e, i = e.Next(), i+1 {
		out = append(out, e.Value.(string))
	}
	return out, nil
}

func (s *memoryRecent) Push(_ *http.Request, tenantID int, userKey string, ids []string, n int) error {
	k := recentKey(tenantID, userKey)
	lst, ok := s.m[k]
	if !ok { lst = list.New(); s.m[k] = lst }
	for _, id := range ids { lst.PushFront(id) }
	for lst.Len() > n { lst.Remove(lst.Back()) }
	return nil
}
