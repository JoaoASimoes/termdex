package tui

import (
	"reflect"
	"testing"

	"github.com/JoaoASimoes/termdex/internal/config"
	"github.com/JoaoASimoes/termdex/internal/dex"
)

func newTestModel() tuiModel {
	chart := dex.TypeChart{"bug": {}, "dark": {}, "dragon": {}, "fire": {}}
	m := newTUIModel(dex.NewOfflineClient(nil, chart), config.Default())
	m.dataReady = true
	return m
}

// Enter on an empty type field must pick the highlighted type, not clear the field.
func TestConfirmEmptyTypeFieldPicksHighlighted(t *testing.T) {
	m := newTestModel()
	m.setFocus(fieldType1)
	m.search.selected = 2 // Bug, Dark, [Dragon], Fire
	m.confirm()

	if m.search.type1 != "dragon" {
		t.Fatalf("type1 = %q, want dragon", m.search.type1)
	}
	if m.search.focus != fieldType2 {
		t.Fatalf("focus = %v, want Type 2", m.search.focus)
	}

	// Type 2 starts with the "none" option, then the types.
	if m.search.suggestions[0] != noType {
		t.Fatalf("Type 2 list should start with the none option, got %v", m.search.suggestions)
	}
	m.search.selected = 4 // none, Bug, Dark, Dragon, [Fire]
	m.confirm()
	if got, want := m.search.typeResult, []string{"dragon", "fire"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("typeResult = %v, want %v", got, want)
	}

	// Picking "none" goes back to a single type.
	m.search.inputs[fieldType2].SetValue("")
	m.refreshSuggestions()
	m.confirm()
	if got, want := m.search.typeResult, []string{"dragon"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("typeResult = %v, want %v", got, want)
	}
}

func TestTabsCycleThroughCatalogs(t *testing.T) {
	m := newTestModel()
	var seen []int
	for range searchTabs {
		m.switchTab(1)
		seen = append(seen, tabOf(m.search.focus))
	}
	if want := []int{modeMoves, modeItems, modeNatures, modeTypes, modePokedex}; !reflect.DeepEqual(seen, want) {
		t.Fatalf("tab order = %v, want %v", seen, want)
	}
}

func TestCatalogListsEverythingThenFilters(t *testing.T) {
	m := newTestModel()
	m.moveNames = []string{"absorb", "thunder", "thunderbolt", "tackle"}
	m.setFocus(fieldMove)
	if len(m.search.suggestions) != 4 {
		t.Fatalf("empty search should list every move, got %v", m.search.suggestions)
	}
	m.search.inputs[fieldMove].SetValue("thunderbot") // typo
	m.refreshSuggestions()
	if len(m.search.suggestions) == 0 || m.search.suggestions[0] != "thunderbolt" {
		t.Fatalf("got %v, want thunderbolt first", m.search.suggestions)
	}
}
