package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/JoaoASimoes/pokedex/internal/dex"
)

// The Family tab lists Pokémon related to the current one (its evolution line and the
// other forms of its species) and lets you jump to them. Esc walks back through the
// Pokémon you came from.

type familyKind int

const (
	famEvolution familyKind = iota
	famForm
	famFemale // the current Pokémon's female look (only the sprite differs)
)

type familyItem struct {
	kind    familyKind
	name    string // Pokémon to open (evolution/form)
	label   string
	depth   int    // indentation in the evolution tree
	how     string // how it evolves from the previous stage
	current bool
}

// femaleKey is the sprite cache key of a Pokémon's female look.
func femaleKey(name string) string { return name + "~female" }

// pokemonForSpecies maps a species to its default Pokémon (e.g. giratina → giratina-altered).
func (m tuiModel) pokemonForSpecies(species string) string {
	if m.data.IsPokemon(species) {
		return species
	}
	for _, n := range m.data.KnownPokemon() {
		if strings.HasPrefix(n, species+"-") {
			return n
		}
	}
	return species
}

func (m tuiModel) familyItems() []familyItem {
	p := m.detail.p
	var items []familyItem

	if p.Evolution != nil && len(p.Evolution.Next) > 0 {
		var walk func(n *dex.EvoNode, depth int)
		walk = func(n *dex.EvoNode, depth int) {
			name := m.pokemonForSpecies(n.Species)
			if n.Species == p.Species {
				name = p.Name // stay on the current form
			}
			items = append(items, familyItem{
				kind: famEvolution, name: name, label: dex.DisplayName(n.Species), depth: depth,
				how: dex.HowText(n.How()), current: n.Species == p.Species,
			})
			for i := range n.Next {
				walk(&n.Next[i], depth+1)
			}
		}
		walk(p.Evolution, 0)
		items[0].how = "" // the first stage does not evolve from anything
	}

	if len(p.Varieties) > 1 {
		for _, v := range p.Varieties {
			items = append(items, familyItem{kind: famForm, name: v, label: dex.DisplayName(v), current: v == p.Name})
		}
	}
	if p.SpriteFemale != "" {
		items = append(items, familyItem{kind: famFemale, name: p.Name, label: "♀ Female look"})
	}
	return items
}

// previewFamily loads what the right panel needs for the selected item.
func (m *tuiModel) previewFamily() tea.Cmd {
	items := m.familyItems()
	if len(items) == 0 {
		return nil
	}
	it := items[clamp(m.detail.familySel, 0, len(items)-1)]
	if it.kind == famFemale {
		return m.fetchSpriteKey(femaleKey(it.name), m.detail.p.SpriteFemale)
	}
	if p, ok := m.pokeCache[it.name]; ok {
		return m.fetchSprite(p)
	}
	return m.loadRelated(it.name)
}

func (m *tuiModel) loadRelated(name string) tea.Cmd {
	key := "pokemon:" + name
	if m.pending[key] {
		return nil
	}
	m.pending[key] = true
	d := m.data
	return tea.Batch(m.spin(), func() tea.Msg {
		p, err := d.Pokemon(name)
		return relatedLoadedMsg{name, p, err}
	})
}

// openFamily opens the selected evolution or form (on its Overview tab).
func (m *tuiModel) openFamily() tea.Cmd {
	items := m.familyItems()
	if len(items) == 0 {
		return nil
	}
	it := items[clamp(m.detail.familySel, 0, len(items)-1)]
	if it.current || it.kind == famFemale {
		return nil
	}
	return m.openPokemon(it.name)
}

// currentFamilyIndex is the position of the current Pokémon in the Family list.
func (m tuiModel) currentFamilyIndex() int {
	for i, it := range m.familyItems() {
		if it.current {
			return i
		}
	}
	return 0
}

// ----------------------------------------------------------------------------- view

func (m tuiModel) viewFamily(totalW, bodyH, avail int) string {
	items := m.familyItems()
	const leftW = 34
	rightW := max(30, totalW-leftW-9)
	if len(items) == 0 {
		return twoPanels(labelStyle.Render("No evolutions or other forms."), leftW, "", rightW, 6)
	}
	sel := clamp(m.detail.familySel, 0, len(items)-1)

	var list strings.Builder
	lastKind := familyKind(-1)
	for i, it := range items {
		if it.kind != lastKind {
			if i > 0 {
				list.WriteString("\n")
			}
			list.WriteString(sectionStyle.Render(map[familyKind]string{
				famEvolution: "Evolution line", famForm: "Forms", famFemale: "Gender differences",
			}[it.kind]) + "\n")
			lastKind = it.kind
		}
		indent := strings.Repeat("  ", it.depth)
		name := truncate(it.label, leftW-10-len(indent))
		suffix := ""
		if it.current {
			suffix = labelStyle.Render("  (this)")
		}
		if i == sel {
			list.WriteString(selectedStyle.Render("▸ "+indent+name) + suffix + "\n")
		} else {
			list.WriteString("  " + indent + name + suffix + "\n")
		}
	}

	right := m.clip(m.familyCard(items[sel], rightW), avail, m.scrollKeys())
	return twoPanels(strings.TrimRight(list.String(), "\n"), leftW, right, rightW, bodyH)
}

func (m tuiModel) familyCard(it familyItem, width int) string {
	if it.kind == famFemale {
		p := m.detail.p
		sprite := m.spriteViewKey(femaleKey(p.Name), 30, 15)
		if sprite == "" {
			sprite = m.spinner.View() + " Loading…"
		}
		return headingStyle.Render(dex.DisplayName(p.Name)+" ♀") + "\n" +
			labelStyle.Render("Females look slightly different; stats and moves are the same.") + "\n\n" + sprite
	}

	p, ok := m.pokeCache[it.name]
	if !ok {
		return headingStyle.Render(it.label) + "\n\n" + m.spinner.View() + " Loading…"
	}
	var extra []func(int) string
	if it.how != "" {
		extra = append(extra, func(w int) string {
			return sectionStyle.Render("How it evolves") + "\n" + wrap(it.how, w)
		})
	}
	extra = append(extra, func(w int) string {
		return sectionStyle.Render("Abilities") + "\n" + wrap(renderAbilityLine(p), w)
	})
	parts := []string{
		renderIntro(p, width, m.spriteView(p.Name, 24, 12), extra...),
		sectionStyle.Render("Base stats") + "\n" + renderStats(p, width),
	}
	hint := "enter open this entry"
	if it.current {
		hint = "this is the entry you are viewing"
	}
	parts = append(parts, labelStyle.Render(hint))
	return strings.Join(parts, "\n\n")
}
