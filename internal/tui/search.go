package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/termdex/internal/dex"
	"github.com/JoaoASimoes/termdex/internal/fuzzy"
)

type field int

const (
	fieldPokemon field = iota
	fieldMove
	fieldItem
	fieldNature
	fieldType1
	fieldType2
	numFields
)

// noType is the "none" entry of the Type 2 list.
const noType = ""

// maxCatalogMatches caps the move/item matches listed while typing.
const maxCatalogMatches = 300

var fieldLabels = [numFields]string{
	"Search Pokémon", "Search moves", "Search items", "Search nature or stat", "Type 1", "Type 2 (optional)",
}

// Search screen tabs, in tab order.
var searchTabs = []string{"Pokédex", "Moves", "Items", "Natures", "Type Matchups"}

// tabFields is the field focused when a search tab is opened.
var tabFields = []field{fieldPokemon, fieldMove, fieldItem, fieldNature, fieldType1}

const (
	modePokedex = iota
	modeMoves
	modeItems
	modeNatures
	modeTypes
)

// switchTab moves to the previous/next search tab, wrapping around.
func (m *tuiModel) switchTab(delta int) tea.Cmd {
	n := len(tabFields)
	return m.setFocus(tabFields[((tabOf(m.search.focus)+delta)%n+n)%n])
}

