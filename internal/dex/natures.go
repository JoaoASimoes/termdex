package dex

import (
	"github.com/JoaoASimoes/pokedex/internal/fuzzy"
	"sort"
	"strings"
	"sync"
)

// Nature raises one stat by 10% and lowers another by 10% (neutral natures change nothing).
type Nature struct {
	Name      string `json:"name"`
	Increased string `json:"increased"` // stat name; equal to Decreased when neutral
	Decreased string `json:"decreased"`
	Likes     string `json:"likes"` // berry flavor, "" when neutral
	Hates     string `json:"hates"`
}

func (n Nature) Neutral() bool { return n.Increased == n.Decreased }

// NatureStats is the order used by the nature grid.
var NatureStats = []string{"attack", "defense", "special-attack", "special-defense", "speed"}

var ShortStat = map[string]string{
	"attack": "Atk", "defense": "Def", "special-attack": "SpA", "special-defense": "SpD", "speed": "Spe",
}

// neutralDiagonal places each neutral nature on the grid, as in the games' nature charts.
var neutralDiagonal = map[string]string{
	"hardy": "attack", "docile": "defense", "bashful": "special-attack",
	"quirky": "special-defense", "serious": "speed",
}

// statAliases maps what a user may type to a stat name.
var statAliases = map[string]string{
	"atk": "attack", "attack": "attack",
	"def": "defense", "defense": "defense", "defence": "defense",
	"spa": "special-attack", "spatk": "special-attack", "sp-atk": "special-attack", "special-attack": "special-attack",
	"spd": "special-defense", "spdef": "special-defense", "sp-def": "special-defense", "special-defense": "special-defense",
	"spe": "speed", "speed": "speed", "spd-speed": "speed",
}

var (
	naturesMu   sync.Mutex
	naturesData []Nature
)

// Natures loads all natures from PokeAPI (cached on disk).
func (d *Client) Natures() ([]Nature, error) {
	naturesMu.Lock()
	defer naturesMu.Unlock()
	if naturesData != nil {
		return naturesData, nil
	}
	data, err := cached(d, "natures", func() ([]Nature, error) {
		var listing struct{ Results []namedURL }
		if err := d.get(apiBase+"/nature?limit=100", &listing); err != nil {
			return nil, err
		}
		out := make([]Nature, len(listing.Results))
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error
		for i, entry := range listing.Results {
			wg.Add(1)
			go func(i int, e namedURL) {
				defer wg.Done()
				var raw struct {
					Name          string
					IncreasedStat *namedURL `json:"increased_stat"`
					DecreasedStat *namedURL `json:"decreased_stat"`
					LikesFlavor   *namedURL `json:"likes_flavor"`
					HatesFlavor   *namedURL `json:"hates_flavor"`
				}
				if err := d.get(e.URL, &raw); err != nil {
					mu.Lock()
					firstErr = err
					mu.Unlock()
					return
				}
				n := Nature{Name: raw.Name}
				if raw.IncreasedStat != nil && raw.DecreasedStat != nil {
					n.Increased, n.Decreased = raw.IncreasedStat.Name, raw.DecreasedStat.Name
				} else {
					// Neutral natures have no stats in PokeAPI; place them on the grid diagonal.
					n.Increased, n.Decreased = neutralDiagonal[raw.Name], neutralDiagonal[raw.Name]
				}
				if !n.Neutral() {
					if raw.LikesFlavor != nil {
						n.Likes = raw.LikesFlavor.Name
					}
					if raw.HatesFlavor != nil {
						n.Hates = raw.HatesFlavor.Name
					}
				}
				out[i] = n
			}(i, entry)
		}
		wg.Wait()
		if firstErr != nil {
			return nil, firstErr
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil
	})
	if err != nil {
		return nil, err // not remembered: a later call retries
	}
	naturesData = data
	return data, nil
}

func NatureByName(natures []Nature, name string) (Nature, bool) {
	for _, n := range natures {
		if n.Name == name {
			return n, true
		}
	}
	return Nature{}, false
}

// NatureAt returns the nature that raises inc and lowers dec (neutral on the diagonal).
func NatureAt(natures []Nature, inc, dec string) (Nature, bool) {
	for _, n := range natures {
		if n.Increased == inc && n.Decreased == dec {
			return n, true
		}
	}
	return Nature{}, false
}

// SuggestNatures matches nature names, or stats ("speed", "+atk", "-spa") to the natures
// that raise (or, with a leading "-", lower) them.
func SuggestNatures(query string, natures []Nature) []string {
	var names []string
	for _, n := range natures {
		names = append(names, n.Name)
	}
	q := fuzzy.Normalize(query)
	if q == "" {
		return names
	}
	lower := strings.HasPrefix(q, "-")
	stat := statAliases[strings.TrimLeft(q, "+-")]
	if stat != "" {
		var out []string
		for _, n := range natures {
			if n.Neutral() {
				continue
			}
			if (!lower && n.Increased == stat) || (lower && n.Decreased == stat) {
				out = append(out, n.Name)
			}
		}
		return out
	}
	return fuzzy.Suggest(q, names, len(names))
}
