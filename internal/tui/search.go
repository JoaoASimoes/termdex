package tui

import (
	"fmt"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/fuzzy"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type field int

const (
	fieldPokemon field = iota
	fieldNature
	fieldType1
	fieldType2
	numFields
)

// noType is the "none" entry of the Type 2 list.
const noType = ""

var fieldLabels = [numFields]string{"Search Pokémon", "Search nature or stat", "Type 1", "Type 2 (optional)"}

// Search screen tabs, in tab order: Pokédex → Natures → Type Matchups.
var searchTabs = []string{"Pokédex", "Natures", "Type Matchups"}

// tabFields is the field focused when a search tab is opened.
var tabFields = []field{fieldPokemon, fieldNature, fieldType1}

// switchTab moves to the previous/next search tab, wrapping around.
func (m *tuiModel) switchTab(delta int) tea.Cmd {
	n := len(tabFields)
	return m.setFocus(tabFields[((tabOf(m.search.focus)+delta)%n+n)%n])
}

func tabOf(f field) int {
	switch f {
	case fieldPokemon:
		return 0
	case fieldNature:
		return 1
	default:
		return 2
	}
}

type searchState struct {
	inputs      [numFields]textinput.Model
	focus       field
	suggestions []string
	selected    int

	preview    *dex.Pokemon // Pokédex entry shown on the right
	openOnLoad string       // open the detail screen once this Pokémon is loaded

	type1, type2 string   // confirmed types
	typeResult   []string // type combination shown on the right

	nature string // nature shown on the right
}

func newSearchState() searchState {
	var s searchState
	placeholders := [numFields]string{"e.g. garchomp", "e.g. adamant, speed, -spa", "e.g. fire", "e.g. flying"}
	for i := range s.inputs {
		in := textinput.New()
		in.Prompt = "› "
		in.Placeholder = placeholders[i]
		in.CharLimit = 40
		in.SetWidth(leftWidth - 3)
		s.inputs[i] = in
	}
	s.inputs[fieldPokemon].Focus()
	return s
}

func (m *tuiModel) setFocus(f field) tea.Cmd {
	s := &m.search
	s.inputs[s.focus].Blur()
	s.focus = (f + numFields) % numFields
	m.refreshSuggestions()
	cmds := []tea.Cmd{s.inputs[s.focus].Focus()}
	if s.focus == fieldNature {
		cmds = append(cmds, m.fetchNatures(), m.preview())
	}
	return tea.Batch(cmds...)
}

func (m *tuiModel) refreshSuggestions() {
	s := &m.search
	s.selected = 0
	s.suggestions = nil
	if !m.dataReady {
		return
	}
	q := fuzzy.Normalize(s.inputs[s.focus].Value())
	switch {
	case s.focus == fieldPokemon && q != "":
		s.suggestions = fuzzy.Suggest(q, m.data.KnownPokemon(), maxSuggestions)
	case s.focus == fieldPokemon:
	case s.focus == fieldNature:
		if m.natures != nil {
			s.suggestions = dex.SuggestNatures(q, m.natures)
		}
	case q == "" && s.focus == fieldType2:
		// "" is the "none" option: a single-type lookup.
		s.suggestions = append([]string{noType}, m.data.TypeNames()...)
	case q == "":
		s.suggestions = m.data.TypeNames()
	default:
		s.suggestions = fuzzy.Suggest(q, m.data.TypeNames(), maxSuggestions)
	}
}

// showPokemon previews name right away if it is cached, otherwise loads it
// (after a short pause when debounce is set).
func (m *tuiModel) showPokemon(name string, debounce bool) tea.Cmd {
	m.seq++
	if p, ok := m.pokeCache[name]; ok {
		m.loading = ""
		m.search.preview = &p
		return m.fetchSprite(p)
	}
	seq := m.seq
	if !debounce {
		m.loading = "Loading " + dex.DisplayName(name) + "…"
		return tea.Batch(m.spin(), loadPokemonCmd(m.data, seq, name))
	}
	return tea.Tick(m.previewDelay, func(time.Time) tea.Msg { return previewMsg{seq, name} })
}

// preview shows the result for the highlighted suggestion without confirming it.
func (m *tuiModel) preview() tea.Cmd {
	s := &m.search
	if !m.dataReady || len(s.suggestions) == 0 {
		return nil
	}
	pick := s.suggestions[s.selected]
	switch s.focus {
	case fieldPokemon:
		return m.showPokemon(pick, true)
	case fieldNature:
		s.nature = pick
	case fieldType1:
		s.typeResult = dex.UniqueTypes(pick, s.type2)
	case fieldType2:
		s.typeResult = dex.UniqueTypes(s.type1, pick)
	}
	return nil
}

// confirm handles Enter on the focused field.
func (m *tuiModel) confirm() tea.Cmd {
	s := &m.search
	if !m.dataReady {
		return nil
	}
	m.status = ""
	in := &s.inputs[s.focus]
	pick := ""
	if len(s.suggestions) > 0 {
		pick = s.suggestions[s.selected]
	}

	if s.focus == fieldPokemon {
		if pick == "" {
			if in.Value() != "" {
				m.status = fmt.Sprintf("No Pokémon matches '%s'.", in.Value())
			}
			return nil
		}
		m.history = nil // a fresh entry from the search starts a new trail
		if p, ok := m.pokeCache[pick]; ok {
			m.search.preview = &p
			return m.openDetail(p)
		}
		s.openOnLoad = pick
		return m.showPokemon(pick, false)
	}

	if s.focus == fieldNature {
		if pick == "" {
			m.status = fmt.Sprintf("No nature matches '%s'.", in.Value())
			return nil
		}
		s.nature = pick
		in.SetValue(pick)
		in.CursorEnd()
		m.refreshSuggestions()
		return nil
	}

	if len(s.suggestions) == 0 {
		if strings.TrimSpace(in.Value()) != "" {
			m.status = fmt.Sprintf("No type matches '%s'.", in.Value())
		}
		return nil
	}
	in.SetValue(pick) // the highlighted type ("" for the "none" option)
	in.CursorEnd()
	if s.focus == fieldType1 {
		s.type1 = pick
	} else {
		s.type2 = pick
	}
	s.typeResult = dex.UniqueTypes(s.type1, s.type2)
	if s.focus == fieldType1 && pick != "" && s.type2 == "" {
		return m.setFocus(fieldType2)
	}
	m.refreshSuggestions()
	return nil
}

func (m tuiModel) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	s := &m.search
	if key, ok := msg.(tea.KeyPressMsg); ok {
		k, keys := key.String(), m.keys.Search
		switch {
		case keys.Clear.Matches(k):
			if s.inputs[s.focus].Value() == "" {
				return m, tea.Quit
			}
			s.inputs[s.focus].SetValue("")
			m.refreshSuggestions()
			return m, nil
		case keys.NextField.Matches(k), keys.PrevField.Matches(k):
			// Only moves between the boxes of the current tab (Type 1 <-> Type 2).
			switch s.focus {
			case fieldType1:
				return m, m.setFocus(fieldType2)
			case fieldType2:
				return m, m.setFocus(fieldType1)
			}
			return m, nil
		case keys.NextTab.Matches(k):
			return m, m.switchTab(1)
		case keys.PrevTab.Matches(k):
			return m, m.switchTab(-1)
		case keys.Up.Matches(k):
			if s.selected > 0 {
				s.selected--
				return m, m.preview()
			}
			return m, nil
		case keys.Down.Matches(k):
			if s.selected < len(s.suggestions)-1 {
				s.selected++
				return m, m.preview()
			}
			return m, nil
		case keys.Confirm.Matches(k):
			return m, m.confirm()
		}
	}

	// Everything else (typing, cursor blink, …) goes to the focused input.
	before := s.inputs[s.focus].Value()
	var cmd tea.Cmd
	s.inputs[s.focus], cmd = s.inputs[s.focus].Update(msg)
	if s.inputs[s.focus].Value() != before {
		m.status = ""
		m.refreshSuggestions()
		return m, tea.Batch(cmd, m.preview())
	}
	return m, cmd
}

