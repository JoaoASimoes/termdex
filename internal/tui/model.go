// Package tui is the interactive Bubble Tea interface of the Pokédex.
package tui

import (
	"errors"
	"fmt"
	"image"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/pokedex/internal/config"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/sprite"
)

// The TUI has three screens:
//   - search: the Pokédex, Moves, Items, Natures and Type Matchups tabs;
//   - detail: a Pokémon's full entry (Overview, Abilities, Moves, Family tabs);
//   - list: the Pokémon related to a move, item or ability.
// The help screen can be shown on top of any of them.

type screen int

const (
	screenSearch screen = iota
	screenDetail
	screenList
)

const (
	leftWidth      = 30 // inner width of the search panel
	minRightWidth  = 44
	maxRightWidth  = 64
	maxSuggestions = 10
)

// ----------------------------------------------------------------------------- messages

type dataLoadedMsg struct{ err error }

type pokemonLoadedMsg struct {
	seq  int
	name string
	p    dex.Pokemon
	err  error
}

// listPreviewMsg fires after a short pause on a Pokémon of a list page.
type listPreviewMsg struct {
	seq  int
	name string
}

// filterLoadedMsg carries the members of a Pokédex filter ("type:fire", "gen:1").
type filterLoadedMsg struct {
	key   string
	names []string
	err   error
}

type previewMsg struct {
	seq  int
	name string
}

type abilityLoadedMsg struct {
	name string
	info dex.AbilityInfo
	err  error
}

// relatedLoadedMsg carries a Pokémon loaded for the Family tab.
type relatedLoadedMsg struct {
	name string
	p    dex.Pokemon
	err  error
}

type spriteLoadedMsg struct {
	name string
	img  image.Image
	err  error
}

type naturesLoadedMsg struct {
	natures []dex.Nature
	err     error
}

// catalogPreviewMsg fires after a short pause on a highlighted move/item not loaded yet.
type catalogPreviewMsg struct {
	seq  int
	kind field // fieldMove or fieldItem
	name string
}

// catalogLoadedMsg carries the full list of move or item names.
type catalogLoadedMsg struct {
	kind  field
	names []string
	err   error
}

type itemLoadedMsg struct {
	name string
	info dex.ItemInfo
	err  error
}

type moveLoadedMsg struct {
	name string
	info dex.MoveInfo
	err  error
}

// ----------------------------------------------------------------------------- model

type tuiModel struct {
	data      *dex.Client
	dataReady bool
	screen    screen

	spinner spinner.Model
	loading string // non-empty while the search screen waits for something
	status  string // last error

	search  searchState
	detail  detailState
	list    listState
	history []page // pages to return to with the back key

	openOnLoad     string // Pokémon to open (from a list/Family) once it is loaded
	openListOnLoad string // "move:<name>" etc. to open as a list once loaded
	listFromSearch bool
	listSeq        int // bumped on every list preview; stale ones are ignored

	showHelp bool
	scroll   int      // scroll offset of the panels (see scroll.go)
	ui       *uiState // written by View

	// retry re-runs whatever failed last (network errors); nil when nothing failed.
	retry       func(m *tuiModel) tea.Cmd
	failedMoves map[string]bool

	// Pokédex order and filters (see dexorder.go).
	dexOrder   []string
	dexRank    map[string]int
	dexNum     map[string]int
	filterSets map[string]map[string]bool

	// Session caches (the disk cache sits behind these).
	pokeCache  map[string]dex.Pokemon
	abilities  map[string]dex.AbilityInfo
	moves      map[string]dex.MoveInfo
	natures    []dex.Nature
	moveNames  []string // every move (Moves tab)
	itemNames  []string // every item (Items tab)
	items      map[string]dex.ItemInfo
	catSeq     int                    // bumped on every move/item preview; stale previews are ignored
	spriteImgs map[string]image.Image // nil entry = no sprite available
	spriteStr  map[string]string      // rendered sprites, keyed by "name@colsxrows"
	pending    map[string]bool        // "ability:<name>" / "move:<name>" requests in flight
	seq        int                    // bumped on every Pokémon preview; stale loads are ignored

	keys         config.Keys
	previewDelay time.Duration // debounces network lookups while scrolling
	sprites      bool

	width, height int
}

