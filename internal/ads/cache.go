package ads

import (
	"context"
	"sync"
	"time"
)

// Cache mantém anúncios em memória por tenant.
type Cache struct {
	mu    sync.RWMutex
	data  map[int][]ResponseAd // tenantID -> lista completa de anúncios ativos
	fresh time.Time
}

func NewCache() *Cache {
	return &Cache{
		data: map[int][]ResponseAd{},
	}
}

// Set substitui o conjunto de anúncios de um tenant.
func (c *Cache) Set(tenantID int, ads []ResponseAd) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[tenantID] = ads
	c.fresh = time.Now()
}

// Get retorna a lista atual de anúncios do tenant; pode retornar slice vazio.
func (c *Cache) Get(tenantID int) []ResponseAd {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.data[tenantID]; ok {
		return v
	}
	return nil
}

// StartRefresher atualiza o cache periodicamente a partir do repo (MySQL).
func (c *Cache) StartRefresher(ctx context.Context, repo Repository, tenants []int, interval time.Duration, logger func(string, ...any)) {
	refresh := func() {
		for _, tid := range tenants {
			list, err := repo.ActiveByTypes(ctx, tid, []int{1, 2, 3, 4})
			if err != nil {
				if logger != nil {
					logger("ads refresh tenant=%d err=%v", tid, err)
				}
				continue
			}
			c.Set(tid, list)
			if logger != nil {
				logger("ads refresh tenant=%d ok items=%d", tid, len(list))
			}
		}
	}

	// Primeira carga imediata
	refresh()

	// Atualizações periódicas
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				refresh()
			}
		}
	}()
}
