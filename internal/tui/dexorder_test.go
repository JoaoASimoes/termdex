package tui

import (
	"reflect"
	"testing"

	"github.com/JoaoASimoes/pokedex/internal/config"
	"github.com/JoaoASimoes/pokedex/internal/dex"
)

func dexTestModel() tuiModel {
	d := dex.NewOfflineClientIDs([]dex.PokemonEntry{
		{Name: "bulbasaur", ID: 1}, {Name: "charmander", ID: 4}, {Name: "charizard", ID: 6},
		{Name: "mewtwo", ID: 150}, {Name: "mew", ID: 151}, {Name: "moltres", ID: 146},
		{Name: "charizard-mega-x", ID: 10034}, {Name: "charizard-gmax", ID: 10196},
	}, dex.TypeChart{"fire": {}, "flying": {}, "psychic": {}})
	m := newTUIModel(d, config.Default())
	m.dataReady = true
	m.buildDexOrder()
	return m
}

func TestDexOrderPutsFormsAfterTheirSpecies(t *testing.T) {
	m := dexTestModel()
	want := []string{"bulbasaur", "charmander", "charizard", "charizard-mega-x", "charizard-gmax", "moltres", "mewtwo", "mew"}
	if !reflect.DeepEqual(m.dexOrder, want) {
		t.Fatalf("dexOrder = %v, want %v", m.dexOrder, want)
	}
	if m.dexNum["charizard-gmax"] != 6 {
		t.Errorf("a form shows its species' number, got %d", m.dexNum["charizard-gmax"])
	}
}

func TestPokedexSuggestions(t *testing.T) {
	m := dexTestModel()
	if got, _ := m.pokedexSuggestions(""); len(got) != len(m.dexOrder) {
		t.Errorf("empty query lists the whole Pokédex, got %v", got)
	}
	// Exact match first, then the rest in Pokédex order.
	if got, _ := m.pokedexSuggestions("mew"); !reflect.DeepEqual(got, []string{"mew", "mewtwo"}) {
		t.Errorf("mew: got %v", got)
	}
	// Filters need their data first…
	if got, note := m.pokedexSuggestions("type:fire gen:1"); got != nil || note == "" {
		t.Errorf("missing filter data should give a note, got %v %q", got, note)
	}
	// …then combine with each other and with a name.
	m.filterSets["type:fire"] = map[string]bool{"charmander": true, "charizard": true, "charizard-gmax": true, "moltres": true}
	m.filterSets["type:flying"] = map[string]bool{"charizard": true, "charizard-gmax": true, "moltres": true}
	m.filterSets["gen:1"] = map[string]bool{"charmander": true, "charizard": true, "moltres": true}
	got, _ := m.pokedexSuggestions("type:fire type:flying gen:1")
	if want := []string{"charizard", "charizard-gmax", "moltres"}; !reflect.DeepEqual(got, want) {
		t.Errorf("filters: got %v, want %v", got, want)
	}
	if got, _ := m.pokedexSuggestions("type:fire mol"); !reflect.DeepEqual(got, []string{"moltres"}) {
		t.Errorf("filter + name: got %v", got)
	}
	if _, note := m.pokedexSuggestions("type:fyre"); note == "" {
		t.Error("an unknown type should be reported")
	}
}

func TestNavigationHistory(t *testing.T) {
	m := dexTestModel()
	charizard := dex.Pokemon{Name: "charizard", Types: []string{"fire", "flying"}}
	m.pokeCache["charizard"] = charizard
	m.pokeCache["moltres"] = dex.Pokemon{Name: "moltres"}
	m.moves["fly"] = dex.MoveInfo{Name: "fly", DisplayName: "Fly", LearnedBy: []string{"moltres", "charizard"}}

	m.openDetail(charizard)
	m.detail.tab = tabMoves
	m.openList("move", "fly", false) // from the entry: pushed on the history
	if m.screen != screenList || len(m.history) != 1 {
		t.Fatalf("screen %v, history %d", m.screen, len(m.history))
	}
	if !reflect.DeepEqual(m.list.members, []string{"charizard", "moltres"}) {
		t.Errorf("list members should be in Pokédex order, got %v", m.list.members)
	}
	m.list.sel = 1
	m.openPokemon("moltres")
	if m.screen != screenDetail || m.detail.p.Name != "moltres" || m.detail.tab != tabOverview {
		t.Fatalf("opening a Pokémon should show its Overview, got %v %q tab %v", m.screen, m.detail.p.Name, m.detail.tab)
	}
	m.back()
	if m.screen != screenList || m.list.sel != 1 {
		t.Fatalf("back should return to the list where we were, got %v sel %d", m.screen, m.list.sel)
	}
	m.back()
	if m.screen != screenDetail || m.detail.p.Name != "charizard" || m.detail.tab != tabMoves {
		t.Fatalf("back should return to Charizard's Moves tab, got %v %q tab %v", m.screen, m.detail.p.Name, m.detail.tab)
	}
	m.back()
	if m.screen != screenSearch {
		t.Fatalf("back with an empty history returns to the search, got %v", m.screen)
	}
}
