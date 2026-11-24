package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	apiBase  = "https://pokeapi.co/api/v2"
	cacheTTL = 30 * 24 * time.Hour
)

var ignoredTypes = map[string]bool{"unknown": true, "shadow": true, "stellar": true}

var errNotFound = errors.New("not found")

// TypeChart maps defending type -> attacking type -> multiplier (only entries != 1).
type TypeChart map[string]map[string]float64

type Pokemon struct {
	Name      string   `json:"name"`
	ID        int      `json:"id"`
	Types     []string `json:"types"`
	Abilities []string `json:"abilities"`
}

type PokeData struct {
	refresh  bool
	cacheDir string
	client   *http.Client

	types        TypeChart
	typeNames    []string
	pokemonNames []string
	pokemonSet   map[string]bool
}

func NewPokeData(refresh bool) *PokeData {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "type_calc")
	_ = os.MkdirAll(dir, 0o755)
	return &PokeData{
		refresh:  refresh,
		cacheDir: dir,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (d *PokeData) get(url string, out any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "type-calc-cli/1.0")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PokeAPI returned %s for %s", resp.Status, url)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// cached loads name.json from the cache if fresh, otherwise calls load and stores the result.
func cached[T any](d *PokeData, name string, load func() (T, error)) (T, error) {
	path := filepath.Join(d.cacheDir, name+".json")
	if !d.refresh {
		if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) < cacheTTL {
			if raw, err := os.ReadFile(path); err == nil {
				var v T
				if json.Unmarshal(raw, &v) == nil {
					return v, nil
				}
			}
		}
	}
	v, err := load()
	if err != nil {
		return v, err
	}
	if raw, err := json.Marshal(v); err == nil {
		_ = os.WriteFile(path, raw, 0o644)
	}
	return v, nil
}

type namedURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (d *PokeData) Types() (TypeChart, error) {
	if d.types != nil {
		return d.types, nil
	}
	chart, err := cached(d, "types", d.loadTypes)
	if err != nil {
		return nil, err
	}
	d.types = chart
	for t := range chart {
		d.typeNames = append(d.typeNames, t)
	}
	sort.Strings(d.typeNames)
	return chart, nil
}

func (d *PokeData) TypeNames() []string {
	if _, err := d.Types(); err != nil {
		return nil
	}
	return d.typeNames
}

func (d *PokeData) loadTypes() (TypeChart, error) {
	fmt.Fprintln(os.Stderr, dim("Downloading type chart from PokeAPI..."))
	var listing struct{ Results []namedURL }
	if err := d.get(apiBase+"/type?limit=100", &listing); err != nil {
		return nil, err
	}

	type typeResp struct {
		DamageRelations struct {
			DoubleFrom []namedURL `json:"double_damage_from"`
			HalfFrom   []namedURL `json:"half_damage_from"`
			NoFrom     []namedURL `json:"no_damage_from"`
		} `json:"damage_relations"`
	}

	chart := TypeChart{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	for _, entry := range listing.Results {
		if ignoredTypes[entry.Name] {
			continue
		}
		wg.Add(1)
		go func(e namedURL) {
			defer wg.Done()
			var tr typeResp
			err := d.get(e.URL, &tr)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			m := map[string]float64{}
			for _, t := range tr.DamageRelations.DoubleFrom {
				m[t.Name] = 2
			}
			for _, t := range tr.DamageRelations.HalfFrom {
				m[t.Name] = 0.5
			}
			for _, t := range tr.DamageRelations.NoFrom {
				m[t.Name] = 0
			}
			chart[e.Name] = m
		}(entry)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return chart, nil
}

func (d *PokeData) PokemonNames() ([]string, error) {
	if d.pokemonNames != nil {
		return d.pokemonNames, nil
	}
	names, err := cached(d, "pokemon_names", func() ([]string, error) {
		fmt.Fprintln(os.Stderr, dim("Downloading Pokémon list from PokeAPI..."))
		var listing struct{ Results []namedURL }
		if err := d.get(apiBase+"/pokemon?limit=100000", &listing); err != nil {
			return nil, err
		}
		out := make([]string, len(listing.Results))
		for i, p := range listing.Results {
			out[i] = p.Name
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	d.pokemonNames = names
	d.pokemonSet = make(map[string]bool, len(names))
	for _, n := range names {
		d.pokemonSet[n] = true
	}
	return names, nil
}

func (d *PokeData) Pokemon(name string) (Pokemon, error) {
	return cached(d, "pokemon_"+name, func() (Pokemon, error) {
		var raw struct {
			Name  string
			ID    int
			Types []struct {
				Slot int
				Type namedURL
			}
			Abilities []struct {
				Ability namedURL
			}
		}
		if err := d.get(apiBase+"/pokemon/"+name, &raw); err != nil {
			return Pokemon{}, err
		}
		sort.Slice(raw.Types, func(i, j int) bool { return raw.Types[i].Slot < raw.Types[j].Slot })
		p := Pokemon{Name: raw.Name, ID: raw.ID}
		for _, t := range raw.Types {
			p.Types = append(p.Types, t.Type.Name)
		}
		for _, a := range raw.Abilities {
			p.Abilities = append(p.Abilities, a.Ability.Name)
		}
		return p, nil
	})
}