func tabOf(f field) int {
	switch f {
	case fieldPokemon:
		return modePokedex
	case fieldMove:
		return modeMoves
	case fieldItem:
		return modeItems
	case fieldNature:
		return modeNatures
	default:
		return modeTypes
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
	move   string // move shown on the right
	item   string // item shown on the right
	note   string // Pokédex filter message ("Loading type:fire…", "Unknown type …")
}

func newSearchState() searchState {
	var s searchState
	placeholders := [numFields]string{
		"name, type:fire, gen:1", "e.g. thunderbolt", "e.g. leftovers", "e.g. adamant, speed, -spa", "e.g. fire", "e.g. flying",
	}
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
	switch s.focus {
	case fieldNature:
		cmds = append(cmds, m.fetchNatures(), m.preview())
	case fieldMove:
		cmds = append(cmds, m.fetchCatalog(fieldMove), m.preview())
	case fieldItem:
		cmds = append(cmds, m.fetchCatalog(fieldItem), m.preview())
	}
	return tea.Batch(cmds...)
}

func (m *tuiModel) refreshSuggestions() {
	s := &m.search
	s.selected = 0
	s.suggestions = nil
	s.note = ""
	if !m.dataReady {
		return
	}
	q := fuzzy.Normalize(s.inputs[s.focus].Value())
	switch {
	case s.focus == fieldPokemon:
		// The whole Pokédex in number order, or the filtered/matching part of it.
		s.suggestions, s.note = m.pokedexSuggestions(s.inputs[fieldPokemon].Value())
	case s.focus == fieldMove || s.focus == fieldItem:
		names := m.moveNames
		if s.focus == fieldItem {
			names = m.itemNames
		}
		if q == "" {
			s.suggestions = names // the whole catalog
		} else {
			s.suggestions = fuzzy.Suggest(q, names, maxCatalogMatches)
		}
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

// showCatalog previews a move or item: at once when loaded, otherwise after a short
// pause so scrolling quickly through the list does not fire a request per row.
func (m *tuiModel) showCatalog(kind field, name string) tea.Cmd {
	m.catSeq++
	if kind == fieldMove {
		if _, ok := m.moves[name]; ok {
			return nil
		}
	} else if it, ok := m.items[name]; ok {
		return m.fetchSpriteKey(itemSpriteKey(name), it.Sprite)
	}
	seq := m.catSeq
	return tea.Tick(m.previewDelay, func(time.Time) tea.Msg { return catalogPreviewMsg{seq, kind, name} })
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
	case fieldMove:
		s.move = pick
		return m.showCatalog(fieldMove, pick)
	case fieldItem:
		s.item = pick
		return m.showCatalog(fieldItem, pick)
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

	switch s.focus {
	case fieldPokemon:
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

	case fieldMove, fieldItem: // the Pokémon that learn the move / hold the item
		if pick == "" {
			m.status = fmt.Sprintf("Nothing matches '%s'.", in.Value())
			return nil
		}
		kind := "move"
		if s.focus == fieldItem {
			kind = "item"
		}
		return m.openList(kind, pick, true)

	case fieldNature:
		if pick == "" {
			m.status = fmt.Sprintf("Nothing matches '%s'.", in.Value())
			return nil
		}
		in.SetValue(pick)
		in.CursorEnd()
		m.refreshSuggestions()
		return m.preview()
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
			return m, m.preview()
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
		return m, tea.Batch(cmd, m.fetchFilters(), m.preview())
	}
	return m, cmd
}

// ----------------------------------------------------------------------------- view

// listRows is how many suggestions fit on screen; longer lists scroll.
func (m tuiModel) listRows() int {
	if m.height <= 0 {
		return 20
	}
	extra := 0
	if m.search.focus == fieldType1 || m.search.focus == fieldType2 {
		extra = 3 // two boxes
	}
	if m.search.focus == fieldMove || m.search.focus == fieldItem || m.search.focus == fieldPokemon {
		extra = 1 // the "N of M" line
	}
	return max(5, m.height-15-extra)
}

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
	catalog := s.focus == fieldMove || s.focus == fieldItem
	switch {
	case !m.dataReady:
	case s.focus == fieldNature && m.natures == nil:
		b.WriteString(m.spinner.View() + helpStyle.Render(" Loading natures…"))
	case s.focus == fieldMove && m.moveNames == nil:
		b.WriteString(m.spinner.View() + helpStyle.Render(" Loading moves…"))
	case s.focus == fieldItem && m.itemNames == nil:
		b.WriteString(m.spinner.View() + helpStyle.Render(" Loading items…"))
	case s.note != "":
		b.WriteString(helpStyle.Render(s.note))
	case len(s.suggestions) == 0:
		b.WriteString(helpStyle.Render("No matches"))
	default:
		if catalog || s.focus == fieldPokemon {
			total := len(m.moveNames)
			switch s.focus {
			case fieldItem:
				total = len(m.itemNames)
			case fieldPokemon:
				total = len(m.dexOrder)
			}
			b.WriteString(labelStyle.Render(fmt.Sprintf("%d of %d", len(s.suggestions), total)) + "\n")
		}
		rows := m.listRows()
		start := clamp(s.selected-rows/2, 0, max(0, len(s.suggestions)-rows))
		end := min(len(s.suggestions), start+rows)
		for i := start; i < end; i++ {
			b.WriteString(m.suggestionRow(s.suggestions[i], i == s.selected) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m tuiModel) suggestionRow(sug string, selected bool) string {
	s := m.search
	hint := ""
	label := sug
	switch s.focus {
	case fieldPokemon:
		return m.dexRow(sug, selected, leftWidth)
	case fieldMove, fieldItem:
		label = truncate(dex.DisplayName(sug), leftWidth-3)
		if s.focus == fieldMove {
			if mv, ok := m.moves[sug]; ok {
				label = truncate(mv.DisplayName, leftWidth-3)
			}
		} else if it, ok := m.items[sug]; ok {
			label = truncate(it.DisplayName, leftWidth-3)
		}
	case fieldNature:
		if n, ok := dex.NatureByName(m.natures, sug); ok {
			hint = natureHint(n)
		}
		label = fmt.Sprintf("%-9s", dex.Capitalize(sug))
	default:
		if sug == noType {
			label = "— none (single type) —"
		} else {
			label = dex.Capitalize(sug)
		}
	}
	if selected {
		return selectedStyle.Render("▸ "+label) + hint
	}
	return "  " + label + hint
}

func (m tuiModel) searchResultPanel(width int) string {
	s := m.search
	switch s.focus {
	case fieldPokemon:
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
	case fieldMove:
		if s.move == "" {
			return helpStyle.Render("Every move in the games. Scroll the list or type\nto search; details appear here.")
		}
		mv, ok := m.moves[s.move]
		if !ok {
			return headingStyle.Render(dex.DisplayName(s.move)) + "\n\n" + m.spinner.View() + " Loading…"
		}
		return renderMoveInfo(mv, nil, width)
	case fieldItem:
		if s.item == "" {
			return helpStyle.Render("Every item in the games. Scroll the list or type\nto search; details appear here.")
		}
		it, ok := m.items[s.item]
		if !ok {
			return headingStyle.Render(dex.DisplayName(s.item)) + "\n\n" + m.spinner.View() + " Loading…"
		}
		return renderItemInfo(it, width, m.spriteViewKey(itemSpriteKey(it.Name), 24, 12))
	case fieldNature:
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

	// Two panels take 9 columns besides their content (borders, padding, gap).
	rightWidth := min(maxRightWidth, max(minRightWidth, m.width-leftWidth-9))
	stacked := m.width > 0 && m.width < leftWidth+minRightWidth+9
	if stacked {
		rightWidth = max(20, m.width-4)
	}

	help := m.searchHelp(mode)
	foot := m.footer(help)
	left := m.searchPanel()
	right := m.clip(m.searchResultPanel(rightWidth), m.availHeight(foot),
		pair(m.keys.Search.ScrollUp, m.keys.Search.ScrollDown))
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

	return lipgloss.JoinVertical(lipgloss.Left, m.header(""), "", tabs, "", body, foot)
}
