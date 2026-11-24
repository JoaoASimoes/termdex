package tui

import (
	"github.com/JoaoASimoes/pokedex/internal/dex"
)

import "testing"

func TestFamilyItems(t *testing.T) {
	d := dex.NewOfflineClient([]string{"charmander", "charmeleon", "charizard", "charizard-mega-x"}, nil)
	evo := &dex.EvoNode{Species: "charmander", Next: []dex.EvoNode{
		{Species: "charmeleon", Next: []dex.EvoNode{{Species: "charizard"}}},
	}}
	m := tuiModel{data: d}
	m.detail.p = dex.Pokemon{Name: "charizard", Species: "charizard", Evolution: evo,
		Varieties: []string{"charizard", "charizard-mega-x"}, SpriteFemale: "x.png"}

	items := m.familyItems()
	want := []struct {
		kind    familyKind
		name    string
		depth   int
		current bool
	}{
		{famEvolution, "charmander", 0, false},
		{famEvolution, "charmeleon", 1, false},
		{famEvolution, "charizard", 2, true},
		{famForm, "charizard", 0, true},
		{famForm, "charizard-mega-x", 0, false},
		{famFemale, "charizard", 0, false},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d: %+v", len(items), len(want), items)
	}
	for i, w := range want {
		it := items[i]
		if it.kind != w.kind || it.name != w.name || it.depth != w.depth || it.current != w.current {
			t.Errorf("item %d = %+v, want %+v", i, it, w)
		}
	}
	if m.currentFamilyIndex() != 2 {
		t.Errorf("currentFamilyIndex = %d, want 2", m.currentFamilyIndex())
	}
}

func TestPokemonForSpecies(t *testing.T) {
	d := dex.NewOfflineClient([]string{"giratina-altered", "giratina-origin"}, nil)
	m := tuiModel{data: d}
	if got := m.pokemonForSpecies("giratina"); got != "giratina-altered" {
		t.Errorf("pokemonForSpecies(giratina) = %q", got)
	}
}
