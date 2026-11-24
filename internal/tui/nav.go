package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/termdex/internal/dex"
)

// Navigation: besides the search screen there are two kinds of pages, a Pokémon entry
// (screenDetail) and a list of Pokémon related to a move, item or ability (screenList).
// Opening a page from another one pushes the current page on m.history; the back key pops
// it, and with an empty history it returns to the search screen.

// page is a snapshot of an entry or list page, kept in the history.
type page struct {
	screen screen
	detail detailState
	list   listState
}

// listState is a list of Pokémon related to a move, item or ability.
type listState struct {
	kind    string // "move", "item" or "ability"
	name    string
	title   string   // display name of the move/item/ability
	caption string   // e.g. "Pokémon that learn Thunderbolt"
	members []string // in Pokédex order
	sel     int
}

func (p page) title() string {
	if p.screen == screenList {
		return p.list.title
	}
	return dex.DisplayName(p.detail.p.Name)
}

func (m tuiModel) currentPage() page { return page{m.screen, m.detail, m.list} }

func (m *tuiModel) pushPage() { m.history = append(m.history, m.currentPage()) }

// back returns to the previous page, or to the search screen.
func (m *tuiModel) back() tea.Cmd {
	if n := len(m.history); n > 0 {
		p := m.history[n-1]
		m.history = m.history[:n-1]
		m.screen, m.detail, m.list = p.screen, p.detail, p.list
		return nil
	}
	m.screen = screenSearch
	return nil
}

// openPokemon opens a Pokémon's entry (Overview tab) from a list or the Family tab,
// remembering the current page for the back key.
func (m *tuiModel) openPokemon(name string) tea.Cmd {
	if p, ok := m.pokeCache[name]; ok {
		m.pushPage()
		return m.openDetail(p)
	}
	m.openOnLoad = name
	return m.loadRelated(name)
}

// openList opens the Pokémon related to a move, item or ability. From the search screen
// it starts a new trail; from another page it is added to the history.
func (m *tuiModel) openList(kind, name string, fromSearch bool) tea.Cmd {
	var title, caption string
	var members []string
	loaded := false
	switch kind {
	case "move":
		if mv, ok := m.moves[name]; ok {
			loaded, title, members = true, mv.DisplayName, mv.LearnedBy
			caption = "Pokémon that learn " + title
		}
	case "item":
		if it, ok := m.items[name]; ok {
			loaded, title, members = true, it.DisplayName, it.HeldBy
			caption = "Wild Pokémon that may hold " + title
		}
	case "ability":
		if a, ok := m.abilities[name]; ok {
			loaded, title, members = true, a.DisplayName, a.Pokemon
			caption = "Pokémon that can have " + title
		}
	}
	if !loaded { // load it first; the loaded message reopens the list
		m.openListOnLoad, m.listFromSearch = kind+":"+name, fromSearch
		switch kind {
		case "move":
			return tea.Batch(m.spin(), m.fetchMove(name))
		case "item":
			return tea.Batch(m.spin(), m.fetchItem(name))
		default:
			return tea.Batch(m.spin(), m.fetchAbility(name))
		}
	}

	if fromSearch {
		m.history = nil
	} else {
		m.pushPage()
	}
	m.list = listState{kind: kind, name: name, title: title, caption: caption, members: m.sortByDex(members)}
	m.screen = screenList
	m.status = ""
	return m.previewList()
}

// reopenListIfWaiting opens the list that was waiting for kind/name to load.
func (m *tuiModel) reopenListIfWaiting(kind, name string) tea.Cmd {
	if m.openListOnLoad != kind+":"+name {
		return nil
	}
	m.openListOnLoad = ""
	return m.openList(kind, name, m.listFromSearch)
}

// previewList loads the selected Pokémon of the list (after a short pause).
func (m *tuiModel) previewList() tea.Cmd {
	l := m.list
	if len(l.members) == 0 {
		return nil
	}
	name := l.members[clamp(l.sel, 0, len(l.members)-1)]
	if p, ok := m.pokeCache[name]; ok {
		return m.fetchSprite(p)
	}
	m.listSeq++
	seq := m.listSeq
	return tea.Tick(m.previewDelay, func(time.Time) tea.Msg { return listPreviewMsg{seq, name} })
}