func newTUIModel(d *dex.Client, cfg config.Config) tuiModel {
	applyTheme(cfg.Colors)
	return tuiModel{
		data:         d,
		keys:         cfg.Keys,
		previewDelay: cfg.General.PreviewDelay(),
		sprites:      cfg.General.Sprites,
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(accent))),
		loading:      "Loading data from PokeAPI…",
		search:       newSearchState(),
		pokeCache:    map[string]dex.Pokemon{},
		abilities:    map[string]dex.AbilityInfo{},
		moves:        map[string]dex.MoveInfo{},
		items:        map[string]dex.ItemInfo{},
		spriteImgs:   map[string]image.Image{},
		spriteStr:    map[string]string{},
		pending:      map[string]bool{},
		failedMoves:  map[string]bool{},
		filterSets:   map[string]map[string]bool{},
		ui:           &uiState{},
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadDataCmd(m.data), textinput.Blink)
}

// ----------------------------------------------------------------------------- commands

func loadDataCmd(d *dex.Client) tea.Cmd {
	return func() tea.Msg {
		_, err := d.Types()
		if err == nil {
			_, err = d.PokemonNames()
		}
		return dataLoadedMsg{err}
	}
}

func loadPokemonCmd(d *dex.Client, seq int, name string) tea.Cmd {
	return func() tea.Msg {
		p, err := d.Pokemon(name)
		return pokemonLoadedMsg{seq, name, p, err}
	}
}

func (m *tuiModel) spin() tea.Cmd { return m.spinner.Tick }

// fail reports a failed download. Network errors get a friendly "offline" message; in
// every case the retry key re-runs retry.
func (m *tuiModel) fail(what string, err error, retry func(m *tuiModel) tea.Cmd) {
	m.loading = ""
	if errors.Is(err, dex.ErrNetwork) {
		m.status = "Can't reach PokeAPI – you seem to be offline. Everything already cached still works."
	} else {
		m.status = "Could not load " + what + ": " + err.Error()
	}
	m.retry = retry
}

func (m *tuiModel) fetchAbility(name string) tea.Cmd {
	key := "ability:" + name
	if _, ok := m.abilities[name]; ok || m.pending[key] {
		return nil
	}
	m.pending[key] = true
	d := m.data
	return tea.Batch(m.spin(), func() tea.Msg {
		a, err := d.Ability(name)
		return abilityLoadedMsg{name, a, err}
	})
}

func (m *tuiModel) fetchSprite(p dex.Pokemon) tea.Cmd {
	return m.fetchSpriteKey(p.Name, p.Sprite)
}

// fetchSpriteKey loads the sprite at url, stored under key (a Pokémon name, or a
// variant such as its female look).
func (m *tuiModel) fetchSpriteKey(key, url string) tea.Cmd {
	pk := "sprite:" + key
	if _, ok := m.spriteImgs[key]; ok || m.pending[pk] || url == "" || !m.sprites {
		return nil
	}
	m.pending[pk] = true
	d := m.data
	return func() tea.Msg {
		img, err := d.Sprite(key, url)
		return spriteLoadedMsg{key, img, err}
	}
}

// spriteView returns the rendered sprite of name (cached), or "" if not available yet.
func (m tuiModel) spriteView(name string, cols, rows int) string {
	return m.spriteViewKey(name, cols, rows)
}

