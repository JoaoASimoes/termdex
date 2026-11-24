// Package tui is the interactive Bubble Tea interface of the Pokédex.
package tui

import (
	"fmt"
	"github.com/JoaoASimoes/pokedex/internal/config"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/sprite"
	"image"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The TUI has two screens:
//   - search: the Pokédex search (with a live entry preview) and the type-matchup search;
//   - detail: a Pokémon's full entry, with Overview / Abilities / Moves tabs.

type screen int

const (
	screenSearch screen = iota
	screenDetail
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
	seq int
	p   dex.Pokemon
	err error
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
	history []detailState // Pokémon entries to return to with Esc

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
		return pokemonLoadedMsg{seq, p, err}
	}
}

func (m *tuiModel) spin() tea.Cmd { return m.spinner.Tick }

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
			m.status = "Could not load data: " + msg.err.Error()
			return m, nil
		}
		m.dataReady = true
		m.refreshSuggestions()
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
			m.status = "Could not load Pokémon: " + msg.err.Error()
			m.search.openOnLoad = ""
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
			m.status = "Could not load Pokémon: " + msg.err.Error()
			return m, nil
		}
		m.pokeCache[msg.name] = msg.p
		if m.screen == screenDetail && m.detail.openOnLoad == msg.name {
			return m, m.openRelated(msg.p)
		}
		return m, m.fetchSprite(msg.p)

	case spriteLoadedMsg:
		delete(m.pending, "sprite:"+msg.name)
		m.spriteImgs[msg.name] = msg.img // nil on error: shown without a sprite
		return m, nil

	case abilityLoadedMsg:
		delete(m.pending, "ability:"+msg.name)
		if msg.err != nil {
			m.status = "Could not load ability: " + msg.err.Error()
		} else {
			m.abilities[msg.name] = msg.info
		}
		return m, nil

	case naturesLoadedMsg:
		delete(m.pending, "natures")
		if msg.err != nil {
			m.status = "Could not load natures: " + msg.err.Error()
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
		if msg.err == nil {
			m.moves[msg.name] = msg.info
		}
		return m, nil

	case catalogLoadedMsg:
		if msg.kind == fieldItem {
			delete(m.pending, "catalog:items")
		} else {
			delete(m.pending, "catalog:moves")
		}
		if msg.err != nil {
			m.status = "Could not load the list: " + msg.err.Error()
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
			m.status = "Could not load item: " + msg.err.Error()
			return m, nil
		}
		m.items[msg.name] = msg.info
		return m, m.fetchSpriteKey(itemSpriteKey(msg.name), msg.info.Sprite)

	case spinner.TickMsg:
		if m.loading == "" && len(m.pending) == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// Global shortcuts, available on every screen.
		k, g := msg.String(), m.keys.Global
		switch {
		case g.Quit.Matches(k):
			return m, tea.Quit
		case g.Search.Matches(k): // new Pokémon search
			m.screen = screenSearch
			m.search.inputs[fieldPokemon].SetValue("")
			return m, m.setFocus(fieldPokemon)
		case g.TypeMatchups.Matches(k):
			m.screen = screenSearch
			return m, m.setFocus(fieldType1)
		}
	}

	if m.screen == screenDetail {
		return m.updateDetail(msg)
	}
	return m.updateSearch(msg)
}

// ----------------------------------------------------------------------------- view

func (m tuiModel) View() tea.View {
	var content string
	if m.screen == screenDetail {
		content = m.viewDetail()
	} else {
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
