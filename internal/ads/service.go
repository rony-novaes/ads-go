package ads

import (
	"math/rand"
	"sort"
)

// Shuffle Fisher–Yates para []ResponseAd
func Shuffle(in []ResponseAd) []ResponseAd {
	out := make([]ResponseAd, len(in))
	copy(out, in)
	for i := len(out)-1; i > 0; i-- {
		j := rand.Intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// FilterByRules aplica limites por tipo (rules: tipo -> limite).
// -1 em rules indica ilimitado para aquele tipo.
// Cada anúncio é incluído no máximo UMA vez; se tiver vários tipos,
// ele consome a cota do primeiro tipo disponível (ordenado pelo id do tipo).
func FilterByRules(list []ResponseAd, rules map[int]int) []ResponseAd {
	if len(rules) == 0 {
		return list
	}

	// cópia mutável das cotas
	remain := make(map[int]int, len(rules))
	for k, v := range rules {
		remain[k] = v
	}

	out := make([]ResponseAd, 0, len(list))

	for _, ad := range list {
		if len(ad.Types) == 0 {
			continue
		}

		// ordena os tipos do anúncio para ter critério estável
		keys := make([]int, 0, len(ad.Types))
		for tp := range ad.Types {
			keys = append(keys, tp)
		}
		sort.Ints(keys)

		chosenType := -1
		unlimited := false

		for _, tp := range keys {
			lim, ok := remain[tp]
			if !ok {
				continue // tipo não solicitado nas regras
			}
			if lim < 0 { // ilimitado
				chosenType = tp
				unlimited = true
				break
			}
			if lim > 0 {
				chosenType = tp
				break
			}
			// lim == 0 -> sem cota restante, tenta próximo tipo do anúncio
		}

		if chosenType == -1 {
			continue // nenhum tipo do anúncio possui cota disponível
		}

		out = append(out, ad)
		if !unlimited {
			remain[chosenType]--
		}
	}

	return out
}
