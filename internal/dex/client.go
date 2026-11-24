// Package dex loads Pokémon data from PokeAPI (with a local cache) and holds the
// domain logic: Pokémon, moves, abilities, evolutions, natures and type effectiveness.
package dex

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const apiBase = "https://pokeapi.co/api/v2"

// Options tune the client. Zero values fall back to the defaults.
type Options struct {
	Refresh     bool          // ignore the cache and download everything again
	CacheTTL    time.Duration // age after which cached data is downloaded again (default 30 days)
	MaxRequests int           // concurrent requests to PokeAPI (default 6)
}

var ignoredTypes = map[string]bool{"unknown": true, "shadow": true, "stellar": true}

var ErrNotFound = errors.New("not found")

// TypeChart maps defending type -> attacking type -> multiplier (only entries != 1).
type TypeChart map[string]map[string]float64

type Client struct {
	refresh  bool
	cacheTTL time.Duration
	cacheDir string
	client   *http.Client
	sem      chan struct{} // limits concurrent requests to PokeAPI

	// Progress, when set, receives short status messages about slow downloads.
	Progress io.Writer

	types        TypeChart
	typeNames    []string
	pokemonNames []string
	pokemonSet   map[string]bool
}

func NewClient(opts Options) *Client {
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = 30 * 24 * time.Hour
	}
	if opts.MaxRequests <= 0 {
		opts.MaxRequests = 6
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "pokedex")
	_ = os.MkdirAll(dir, 0o755)
	return &Client{
		refresh:  opts.Refresh,
		cacheTTL: opts.CacheTTL,
		cacheDir: dir,
		client:   &http.Client{Timeout: 15 * time.Second},
		sem:      make(chan struct{}, opts.MaxRequests),
	}
}

func (d *Client) get(url string, out any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pokedex-cli/1.0")
	d.sem <- struct{}{}
	defer func() { <-d.sem }()
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PokeAPI returned %s for %s", resp.Status, url)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// cached loads name.json from the cache if fresh, otherwise calls load and stores the result.
func cached[T any](d *Client, name string, load func() (T, error)) (T, error) {
	path := filepath.Join(d.cacheDir, name+".json")
	if !d.refresh {
		if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) < d.cacheTTL {
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

func (d *Client) Types() (TypeChart, error) {
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

func (d *Client) TypeNames() []string {
	if _, err := d.Types(); err != nil {
		return nil
	}
	return d.typeNames
}

func (d *Client) loadTypes() (TypeChart, error) {
	d.progress("Downloading type chart from PokeAPI...")
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

func (d *Client) PokemonNames() ([]string, error) {
	if d.pokemonNames != nil {
		return d.pokemonNames, nil
	}
	names, err := cached(d, "pokemon_names", func() ([]string, error) {
		d.progress("Downloading Pokémon list from PokeAPI...")
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

func (d *Client) progress(msg string) {
	if d.Progress != nil {
		fmt.Fprintln(d.Progress, msg)
	}
}

// KnownPokemon returns every Pokémon name, once PokemonNames has loaded them.
func (d *Client) KnownPokemon() []string { return d.pokemonNames }

// IsPokemon reports whether name is a known Pokémon (after PokemonNames has loaded).
func (d *Client) IsPokemon(name string) bool { return d.pokemonSet[name] }

// Chart returns the type chart, once Types has loaded it.
func (d *Client) Chart() TypeChart { return d.types }

// NewOfflineClient returns a client preloaded with Pokémon names and a type chart,
// without network access or cache. Useful for tests.
func NewOfflineClient(names []string, chart TypeChart) *Client {
	d := &Client{types: chart, pokemonNames: names, pokemonSet: map[string]bool{}}
	for _, n := range names {
		d.pokemonSet[n] = true
	}
	for t := range chart {
		d.typeNames = append(d.typeNames, t)
	}
	sort.Strings(d.typeNames)
	return d
}
