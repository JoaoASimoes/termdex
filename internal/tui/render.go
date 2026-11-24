package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/JoaoASimoes/pokedex/internal/config"
	"github.com/JoaoASimoes/pokedex/internal/dex"
)

// ----------------------------------------------------------------------------- styles

// The styles below are built by applyTheme from the configured colours.
var (
	accent    color.Color
	accentAlt color.Color
	subtle    color.Color

	titleStyle, subtitleStyle, labelStyle, labelFocused, selectedStyle, helpStyle lipgloss.Style
	errStyle, warnStyle, headingStyle, sectionStyle, flavorStyle                  lipgloss.Style
	panelStyle, panelFocusedStyle, tabActive, tabInactive, accentBlock            lipgloss.Style

	typeBg = map[string]string{} // type -> badge background
)

func init() { applyTheme(config.Default().Colors) }

// applyTheme (re)builds every style from the configured colours.
func applyTheme(c config.Colors) {
	accent = lipgloss.Color(c.Accent)
	accentAlt = lipgloss.Color(c.Heading)
	subtle = lipgloss.Color(c.Muted)
	onAccent := lipgloss.Color(c.AccentText)

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent).Padding(0, 1)
	accentBlock = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent)
	subtitleStyle = lipgloss.NewStyle().Foreground(subtle)
	labelStyle = lipgloss.NewStyle().Foreground(subtle)
	labelFocused = lipgloss.NewStyle().Foreground(accent).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	helpStyle = lipgloss.NewStyle().Foreground(subtle)
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(c.Error))
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(c.Warning))
	headingStyle = lipgloss.NewStyle().Bold(true)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(accentAlt)
	flavorStyle = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color(c.Flavor))

	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(subtle).Padding(0, 1)
	panelFocusedStyle = panelStyle.BorderForeground(accent)

	tabActive = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent).Padding(0, 2)
	tabInactive = lipgloss.NewStyle().Foreground(subtle).Padding(0, 2)

	typeBg = map[string]string{}
	for t, col := range c.Types {
		typeBg[t] = col
	}
}

// Default light backgrounds that need dark text (used for ANSI-number colours, whose
// brightness is unknown).
var darkText = map[string]bool{
	"normal": true, "electric": true, "grass": true, "ice": true, "ground": true,
	"bug": true, "rock": true, "steel": true, "fairy": true, "fire": true,
}

// lightBackground reports whether text on bg should be dark.
func lightBackground(t, bg string) bool {
	if len(bg) == 7 && bg[0] == '#' {
		var r, g, b int
		if _, err := fmt.Sscanf(bg[1:], "%02x%02x%02x", &r, &g, &b); err == nil {
			return r*299+g*587+b*114 > 140*1000 // perceived brightness
		}
	}
	return darkText[t]
}

func typeStyle(t string) lipgloss.Style {
	bg, ok := typeBg[t]
	if !ok {
		bg = "#777777"
	}
	fg := "#FFFFFF"
	if lightBackground(t, bg) {
		fg = "#1C1C1C"
	}
	return lipgloss.NewStyle().Background(lipgloss.Color(bg)).Foreground(lipgloss.Color(fg)).Bold(true)
}

func typeBadge(t string) string {
	return typeStyle(t).Padding(0, 1).Render(dex.Capitalize(t))
}

// typeBadgeFixed renders a centered badge of a fixed width, for aligned columns.
func typeBadgeFixed(t string, width int) string {
	return typeStyle(t).Width(width).Align(lipgloss.Center).Render(dex.Capitalize(t))
}

var classStyles = map[string]lipgloss.Style{
	"physical": lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C5A")),
	"special":  lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")),
	"status":   lipgloss.NewStyle().Foreground(lipgloss.Color("#A8A8A8")),
}

func classLabel(class string, short bool) string {
	label := dex.Capitalize(class)
	if short && len([]rune(label)) > 4 {
		label = string([]rune(label)[:4])
	}
	if st, ok := classStyles[class]; ok {
		return st.Render(label)
	}
	return label
}

