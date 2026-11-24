package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/fuzzy"
)

// The Pokédex list is in National Pokédex order, each alternate form right after its
// species (#0006 Charizard, Charizard Mega X, Charizard Gmax, #0007 Squirtle…), and can be
// filtered with "type:<type>" and "gen:<n>".

const maxPokemonMatches = 300

// buildDexOrder computes m.dexOrder, m.dexRank and m.dexNum from the Pokémon list.
func (m *tuiModel) buildDexOrder() {
	names := m.data.KnownPokemon()
	species := map[string]int{} // species-level Pokémon -> Pokédex number
	for _, n := range names {
		if id := m.data.PokemonID(n); id > 0 && id < 10000 {
			species[n] = id
		}
	}
	type entry struct {
		name     string
		num, id  int
		isFormOf bool
	}
	entries := make([]entry, 0, len(names))
	m.dexNum = map[string]int{}
	for _, n := range names {
		id := m.data.PokemonID(n)
		e := entry{name: n, id: id, num: id}
		if _, ok := species[n]; !ok {
			e.isFormOf = true
			e.num = 1 << 30 // forms without a known species go last
			if base := baseSpecies(n, species); base != "" {
				e.num = species[base]
			}
		}
		m.dexNum[n] = e.num
		entries = append(entries, e)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.num != b.num {
			return a.num < b.num
		}
		if a.isFormOf != b.isFormOf {
			return !a.isFormOf
		}
		return a.id < b.id
	})
	m.dexOrder = make([]string, len(entries))
	m.dexRank = make(map[string]int, len(entries))
	for i, e := range entries {
		m.dexOrder[i] = e.name
		m.dexRank[e.name] = i
	}
}

// baseSpecies finds the species of a form by dropping "-suffix" parts:
// charizard-mega-x -> charizard.
func baseSpecies(name string, species map[string]int) string {
	for n := name; strings.Contains(n, "-"); {
		n = n[:strings.LastIndex(n, "-")]
		if _, ok := species[n]; ok {
			return n
		}
	}
	return ""
}

// sortByDex returns names sorted in Pokédex order (unknown names last).
func (m tuiModel) sortByDex(names []string) []string {
	out := append([]string(nil), names...)
	rank := func(n string) int {
		if r, ok := m.dexRank[n]; ok {
			return r
		}
		return 1 << 30
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

// dexRow renders "#0025 Pikachu" (forms show their species' number, dimmed).
func (m tuiModel) dexRow(name string, selected bool, width int) string {
	num := "     "
	if n := m.dexNum[name]; n > 0 && n < 1<<30 {
		num = fmt.Sprintf("#%04d", n)
	}
	label := truncate(dex.DisplayName(name), max(5, width-9))
	isForm := m.data.PokemonID(name) >= 10000
	if selected {
		return selectedStyle.Render("▸ " + num + " " + label)
	}
	if isForm {
		return "  " + labelStyle.Render(num) + " " + label
	}
	return "  " + num + " " + label
}

// ----------------------------------------------------------------------------- filters

type pokedexQuery struct {
	text  string   // what is left after the filters, normalised
	types []string // type:fire
	gens  []int    // gen:1
	bad   string   // why the query is invalid
}

var romanNumerals = map[string]int{"i": 1, "ii": 2, "iii": 3, "iv": 4, "v": 5, "vi": 6, "vii": 7, "viii": 8, "ix": 9, "x": 10}

func (m tuiModel) parsePokedexQuery(raw string) pokedexQuery {
	var q pokedexQuery
	var words []string
	for _, tok := range strings.Fields(strings.ToLower(raw)) {
		key, val, ok := strings.Cut(tok, ":")
		switch {
		case ok && (key == "type" || key == "t"):
			if _, known := m.data.Chart()[val]; !known {
				if val != "" { // still typing the value
					q.bad = fmt.Sprintf("Unknown type %q", val)
				}
				continue
			}
			q.types = append(q.types, val)
		case ok && (key == "gen" || key == "g"):
			n, err := strconv.Atoi(val)
			if err != nil {
				n = romanNumerals[val]
			}
			if n < 1 || n > 20 {
				if val != "" {
					q.bad = fmt.Sprintf("Unknown generation %q (use 1-9)", val)
				}
				continue
			}
			q.gens = append(q.gens, n)
		default:
			words = append(words, tok)
		}
	}
	q.text = fuzzy.Normalize(strings.Join(words, " "))
	return q
}

func typeKey(t string) string { return "type:" + t }
func genKey(n int) string     { return fmt.Sprintf("gen:%d", n) }

// missingFilters lists the filter data that still has to be downloaded.
func (m tuiModel) missingFilters(q pokedexQuery) []string {
	var out []string
	for _, t := range q.types {
		if m.filterSets[typeKey(t)] == nil {
			out = append(out, typeKey(t))
		}
	}
	for _, g := range q.gens {
		if m.filterSets[genKey(g)] == nil {
			out = append(out, genKey(g))
		}
	}
	return out
}

// matchesFilters keeps Pokémon that have every requested type and belong to one of the
// requested generations (by species: regional forms count with their species).
func (m tuiModel) matchesFilters(name string, q pokedexQuery) bool {
	for _, t := range q.types {
		if !m.filterSets[typeKey(t)][name] {
			return false
		}
	}
	if len(q.gens) == 0 {
		return true
	}
	for _, g := range q.gens {
		set := m.filterSets[genKey(g)]
		for n := name; ; n = n[:strings.LastIndex(n, "-")] {
			if set[n] {
				return true
			}
			if !strings.Contains(n, "-") {
				break
			}
		}
	}
	return false
}

// pokedexSuggestions returns the Pokédex list for the current query.
func (m tuiModel) pokedexSuggestions(raw string) (list []string, note string) {
	q := m.parsePokedexQuery(raw)
	if q.bad != "" {
		return nil, q.bad
	}
	if missing := m.missingFilters(q); len(missing) > 0 {
		return nil, "Loading " + strings.Join(missing, ", ") + "…"
	}
	candidates := m.dexOrder
	if len(q.types)+len(q.gens) > 0 {
		candidates = nil
		for _, n := range m.dexOrder {
			if m.matchesFilters(n, q) {
				candidates = append(candidates, n)
			}
		}
	}
	if q.text == "" {
		return candidates, ""
	}
	return fuzzy.Ranked(q.text, candidates, maxPokemonMatches), ""
}

// fetchFilters downloads the filter data the current Pokédex query needs.
func (m *tuiModel) fetchFilters() tea.Cmd {
	if m.search.focus != fieldPokemon {
		return nil
	}
	var cmds []tea.Cmd
	for _, key := range m.missingFilters(m.parsePokedexQuery(m.search.inputs[fieldPokemon].Value())) {
		cmds = append(cmds, m.fetchFilter(key))
	}
	return tea.Batch(cmds...)
}

func (m *tuiModel) fetchFilter(key string) tea.Cmd {
	if m.pending["filter:"+key] {
		return nil
	}
	m.pending["filter:"+key] = true
	d := m.data
	return tea.Batch(m.spin(), func() tea.Msg {
		kind, val, _ := strings.Cut(key, ":")
		var names []string
		var err error
		if kind == "type" {
			names, err = d.PokemonOfType(val)
		} else {
			n, _ := strconv.Atoi(val)
			names, err = d.SpeciesOfGeneration(n)
		}
		return filterLoadedMsg{key, names, err}
	})
}