func (m tuiModel) spriteViewKey(name string, cols, rows int) string {
	img := m.spriteImgs[name]
	if img == nil {
		return ""
	}
	key := fmt.Sprintf("%s@%dx%d", name, cols, rows)
	if s, ok := m.spriteStr[key]; ok {
		return s
	}
	s := sprite.Render(img, cols, rows)
	m.spriteStr[key] = s
	return s
}

func (m *tuiModel) fetchNatures() tea.Cmd {
	if m.natures != nil || m.pending["natures"] {
		return nil
	}
	m.pending["natures"] = true
	d := m.data
	return tea.Batch(m.spin(), func() tea.Msg {
		n, err := d.Natures()
		return naturesLoadedMsg{n, err}
	})
}

// itemSpriteKey is the sprite cache key of an item (kept apart from Pokémon names).
func itemSpriteKey(name string) string { return "item~" + name }

// fetchCatalog loads the list of every move (kind fieldMove) or item (fieldItem).
func (m *tuiModel) fetchCatalog(kind field) tea.Cmd {
	key := "catalog:moves"
	loaded := m.moveNames != nil
	if kind == fieldItem {
		key, loaded = "catalog:items", m.itemNames != nil
	}
	if loaded || m.pending[key] {
		return nil
	}
	m.pending[key] = true
	d := m.data
	return tea.Batch(m.spin(), func() tea.Msg {
		var names []string
		var err error
		if kind == fieldItem {
			names, err = d.ItemNames()
		} else {
			names, err = d.MoveNames()
		}
		return catalogLoadedMsg{kind, names, err}
	})
}

func (m *tuiModel) fetchItem(name string) tea.Cmd {
	key := "item:" + name
	if _, ok := m.items[name]; ok || m.pending[key] {
		return nil
	}
	m.pending[key] = true
	d := m.data
	return func() tea.Msg {
		it, err := d.Item(name)
		return itemLoadedMsg{name, it, err}
	}
}

func (m *tuiModel) fetchMove(name string) tea.Cmd {
	key := "move:" + name
	if _, ok := m.moves[name]; ok || m.pending[key] {
		return nil
	}
	m.pending[key] = true
	d := m.data
	return func() tea.Msg {
		mv, err := d.Move(name)
		return moveLoadedMsg{name, mv, err}
	}
}

