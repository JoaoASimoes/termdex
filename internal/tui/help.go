package tui

import (
	"strings"

	"github.com/JoaoASimoes/pokedex/internal/config"
)

// Help lines are built from the configured key bindings, so they always show the
// keys that actually work.

// pair shows two opposite bindings, e.g. "↑/↓ ctrl+k/ctrl+j".
func pair(a, b config.Binding) string {
	s := a.Key(0) + "/" + b.Key(0)
	if a.Key(1) != "" && b.Key(1) != "" {
		s += " " + a.Key(1) + "/" + b.Key(1)
	}
	return s
}

func item(keys, desc string) string {
	if strings.TrimSpace(keys) == "" || keys == "/" {
		return ""
	}
	return keys + " " + desc
}

func helpLine(items ...string) string {
	var out []string
	for _, it := range items {
		if it != "" {
			out = append(out, it)
		}
	}
	return strings.Join(out, " • ")
}

func (m tuiModel) searchHelp(mode int) string {
	s, g := m.keys.Search, m.keys.Global
	sel := item(pair(s.Up, s.Down), "select")
	tabs := item(pair(s.PrevTab, s.NextTab), "tabs")
	help := item(g.Help.Key(0), "help")
	clear := item(s.Clear.Key(0), "clear/quit")
	switch mode {
	case modePokedex:
		return helpLine(sel, item(s.Confirm.Key(0), "open"), "type:fire gen:1 filters", tabs, help, clear)
	case modeMoves:
		return helpLine(sel, item(s.Confirm.Key(0), "Pokémon that learn it"), tabs, help, clear)
	case modeItems:
		return helpLine(sel, item(s.Confirm.Key(0), "Pokémon that hold it"), tabs, help, clear)
	case modeNatures:
		return helpLine(sel, "type a name or stat (speed, -spa)", tabs, help, clear)
	default:
		return helpLine(sel, item(s.Confirm.Key(0), "confirm"), item(s.NextField.Key(0), "Type 1/Type 2"), tabs, help, clear)
	}
}

func (m tuiModel) entryHelp() string {
	e, g, f := m.keys.Entry, m.keys.Global, m.keys.Filter
	sel := item(pair(e.Up, e.Down), "select")
	tabs := item(pair(e.PrevTab, e.NextTab), "tabs")
	help := item(g.Help.Key(0), "help")
	switch m.detail.tab {
	case tabOverview:
		jump := helpLine(e.Overview.Key(0), e.Abilities.Key(0), e.Moves.Key(0), e.Family.Key(0))
		return helpLine(tabs, item(strings.ReplaceAll(jump, " • ", "/"), "jump to tab"),
			item(pair(e.ScrollUp, e.ScrollDown), "scroll"), help, m.backLabel())
	case tabAbilities:
		return helpLine(sel, item(e.Open.Key(0), "Pokémon with it"), tabs, help, m.backLabel())
	case tabMoves:
		if m.detail.filtering {
			return helpLine("type to filter", item(pair(f.Up, f.Down), "select"),
				item(f.Apply.Key(0), "keep filter"), item(f.Cancel.Key(0), "clear filter"))
		}
		return helpLine(sel, item(e.Open.Key(0), "Pokémon that learn it"), item(pair(e.PrevGame, e.NextGame), "game"),
			item(e.Filter.Key(0), "filter"), tabs, help, m.backLabel())
	default: // family
		return helpLine(sel, item(e.Open.Key(0), "open"), tabs, help, m.backLabel())
	}
}
