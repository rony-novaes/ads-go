package ads

import (
	"math/rand"
)

// Shuffle Fisher–Yates
func Shuffle(in []Ad) []Ad {
	out := make([]Ad, len(in))
	copy(out, in)
	for i := len(out)-1; i>0; i-- {
		j := rand.Intn(i+1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// FilterByRules aplica limites por tipo após shuffle
func FilterByRules(list []Ad, rules map[int]int) []Ad {
	out := make([]Ad, 0)
	count := map[int]int{}
	for _, a := range list {
		lim, ok := rules[a.Type]
		if !ok || lim == 0 { continue }
		if lim > 0 && count[a.Type] >= lim { continue }
		out = append(out, a)
		count[a.Type]++
	}
	return out
}
