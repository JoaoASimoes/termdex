package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/pokedex/internal/config"
)

// The help screen lists every key of the current screen, built from the configuration,
// plus a few tips. It is opened with the help key and closed with it, esc or q.

func (m tuiModel) helpScreenName() string {
	switch m.screen {
	case screenDetail:
		return "entry"
	case screenList:
		return "list"
	}
	return "search"
}

var helpTips = map[string][]string{
	"search": {
		"Typos are fine: suggestions update as you type, and the highlighted one is shown on the right.",
		"Pokédex filters: type:fire, gen:1 (or gen:i), combined with each other and a name, e.g. \"type:dragon gen:4\" or \"type:fire type:flying char\".",
		"In Moves and Items, enter lists the Pokémon that learn the move / hold the item.",
	},
	"entry": {
		"The line under the title shows where you are; the back key returns there step by step.",
		"Enter on a move (Moves tab) or an ability (Abilities tab) lists the Pokémon that learn / have it.",
		"Enter on an evolution or form (Family tab) opens its entry.",
	},
	"list": {
		"Enter opens the selected Pokémon; the back key returns to this list.",
	},
}

func (m tuiModel) viewHelp() string {
	name := m.helpScreenName()
	keyW := 30
	var b strings.Builder
	for i, sec := range m.keys.HelpFor(name) {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(sectionStyle.Render(sec.Title) + "\n")
		for _, r := range sec.Rows {
			b.WriteString(lipgloss.NewStyle().Width(keyW).Render(keysLabel(r.Keys)) + r.Desc + "\n")
		}
	}
	if tips := helpTips[name]; len(tips) > 0 {
		b.WriteString("\n" + sectionStyle.Render("Tips") + "\n")
		width := min(max(40, m.width-8), 110)
		for _, t := range tips {
			b.WriteString(wrap("• "+t, width) + "\n")
		}
	}
	b.WriteString("\n" + labelStyle.Render("Every key can be changed in the config file: run `pokedex config path` to find it, see docs/keybindings.md."))

	g := m.keys.Global
	foot := m.footer(helpLine(item(g.Help.Key(0)+"/esc", "close"),
		item(pair(m.keys.Search.ScrollUp, m.keys.Search.ScrollDown), "scroll")))
	body := m.clip(strings.TrimRight(b.String(), "\n"), m.availHeight(foot), "↑/↓")
	return lipgloss.JoinVertical(lipgloss.Left, m.header(headingStyle.Render("Keys")), "",
		panelFocusedStyle.Render(body), foot)
}

// keysLabel lists every key of a binding: "↑, ctrl+k, ctrl+p".
func keysLabel(b config.Binding) string {
	if len(b) == 0 {
		return labelStyle.Render("(unbound)")
	}
	var keys []string
	for i := range b {
		keys = append(keys, b.Key(i))
	}
	return accentText(strings.Join(keys, ", "))
}

func accentText(s string) string { return selectedStyle.Render(s) }