// ----------------------------------------------------------------------------- update

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case dataLoadedMsg:
		m.loading = ""
		if msg.err != nil {
			m.fail("data", msg.err, func(m *tuiModel) tea.Cmd {
				m.loading = "Loading data from PokeAPI…"
				return tea.Batch(m.spin(), loadDataCmd(m.data))
			})
			return m, nil
		}
		m.dataReady = true
		m.buildDexOrder()
		m.refreshSuggestions()
		return m, m.preview()

	case listPreviewMsg:
		if msg.seq != m.listSeq {
			return m, nil
		}
		return m, m.loadRelated(msg.name)

	case filterLoadedMsg:
		delete(m.pending, "filter:"+msg.key)
		if msg.err != nil {
			key := msg.key
			m.fail("the "+key+" filter", msg.err, func(m *tuiModel) tea.Cmd { return m.fetchFilter(key) })
			return m, nil
		}
		set := make(map[string]bool, len(msg.names))
		for _, n := range msg.names {
			set[n] = true
		}
		m.filterSets[msg.key] = set
		if m.search.focus == fieldPokemon {
			m.refreshSuggestions()
			return m, m.preview()
		}
		return m, nil

	case previewMsg:
		if msg.seq != m.seq {
			return m, nil
		}
		m.loading = "Loading " + dex.DisplayName(msg.name) + "…"
		return m, tea.Batch(m.spin(), loadPokemonCmd(m.data, msg.seq, msg.name))

	case pokemonLoadedMsg:
		if msg.err == nil {
			m.pokeCache[msg.p.Name] = msg.p
		}
		if msg.seq != m.seq {
			return m, nil // the user has moved on
		}
		m.loading = ""
		if msg.err != nil {
			name, open := msg.name, m.search.openOnLoad == msg.name
			m.search.openOnLoad = ""
			m.fail(dex.DisplayName(name), msg.err, func(m *tuiModel) tea.Cmd {
				if open {
					m.search.openOnLoad = name
				}
				return m.showPokemon(name, false)
			})
			return m, nil
		}
		p := msg.p
		m.search.preview = &p
		if m.search.openOnLoad == p.Name {
			m.search.openOnLoad = ""
			return m, m.openDetail(p)
		}
		return m, m.fetchSprite(p)

	case relatedLoadedMsg:
		delete(m.pending, "pokemon:"+msg.name)
		if msg.err != nil {
			name := msg.name
			m.fail(dex.DisplayName(name), msg.err, func(m *tuiModel) tea.Cmd { return m.loadRelated(name) })
			return m, nil
		}
		m.pokeCache[msg.name] = msg.p
		if m.openOnLoad == msg.name {
			m.openOnLoad = ""
			m.pushPage()
			return m, m.openDetail(msg.p)
		}
		return m, m.fetchSprite(msg.p)

	case spriteLoadedMsg:
		delete(m.pending, "sprite:"+msg.name)
		if errors.Is(msg.err, dex.ErrNetwork) {
			return m, nil // not remembered: tried again next time it is shown
		}
		m.spriteImgs[msg.name] = msg.img // nil on other errors: shown without a sprite
		return m, nil

	case abilityLoadedMsg:
		delete(m.pending, "ability:"+msg.name)
		if msg.err != nil {
			name := msg.name
			m.fail("the ability", msg.err, func(m *tuiModel) tea.Cmd { return m.fetchAbility(name) })
			return m, nil
		}
		m.abilities[msg.name] = msg.info
		return m, m.reopenListIfWaiting("ability", msg.name)

	case naturesLoadedMsg:
		delete(m.pending, "natures")
		if msg.err != nil {
			m.fail("natures", msg.err, func(m *tuiModel) tea.Cmd { return m.fetchNatures() })
			return m, nil
		}
		m.natures = msg.natures
		if m.search.focus == fieldNature {
			m.refreshSuggestions()
			return m, m.preview()
		}
		return m, nil

	case moveLoadedMsg:
		delete(m.pending, "move:"+msg.name)
		if msg.err != nil {
			m.failedMoves[msg.name] = true
			m.fail("moves", msg.err, func(m *tuiModel) tea.Cmd {
				var cmds []tea.Cmd
				for name := range m.failedMoves {
					cmds = append(cmds, m.fetchMove(name))
				}
				m.failedMoves = map[string]bool{}
				return tea.Batch(append(cmds, m.spin())...)
			})
			return m, nil
		}
		delete(m.failedMoves, msg.name)
		m.moves[msg.name] = msg.info
		return m, m.reopenListIfWaiting("move", msg.name)

	case catalogLoadedMsg:
		if msg.kind == fieldItem {
			delete(m.pending, "catalog:items")
		} else {
			delete(m.pending, "catalog:moves")
		}
		if msg.err != nil {
			kind := msg.kind
			m.fail("the list", msg.err, func(m *tuiModel) tea.Cmd { return m.fetchCatalog(kind) })
			return m, nil
		}
		if msg.kind == fieldItem {
			m.itemNames = msg.names
		} else {
			m.moveNames = msg.names
		}
		if m.search.focus == msg.kind {
			m.refreshSuggestions()
			return m, m.preview()
		}
		return m, nil

	case catalogPreviewMsg:
		if msg.seq != m.catSeq {
			return m, nil // the user has moved on
		}
		if msg.kind == fieldItem {
			return m, tea.Batch(m.spin(), m.fetchItem(msg.name))
		}
		return m, tea.Batch(m.spin(), m.fetchMove(msg.name))

	case itemLoadedMsg:
		delete(m.pending, "item:"+msg.name)
		if msg.err != nil {
			name := msg.name
			m.fail("the item", msg.err, func(m *tuiModel) tea.Cmd { return m.fetchItem(name) })
			return m, nil
		}
		m.items[msg.name] = msg.info
		return m, tea.Batch(m.fetchSpriteKey(itemSpriteKey(msg.name), msg.info.Sprite),
			m.reopenListIfWaiting("item", msg.name))

	case spinner.TickMsg:
		if m.loading == "" && len(m.pending) == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		k, g := msg.String(), m.keys.Global

		// Scrolling the panels; any other key starts again from the top.
		up, down := m.keys.Search.ScrollUp, m.keys.Search.ScrollDown
		if m.screen != screenSearch && !m.showHelp {
			up, down = m.keys.Entry.ScrollUp, m.keys.Entry.ScrollDown
		}
		if m.showHelp {
			up = append(append(config.Binding{}, up...), "up", "k")
			down = append(append(config.Binding{}, down...), "down", "j")
		}
		switch {
		case up.Matches(k):
			m.scrollBy(-scrollStep)
			return m, nil
		case down.Matches(k):
			m.scrollBy(scrollStep)
			return m, nil
		}
		m.scroll = 0

		// The help screen takes every other key.
		if m.showHelp {
			if g.Help.Matches(k) || k == "esc" || k == "q" {
				m.showHelp = false
			} else if g.Quit.Matches(k) {
				return m, tea.Quit
			}
			return m, nil
		}

		// Global shortcuts, available on every screen.
		switch {
		case g.Quit.Matches(k):
			return m, tea.Quit
		case g.Help.Matches(k): // also while typing: no Pokémon, move or item has a "?"
			m.showHelp = true
			return m, nil
		case g.Retry.Matches(k):
			if m.retry != nil {
				retry := m.retry
				m.retry, m.status = nil, ""
				return m, retry(&m)
			}
			return m, nil
		case g.Search.Matches(k): // new Pokémon search
			m.screen = screenSearch
			m.search.inputs[fieldPokemon].SetValue("")
			return m, m.setFocus(fieldPokemon)
		case g.TypeMatchups.Matches(k):
			m.screen = screenSearch
			return m, m.setFocus(fieldType1)
		}
	}

	switch m.screen {
	case screenDetail:
		return m.updateDetail(msg)
	case screenList:
		return m.updateList(msg)
	}
	return m.updateSearch(msg)
}

