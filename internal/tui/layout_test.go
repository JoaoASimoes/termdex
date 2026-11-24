package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/termdex/internal/config"
	"github.com/JoaoASimoes/termdex/internal/dex"
)

// A frame larger than the terminal makes it scroll, and the renderer then leaves stray
// text on screen. Every screen must fit, before fitScreen's safety cut.
func TestScreensFitTheTerminal(t *testing.T) {
	chart := dex.TypeChart{"fire": {"water": 2, "rock": 2}, "flying": {"rock": 2, "ground": 0}, "water": {}, "rock": {}, "ground": {}}
	d := dex.NewOfflineClientIDs([]dex.PokemonEntry{
		{Name: "charmander", ID: 4}, {Name: "charmeleon", ID: 5}, {Name: "charizard", ID: 6},
		{Name: "charizard-mega-x", ID: 10034},
	}, chart)

	var moves []dex.MoveRef
	for i := 0; i < 80; i++ {
		moves = append(moves, dex.MoveRef{Name: fmt.Sprintf("move-number-%d", i), Method: "level-up", Level: i})
	}
	p := dex.Pokemon{
		Name: "charizard", ID: 6, SpeciesID: 6, Species: "charizard", Types: []string{"fire", "flying"},
		Genus: "Flame Pokémon", Generation: "generation-i", Height: 17, Weight: 905,
		Flavor:    strings.Repeat("It spits fire that is hot enough to melt boulders. ", 4),
		Abilities: []dex.AbilityRef{{Name: "blaze"}, {Name: "solar-power", Hidden: true}},
		Stats: []dex.Stat{{Name: "hp", Base: 78}, {Name: "attack", Base: 84}, {Name: "defense", Base: 78},
			{Name: "special-attack", Base: 109}, {Name: "special-defense", Base: 85}, {Name: "speed", Base: 100}},
		Learnsets: []dex.Learnset{{VersionGroup: "scarlet-violet", Moves: moves}},
		Varieties: []string{"charizard", "charizard-mega-x"},
		Evolution: &dex.EvoNode{Species: "charmander", Next: []dex.EvoNode{
			{Species: "charmeleon", Next: []dex.EvoNode{{Species: "charizard"}}}}},
	}

	for _, size := range [][2]int{{80, 24}, {100, 30}, {120, 42}, {160, 50}} {
		m := newTUIModel(d, config.Default())
		m.dataReady = true
		m.buildDexOrder()
		m.width, m.height = size[0], size[1]
		m.pokeCache[p.Name] = p
		m.moves["move-number-3"] = dex.MoveInfo{Name: "move-number-3", DisplayName: "Move Number 3", Type: "fire",
			Class: "special", ShortEffect: strings.Repeat("Has a chance to burn the target. ", 6),
			LearnedBy: []string{"charmander", "charmeleon", "charizard"}}

		check := func(screen, view string) {
			t.Helper()
			if h, w := lipgloss.Height(view), lipgloss.Width(view); h > size[1] || w > size[0] {
				t.Errorf("%dx%d %s: frame is %dx%d", size[0], size[1], screen, w, h)
			}
		}

		m.search.inputs[fieldPokemon].SetValue("char")
		m.refreshSuggestions()
		m.search.preview = &p
		check("pokedex", m.viewSearch())
		for _, f := range []field{fieldType1, fieldNature} {
			m.setFocus(f)
			check(fmt.Sprint("search field ", f), m.viewSearch())
		}

		m.openDetail(p)
		for _, tab := range []detailTab{tabOverview, tabAbilities, tabMoves, tabFamily} {
			m.setTab(tab)
			m.detail.moveSel = 3
			check(fmt.Sprint("entry tab ", tab), m.viewDetail())
		}

		m.openList("move", "move-number-3", false)
		check("list", m.viewList())

		m.showHelp = true
		check("help", m.viewHelp())
	}
}
