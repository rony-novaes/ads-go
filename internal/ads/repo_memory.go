package ads

import (
    "context"
    "encoding/json"
    "errors"
    "strconv"
    "time"

    "github.com/redis/go-redis/v9"
)

type memoryRepo struct {
	rdb *redis.Client
}

func NewMemoryRepo(rdb *redis.Client) Repository { return &memoryRepo{rdb: rdb} }

func cacheKeyAds(tenantID int) string { return "ads:" + strconv.Itoa(tenantID) }

func (m *memoryRepo) ActiveByTypes(ctx context.Context, tenantID int, types []int) ([]Ad, error) {
	key := cacheKeyAds(tenantID)
	raw, err := m.rdb.Get(ctx, key).Result()
	var list []Ad
	if err == nil {
		_ = json.Unmarshal([]byte(raw), &list)
	} else if errors.Is(err, redis.Nil) {
		list = demo(tenantID)
		b, _ := json.Marshal(list)
		_ = m.rdb.Set(ctx, key, string(b), 10*time.Minute).Err()
	} else { return nil, err }
	if len(types) == 0 { return list, nil }
	set := map[int]struct{}{}
	for _, t := range types { set[t] = struct{}{} }
	out := make([]Ad, 0, len(list))
	for _, a := range list { if a.Active { if _, ok := set[a.Type]; ok { out = append(out, a) } } }
	return out, nil
}

func demo(tenantID int) []Ad {
	static := "https://static.conexaoguarulhos.com.br"
	return []Ad{{ID:"a1",Type:1,ImageURL:static+"/img/a1.webp",TargetURL:"https://ex.com/a1",Active:true,Weight:1},
		{ID:"a2",Type:2,ImageURL:static+"/img/a2.webp",TargetURL:"https://ex.com/a2",Active:true,Weight:1},
		{ID:"a3",Type:3,ImageURL:static+"/img/a3.webp",TargetURL:"https://ex.com/a3",Active:true,Weight:1},
		{ID:"a4",Type:4,ImageURL:static+"/img/a4.webp",TargetURL:"https://ex.com/a4",Active:true,Weight:1}}
}