func (m tuiModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	k, e := key.String(), m.keys.Entry
	l := &m.list
	move := func(delta int) tea.Cmd {
		l.sel = clamp(l.sel+delta, 0, len(l.members)-1)
		return m.previewList()
	}
	switch {
	case e.Back.Matches(k):
		return m, m.back()
	case e.Open.Matches(k):
		if len(l.members) > 0 {
			return m, m.openPokemon(l.members[clamp(l.sel, 0, len(l.members)-1)])
		}
	case e.Up.Matches(k):
		return m, move(-1)
	case e.Down.Matches(k):
		return m, move(1)
	case e.PageUp.Matches(k):
		return m, move(-10)
	case e.PageDown.Matches(k):
		return m, move(10)
	case e.Top.Matches(k):
		return m, move(-1 << 20)
	case e.Bottom.Matches(k):
		return m, move(1 << 20)
	}
	return m, nil
}

func (m tuiModel) viewList() string {
	l := m.list
	totalW := m.width
	if totalW <= 0 {
		totalW = 100
	}
	totalW = min(totalW, 130)
	const leftW = 34
	rightW := max(30, min(maxRightWidth, totalW-leftW-9))

	e := m.keys.Entry
	help := helpLine(item(pair(e.Up, e.Down), "select"), item(e.Open.Key(0), "open"),
		item(pair(e.ScrollUp, e.ScrollDown), "scroll"), item(m.keys.Global.Help.Key(0), "help"), m.backLabel())
	foot := m.footer(help)
	avail := m.availHeight(foot)

	var left strings.Builder
	if len(l.members) == 0 {
		left.WriteString(labelStyle.Render("None."))
	}
	rows := max(5, avail)
	start := clamp(l.sel-rows/2, 0, max(0, len(l.members)-rows))
	for i := start; i < min(len(l.members), start+rows); i++ {
		left.WriteString(m.dexRow(l.members[i], i == l.sel, leftW) + "\n")
	}

	var right string
	if len(l.members) > 0 {
		name := l.members[clamp(l.sel, 0, len(l.members)-1)]
		if p, ok := m.pokeCache[name]; ok {
			right = renderPokemonCard(m.data.Chart(), p, rightW, m.spriteView(p.Name, 24, 12))
		} else {
			right = headingStyle.Render(dex.DisplayName(name)) + "\n\n" + m.spinner.View() + " Loading…"
		}
	}
	right = m.clip(right, avail, pair(e.ScrollUp, e.ScrollDown))

	caption := labelStyle.Render(fmt.Sprintf(" %s · %d", l.caption, len(l.members)))
	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(headingStyle.Render(l.title)), m.breadcrumb(totalW), caption, "",
		twoPanels(strings.TrimRight(left.String(), "\n"), leftW, right, rightW, 0), foot)
}

// ----------------------------------------------------------------------------- breadcrumb

// breadcrumb shows where you are ("Search › Charmeleon › Charizard") and, on the right,
// where the back key takes you.
func (m tuiModel) breadcrumb(width int) string {
	sep := labelStyle.Render(" › ")
	crumbs := []string{labelStyle.Render("Search")}
	for _, h := range m.history {
		crumbs = append(crumbs, labelStyle.Render(h.title()))
	}
	crumbs = append(crumbs, headingStyle.Render(m.currentPage().title()))
	left := " " + strings.Join(crumbs, sep)

	right := accentBlock.Render(" "+m.keys.Entry.Back.Key(0)+" ") + labelStyle.Render(" ← back to "+m.backTarget()+" ")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		return left + "   " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m tuiModel) backTarget() string {
	if n := len(m.history); n > 0 {
		return m.history[n-1].title()
	}
	return "search"
}

// backLabel names the page the back key returns to, for the help line.
func (m tuiModel) backLabel() string {
	return m.keys.Entry.Back.Key(0) + " back to " + m.backTarget()
}