// wrapBadges lays out type badges in lines no wider than width.
func wrapBadges(types []string, width int) []string {
	var lines []string
	line, lineW := "", 0
	for _, t := range types {
		b := typeBadge(t)
		w := lipgloss.Width(b)
		if lineW > 0 && lineW+1+w > width {
			lines = append(lines, line)
			line, lineW = "", 0
		}
		if lineW > 0 {
			line += " "
			lineW++
		}
		line += b
		lineW += w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func renderTabs(labels []string, active int) string {
	var parts []string
	for i, l := range labels {
		if i == active {
			parts = append(parts, tabActive.Render(l))
		} else {
			parts = append(parts, tabInactive.Render(l))
		}
	}
	return strings.Join(parts, " ")
}

func wrap(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

// ----------------------------------------------------------------------------- Pokémon card

func statColor(v int) string {
	switch {
	case v < 50:
		return "#F34444"
	case v < 80:
		return "#FF7F0F"
	case v < 100:
		return "#FFDD57"
	case v < 120:
		return "#A0E515"
	case v < 150:
		return "#23CD5E"
	default:
		return "#00C2B8"
	}
}

func renderStats(p dex.Pokemon, width int) string {
	const labelW, valueW = 9, 5
	barW := max(10, min(36, width-labelW-valueW))
	var b strings.Builder
	for _, s := range p.Stats {
		label := dex.StatLabels[s.Name]
		if label == "" {
			label = dex.DisplayName(s.Name)
		}
		filled := min(barW, s.Base*barW/200)
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(statColor(s.Base))).Render(strings.Repeat("█", filled)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3A")).Render(strings.Repeat("░", barW-filled))
		fmt.Fprintf(&b, "%s%s%s\n", labelStyle.Render(fmt.Sprintf("%-*s", labelW, label)), fmt.Sprintf("%*d ", valueW-1, s.Base), bar)
	}
	fmt.Fprintf(&b, "%s%s", labelStyle.Render(fmt.Sprintf("%-*s", labelW, "Total")), headingStyle.Render(fmt.Sprintf("%*d", valueW-1, p.StatTotal())))
	return b.String()
}

func renderHeader(p dex.Pokemon) string {
	title := headingStyle.Render(fmt.Sprintf("#%04d %s", p.Number(), dex.DisplayName(p.Name)))
	var tags []string
	if p.Legendary {
		tags = append(tags, warnStyle.Render("Legendary"))
	}
	if p.Mythical {
		tags = append(tags, warnStyle.Render("Mythical"))
	}
	if p.Generation != "" {
		tags = append(tags, labelStyle.Render(dex.GenLabel(p.Generation)))
	}
	if len(tags) > 0 {
		title += "  " + strings.Join(tags, labelStyle.Render(" · "))
	}
	lines := []string{title}
	if p.Genus != "" {
		lines = append(lines, labelStyle.Render(p.Genus))
	}
	badges := strings.Join(wrapBadges(p.Types, 40), " ")
	size := labelStyle.Render(fmt.Sprintf("   %.1f m · %.1f kg", float64(p.Height)/10, float64(p.Weight)/10))
	lines = append(lines, badges+size)
	return strings.Join(lines, "\n")
}

func renderAbilityLine(p dex.Pokemon) string {
	var parts []string
	for _, a := range p.Abilities {
		s := dex.DisplayName(a.Name)
		if a.Hidden {
			s += labelStyle.Render(" (hidden)")
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, labelStyle.Render(" · "))
}

// renderMatchups renders the weakness groups for defTypes.
func renderMatchups(chart dex.TypeChart, defTypes []string, width int, compact bool) string {
	const labelW = 22
	var b strings.Builder
	for _, g := range dex.GroupedEffectiveness(chart, defTypes) {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color(g.Color)).Bold(true).Width(labelW).
			Render(fmt.Sprintf("%-4s%s", g.Label, g.Desc))
		pad := strings.Repeat(" ", labelW)
		for i, line := range wrapBadges(g.Attackers, width-labelW) {
			if i == 0 {
				b.WriteString(label + line + "\n")
			} else {
				b.WriteString(pad + line + "\n")
			}
		}
		if !compact {
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderAbilityWarnings(p dex.Pokemon, width int) string {
	notes := dex.AbilityWarnings(p.AbilityNames())
	if len(notes) == 0 {
		return ""
	}
	return wrap(warnStyle.Render("⚠ May change matchups: ")+strings.Join(notes, "; "), width)
}

// renderIntro is the top of an entry: the sprite on the left, and the header, description
// and any extra sections (rendered for the given text width) beside it.
func renderIntro(p dex.Pokemon, width int, sprite string, extra ...func(width int) string) string {
	textW := width
	if sprite != "" {
		textW = max(20, width-lipgloss.Width(sprite)-2)
	}
	parts := []string{renderHeader(p)}
	if p.Flavor != "" {
		parts = append(parts, flavorStyle.Render(wrap(p.Flavor, textW)))
	}
	for _, e := range extra {
		parts = append(parts, e(textW))
	}
	text := strings.Join(parts, "\n\n")
	if sprite == "" {
		return text
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sprite, "  ", lipgloss.NewStyle().Width(textW).Render(text))
}

// renderPokemonCard is the Pokédex entry shown next to the search list.
func renderPokemonCard(chart dex.TypeChart, p dex.Pokemon, width int, sprite string) string {
	parts := []string{renderIntro(p, width, sprite,
		func(w int) string { return sectionStyle.Render("Abilities") + "\n" + wrap(renderAbilityLine(p), w) },
		func(w int) string { return sectionStyle.Render("Evolution") + "\n" + renderEvoSummary(p, w) },
	)}
	parts = append(parts,
		sectionStyle.Render("Base stats")+"\n"+renderStats(p, width),
		sectionStyle.Render("Weaknesses & resistances")+"\n"+renderMatchups(chart, p.Types, width, true),
	)
	if w := renderAbilityWarnings(p, width); w != "" {
		parts = append(parts, w)
	}
	return strings.Join(parts, "\n\n")
}

// ----------------------------------------------------------------------------- evolution

// renderEvoSummary says what p evolves from and into, and how.
func renderEvoSummary(p dex.Pokemon, width int) string {
	node, parent := dex.FindEvo(p.Evolution, p.Species)
	if node == nil {
		return labelStyle.Render("No evolution data.")
	}
	if parent == nil && len(node.Next) == 0 {
		return labelStyle.Render("Does not evolve.")
	}
	var lines []string
	if parent != nil {
		lines = append(lines, labelStyle.Render("Evolves from ")+headingStyle.Render(dex.DisplayName(parent.Species))+
			labelStyle.Render(" — ")+dex.HowText(node.How()))
	}
	for _, c := range node.Next {
		lines = append(lines, labelStyle.Render("Evolves into ")+headingStyle.Render(dex.DisplayName(c.Species))+
			labelStyle.Render(" — ")+dex.HowText(c.How()))
	}
	if len(node.Next) == 0 {
		lines = append(lines, labelStyle.Render("Final stage: does not evolve further."))
	}
	return wrap(strings.Join(lines, "\n"), width)
}

// renderEvoTree draws the whole evolution chain, highlighting the current species.
func renderEvoTree(p dex.Pokemon, width int) string {
	root := p.Evolution
	if root == nil {
		return labelStyle.Render("No evolution data.")
	}
	if len(root.Next) == 0 {
		return labelStyle.Render("Does not evolve.")
	}
	name := func(n *dex.EvoNode) string {
		s := dex.DisplayName(n.Species)
		if n.Species == p.Species {
			return selectedStyle.Render("● " + s)
		}
		return s
	}
	lines := []string{name(root)}
	var walk func(n *dex.EvoNode, prefix string)
	walk = func(n *dex.EvoNode, prefix string) {
		for i := range n.Next {
			c := &n.Next[i]
			branch, cont := "├─ ", "│  "
			if i == len(n.Next)-1 {
				branch, cont = "└─ ", "   "
			}
			lines = append(lines, labelStyle.Render(prefix+branch)+name(c)+labelStyle.Render("  "+dex.HowText(c.How())))
			walk(c, prefix+cont)
		}
	}
	walk(root, "")
	return wrap(strings.Join(lines, "\n"), width)
}

// ----------------------------------------------------------------------------- natures

var (
	plusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD75F")).Bold(true)
	minusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true)
)

// natureHint is the short "+Atk −SpA" label next to a nature in lists.
func natureHint(n dex.Nature) string {
	if n.Neutral() {
		return labelStyle.Render("neutral")
	}
	return plusStyle.Render("+"+dex.ShortStat[n.Increased]) + " " + minusStyle.Render("−"+dex.ShortStat[n.Decreased])
}

const naturesAbout = "A Pokémon's nature raises one stat by 10% and lowers another by 10%. " +
	"HP is never affected. The five natures on the diagonal (Hardy, Docile, Bashful, Quirky, " +
	"Serious) are neutral and change nothing. A nature also decides which berry flavor the " +
	"Pokémon likes (linked to the raised stat) and dislikes (linked to the lowered stat). " +
	"Since Sword & Shield, a Mint changes the stat effect of a nature without changing the nature itself."

func renderNaturePanel(natures []dex.Nature, selected string, width int) string {
	if natures == nil {
		return helpStyle.Render(wrap(naturesAbout, width))
	}
	var parts []string
	if n, ok := dex.NatureByName(natures, selected); ok {
		head := headingStyle.Render(dex.Capitalize(n.Name))
		if n.Neutral() {
			parts = append(parts, head+"\n"+labelStyle.Render("Neutral: no stat changes, no flavor preference."))
		} else {
			effect := plusStyle.Render("▲ "+dex.StatLabels[n.Increased]+" +10%") + "    " +
				minusStyle.Render("▼ "+dex.StatLabels[n.Decreased]+" −10%")
			flavor := labelStyle.Render("Likes ") + dex.Capitalize(n.Likes) + labelStyle.Render(" food · dislikes ") + dex.Capitalize(n.Hates) + labelStyle.Render(" food")
			parts = append(parts, head+"\n"+effect+"\n"+flavor)
		}
	}
	parts = append(parts, sectionStyle.Render("Nature chart")+labelStyle.Render("  (row raises, column lowers)")+"\n"+renderNatureGrid(natures, selected))
	parts = append(parts, sectionStyle.Render("About natures")+"\n"+wrap(naturesAbout, width))
	return strings.Join(parts, "\n\n")
}

func renderNatureGrid(natures []dex.Nature, selected string) string {
	const cellW = 9
	cell := func(s string) string { return fmt.Sprintf("%-*s", cellW, s) }
	var b strings.Builder
	b.WriteString(cell(""))
	for _, dec := range dex.NatureStats {
		b.WriteString(minusStyle.Render(cell("−" + dex.ShortStat[dec])))
	}
	b.WriteString("\n")
	for _, inc := range dex.NatureStats {
		b.WriteString(plusStyle.Render(cell("+" + dex.ShortStat[inc])))
		for _, dec := range dex.NatureStats {
			n, ok := dex.NatureAt(natures, inc, dec)
			name := ""
			if ok {
				name = dex.Capitalize(n.Name)
			}
			switch {
			case ok && n.Name == selected:
				b.WriteString(accentBlock.Render(fmt.Sprintf("%-*s", cellW-1, name)) + " ")
			case inc == dec:
				b.WriteString(labelStyle.Render(cell(name)))
			default:
				b.WriteString(cell(name))
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ----------------------------------------------------------------------------- items

var categoryLabels = map[string]string{
	"held-items": "Held item", "choice": "Held item (Choice)", "bad-held-items": "Held item (harmful)",
	"healing": "Medicine", "status-cures": "Medicine", "revival": "Medicine", "pp-recovery": "Medicine",
	"vitamins": "Vitamin", "evolution": "Evolution item", "standard-balls": "Poké Ball",
	"special-balls": "Poké Ball", "apricorn-balls": "Poké Ball", "all-machines": "TM / HM / TR",
	"mega-stones": "Mega Stone", "z-crystals": "Z-Crystal", "plates": "Plate", "memories": "Memory",
	"type-enhancement": "Type booster", "medicine": "Medicine",
}

func itemCategory(c string) string {
	if l, ok := categoryLabels[c]; ok {
		return l
	}
	return dex.DisplayName(c)
}

func renderItemInfo(it dex.ItemInfo, width int, sprite string) string {
	facts := func(w int) string {
		lines := []string{labelStyle.Render("Category ") + itemCategory(it.Category)}
		if it.FlingPower != nil {
			lines = append(lines, labelStyle.Render("Fling power ")+fmt.Sprint(*it.FlingPower))
		}
		if len(it.Attributes) > 0 {
			var attrs []string
			for _, a := range it.Attributes {
				attrs = append(attrs, dex.DisplayName(a))
			}
			lines = append(lines, wrap(labelStyle.Render("Attributes ")+strings.Join(attrs, ", "), w))
		}
		return headingStyle.Render(it.DisplayName) + "\n\n" + strings.Join(lines, "\n")
	}

	head := facts(width)
	if sprite != "" {
		textW := max(20, width-lipgloss.Width(sprite)-2)
		head = lipgloss.JoinHorizontal(lipgloss.Top, sprite, "  ", lipgloss.NewStyle().Width(textW).Render(facts(textW)))
	}

	parts := []string{head}
	if it.ShortEffect != "" {
		parts = append(parts, sectionStyle.Render("In short")+"\n"+wrap(it.ShortEffect, width))
	}
	if it.Effect != "" && it.Effect != it.ShortEffect {
		parts = append(parts, sectionStyle.Render("Effect")+"\n"+wrap(it.Effect, width))
	}
	if it.Flavor != "" {
		parts = append(parts, flavorStyle.Render(wrap(it.Flavor, width)))
	}
	if len(it.HeldBy) > 0 {
		parts = append(parts, sectionStyle.Render("Held by wild Pokémon")+"\n"+wrap(nameList(it.HeldBy, 30), width))
	}
	return strings.Join(parts, "\n\n")
}