// ----------------------------------------------------------------------------- view

func (m tuiModel) View() tea.View {
	if m.ui != nil {
		m.ui.maxScroll = 0 // set again by clip while drawing
	}
	var content string
	switch {
	case m.showHelp:
		content = m.viewHelp()
	case m.screen == screenDetail:
		content = m.viewDetail()
	case m.screen == screenList:
		content = m.viewList()
	default:
		content = m.viewSearch()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "Pokédex"
	return v
}

func (m tuiModel) header(extra string) string {
	h := titleStyle.Render("◓ Pokédex")
	if extra != "" {
		h += "  " + extra
	}
	return h
}

func (m tuiModel) footer(help string) string {
	var status string
	switch {
	case m.loading != "":
		status = m.spinner.View() + " " + m.loading
	case m.status != "":
		status = errStyle.Render(m.status)
		if m.retry != nil {
			status += labelStyle.Render(" · ") + selectedStyle.Render(m.keys.Global.Retry.Key(0)) + labelStyle.Render(" retry")
		}
	}
	h := helpStyle.Render(help)
	if m.width > 0 && lipgloss.Width(help) > m.width {
		h = helpStyle.Width(m.width).Render(help)
	}
	return status + "\n" + h
}

// Run starts the interactive interface and blocks until the user quits.
func Run(d *dex.Client, cfg config.Config) error {
	_, err := tea.NewProgram(newTUIModel(d, cfg)).Run()
	return err
}