// ----------------------------------------------------------------------------- view

func (m tuiModel) searchPanel() string {
	s := m.search
	var b strings.Builder
	fields := []field{s.focus}
	if s.focus == fieldType1 || s.focus == fieldType2 {
		fields = []field{fieldType1, fieldType2}
	}
	for i, f := range fields {
		label := labelStyle
		if f == s.focus {
			label = labelFocused
		}
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(label.Render(fieldLabels[f]) + "\n")
		b.WriteString(s.inputs[f].View() + "\n")
	}

	b.WriteString("\n")
	switch {
	case !m.dataReady:
	case s.focus == fieldNature && m.natures == nil:
		b.WriteString(m.spinner.View() + helpStyle.Render(" Loading natures…"))
	case len(s.suggestions) == 0 && s.focus == fieldPokemon && s.inputs[fieldPokemon].Value() == "":
		b.WriteString(helpStyle.Render("Start typing a name…"))
	case len(s.suggestions) == 0:
		b.WriteString(helpStyle.Render("No matches"))
	default:
		for i, sug := range s.suggestions {
			if sug == noType {
				sug = "— none (single type) —"
			}
			hint := ""
			if s.focus == fieldNature {
				if n, ok := dex.NatureByName(m.natures, sug); ok {
					hint = natureHint(n)
				}
			}
			if s.focus != fieldPokemon {
				sug = dex.Capitalize(sug)
			}
			if i == s.selected {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("▸ %-9s", sug)) + hint + "\n")
			} else {
				b.WriteString(fmt.Sprintf("  %-9s", sug) + hint + "\n")
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) searchResultPanel(width int) string {
	s := m.search
	if s.focus == fieldPokemon {
		if s.preview == nil {
			return helpStyle.Render("Type a Pokémon name. The Pokédex entry of the\n" +
				"highlighted suggestion appears here.\n\n" +
				"Press enter to open the full entry: abilities,\nmove pool and more.")
		}
		// Shrink the sprite on short terminals so the whole card stays visible.
		rows := 13
		if m.height > 0 {
			rows = clamp(m.height-30, 6, 13)
		}
		return renderPokemonCard(m.data.Chart(), *s.preview, width, m.spriteView(s.preview.Name, rows*2, rows))
	}
	if s.focus == fieldNature {
		return renderNaturePanel(m.natures, s.nature, width)
	}
	if len(s.typeResult) == 0 {
		return helpStyle.Render("Pick one or two types to see their\nweaknesses and resistances.")
	}
	return headingStyle.Render("Type combination") + "\n" +
		strings.Join(wrapBadges(s.typeResult, width), "\n") + "\n\n" +
		renderMatchups(m.data.Chart(), s.typeResult, width, false)
}

func (m tuiModel) viewSearch() string {
	s := m.search
	mode := tabOf(s.focus)
	tabs := renderTabs(searchTabs, mode)

	rightWidth := min(maxRightWidth, max(minRightWidth, m.width-leftWidth-8))
	stacked := m.width > 0 && m.width < leftWidth+minRightWidth+8
	if stacked {
		rightWidth = max(20, m.width-4)
	}

	left := m.searchPanel()
	right := m.searchResultPanel(rightWidth)
	leftBox := lipgloss.NewStyle().Width(leftWidth)
	rightBox := lipgloss.NewStyle().Width(rightWidth)

	var body string
	if stacked {
		body = lipgloss.JoinVertical(lipgloss.Left,
			panelFocusedStyle.Render(leftBox.Render(left)),
			panelStyle.Render(rightBox.Render(right)))
	} else {
		h := max(lipgloss.Height(left), lipgloss.Height(right))
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			panelFocusedStyle.Render(leftBox.Height(h).Render(left)),
			" ",
			panelStyle.Render(rightBox.Height(h).Render(right)))
	}

	help := m.searchHelp(mode)
	return lipgloss.JoinVertical(lipgloss.Left, m.header(""), "", tabs, "", body, m.footer(help))
}
