package tui

import (
	"fmt"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/fuzzy"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type detailTab int

const (
	tabOverview detailTab = iota
	tabAbilities
	tabMoves
	tabFamily
	numTabs
)

type detailState struct {
	p          dex.Pokemon
	tab        detailTab
	abilitySel int
	moveSel    int
	game       int // index into p.Learnsets
	filter     textinput.Model
	filtering  bool
	asked      map[int]bool // learnsets whose move details were requested
	familySel  int
	openOnLoad string // related Pokémon to open once loaded
}

func (d detailState) learnset() dex.Learnset {
	if d.game < 0 || d.game >= len(d.p.Learnsets) {
		return dex.Learnset{}
	}
	return d.p.Learnsets[d.game]
}

func (m *tuiModel) openDetail(p dex.Pokemon) tea.Cmd {
	f := textinput.New()
	f.Prompt = "/ "
	f.Placeholder = "filter moves (name, type, TM, egg…)"
	f.CharLimit = 30
	f.SetWidth(40)
	m.detail = detailState{p: p, filter: f, asked: map[int]bool{}, game: defaultGame(p)}
	m.screen = screenDetail
	m.status = ""

	cmds := []tea.Cmd{m.fetchSprite(p), m.fetchSpriteKey(femaleKey(p.Name), p.SpriteFemale)}
	for _, a := range p.Abilities {
		cmds = append(cmds, m.fetchAbility(a.Name))
	}
	return tea.Batch(cmds...)
}

// requestMoves fetches the details of every move in the selected game's pool
// (bounded by the client's request limiter).
func (m *tuiModel) requestMoves() tea.Cmd {
	d := &m.detail
	if d.asked[d.game] {
		return nil
	}
	d.asked[d.game] = true
	cmds := []tea.Cmd{m.spin()}
	for _, mv := range d.learnset().Moves {
		cmds = append(cmds, m.fetchMove(mv.Name))
	}
	return tea.Batch(cmds...)
}

// defaultGame picks the newest game with a level-up learnset (skipping spin-offs
// such as Pokémon Champions, whose moves are all "train").
func defaultGame(p dex.Pokemon) int {
	for i, ls := range p.Learnsets {
		for _, mv := range ls.Moves {
			if mv.Method == "level-up" {
				return i
			}
		}
	}
	return 0
}

func (m *tuiModel) setGame(delta int) tea.Cmd {
	d := &m.detail
	g := clamp(d.game+delta, 0, len(d.p.Learnsets)-1)
	if g == d.game {
		return nil
	}
	d.game = g
	d.moveSel = 0
	return m.requestMoves()
}

// movesLoaded counts how many moves of the current game have their details loaded.
func (m tuiModel) movesLoaded() (loaded, total int) {
	seen := map[string]bool{}
	for _, mv := range m.detail.learnset().Moves {
		if seen[mv.Name] {
			continue
		}
		seen[mv.Name] = true
		total++
		if _, ok := m.moves[mv.Name]; ok {
			loaded++
		}
	}
	return loaded, total
}

func (m tuiModel) filteredMoves() []dex.MoveRef {
	all := m.detail.learnset().Moves
	q := strings.ToLower(strings.TrimSpace(m.detail.filter.Value()))
	if q == "" {
		return all
	}
	qn := fuzzy.Normalize(q)
	var out []dex.MoveRef
	for _, mv := range all {
		hay := []string{mv.Name, strings.ToLower(dex.MethodLabel(mv)), mv.Method}
		if info, ok := m.moves[mv.Name]; ok {
			hay = append(hay, strings.ToLower(info.DisplayName), info.Type, info.Class)
		}
		for _, h := range hay {
			if strings.Contains(h, q) || strings.Contains(h, qn) {
				out = append(out, mv)
				break
			}
		}
	}
	return out
}

func (m *tuiModel) setTab(t detailTab) tea.Cmd {
	m.detail.tab = (t + numTabs) % numTabs
	switch m.detail.tab {
	case tabMoves:
		return m.requestMoves()
	case tabFamily:
		if m.detail.familySel == 0 {
			m.detail.familySel = m.currentFamilyIndex()
		}
		return m.previewFamily()
	}
	return nil
}

// moveSelection moves the cursor of the current list by delta (clamped).
func (m *tuiModel) moveSelection(delta int) tea.Cmd {
	d := &m.detail
	switch d.tab {
	case tabFamily:
		d.familySel = clamp(d.familySel+delta, 0, len(m.familyItems())-1)
		return m.previewFamily()
	case tabAbilities:
		d.abilitySel = clamp(d.abilitySel+delta, 0, len(d.p.Abilities)-1)
	case tabMoves:
		d.moveSel = clamp(d.moveSel+delta, 0, len(m.filteredMoves())-1)
	}
	return nil
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return max(lo, min(hi, v))
}

func (m tuiModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	d := &m.detail
	key, isKey := msg.(tea.KeyPressMsg)

	// While filtering, keys go to the filter input (except list navigation).
	if d.filtering {
		if isKey {
			switch key.String() {
			case "esc":
				d.filtering = false
				d.filter.Blur()
				d.filter.SetValue("")
				d.moveSel = 0
				return m, nil
			case "enter":
				d.filtering = false
				d.filter.Blur()
				return m, nil
			case "up", "ctrl+k", "ctrl+p":
				return m, m.moveSelection(-1)
			case "down", "ctrl+j", "ctrl+n":
				return m, m.moveSelection(1)
			case "ctrl+l":
				return m, m.setTab(d.tab + 1)
			case "ctrl+h":
				return m, m.setTab(d.tab - 1)
			}
		}
		before := d.filter.Value()
		var cmd tea.Cmd
		d.filter, cmd = d.filter.Update(msg)
		if d.filter.Value() != before {
			d.moveSel = 0
		}
		return m, cmd
	}

	if !isKey {
		return m, nil
	}
	switch key.String() {
	case "esc", "backspace", "q":
		if n := len(m.history); n > 0 { // back to the Pokémon we came from
			m.detail = m.history[n-1]
			m.history = m.history[:n-1]
			return m, nil
		}
		m.screen = screenSearch
		return m, nil
	case "enter":
		if d.tab == tabFamily {
			return m, m.openFamily()
		}
	case "tab", "right", "l", "ctrl+l":
		return m, m.setTab(d.tab + 1)
	case "shift+tab", "left", "h", "ctrl+h":
		return m, m.setTab(d.tab - 1)
	case "1":
		return m, m.setTab(tabOverview)
	case "2":
		return m, m.setTab(tabAbilities)
	case "3":
		return m, m.setTab(tabMoves)
	case "4", "f":
		return m, m.setTab(tabFamily)
	case "up", "k", "ctrl+k", "ctrl+p":
		return m, m.moveSelection(-1)
	case "down", "j", "ctrl+j", "ctrl+n":
		return m, m.moveSelection(1)
	case "pgup":
		return m, m.moveSelection(-10)
	case "pgdown":
		return m, m.moveSelection(10)
	case "home", "g":
		return m, m.moveSelection(-1 << 20)
	case "end", "G":
		return m, m.moveSelection(1 << 20)
	case "]", ">":
		if d.tab == tabMoves {
			return m, m.setGame(1)
		}
	case "[", "<":
		if d.tab == tabMoves {
			return m, m.setGame(-1)
		}
	case "/":
		if d.tab == tabMoves {
			d.filtering = true
			return m, d.filter.Focus()
		}
	}
	return m, nil
}

// ----------------------------------------------------------------------------- view

func (m tuiModel) viewDetail() string {
	d := m.detail
	p := d.p
	extra := headingStyle.Render(fmt.Sprintf("#%04d %s", p.ID, dex.DisplayName(p.Name))) + "  " +
		strings.Join(wrapBadges(p.Types, 40), " ")

	tabs := renderTabs([]string{"1 Overview", "2 Abilities", "3 Moves", "4 Family"}, int(d.tab))

	totalW := m.width
	if totalW <= 0 {
		totalW = 100
	}
	totalW = min(totalW, 130)
	bodyH := max(12, m.height-10)

	var body, help string
	switch d.tab {
	case tabOverview:
		body = m.viewOverview(totalW)
		help = "ctrl+h/l, ←/→ or 1-4 tabs • 4 family • ctrl+f search • ctrl+t types • " + m.backLabel()
	case tabAbilities:
		body = m.viewAbilities(totalW)
		help = "↑/↓ select ability • ctrl+h/l, ←/→ or 1-4 tabs • ctrl+f search • ctrl+t types • " + m.backLabel()
	case tabMoves:
		body = m.viewMoves(totalW, bodyH)
		help = "↑/↓ ctrl+k/j select • [/] game • / filter • ctrl+h/l tabs • ctrl+f search • " + m.backLabel()
		if d.filtering {
			help = "type to filter • ↑/↓ select • enter keep filter • esc clear filter"
		}
	case tabFamily:
		body = m.viewFamily(totalW, bodyH)
		help = "↑/↓ ctrl+k/j select • enter open • ctrl+h/l tabs • ctrl+f search • " + m.backLabel()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.header(extra), "", tabs, "", body, m.footer(help))
}

// twoPanels renders a left and right panel side by side with equal heights.
func twoPanels(left string, leftW int, right string, rightW int, height int) string {
	h := max(height, max(lipgloss.Height(left), lipgloss.Height(right)))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		panelFocusedStyle.Render(lipgloss.NewStyle().Width(leftW).Height(h).Render(left)),
		" ",
		panelStyle.Render(lipgloss.NewStyle().Width(rightW).Height(h).Render(right)))
}

func (m tuiModel) viewOverview(totalW int) string {
	p := m.detail.p
	leftW := max(36, (totalW-9)/2)
	rightW := max(36, totalW-9-leftW)

	leftParts := []string{renderIntro(p, leftW, m.spriteView(p.Name, 30, 15))}
	leftParts = append(leftParts, sectionStyle.Render("Base stats")+"\n"+renderStats(p, leftW))

	rightParts := []string{
		sectionStyle.Render("Weaknesses & resistances") + "\n" + renderMatchups(m.data.Chart(), p.Types, rightW, false),
		sectionStyle.Render("Abilities") + "\n" + wrap(renderAbilityLine(p), rightW),
		sectionStyle.Render("Evolution") + "\n" + renderEvoTree(p, rightW),
	}
	if w := renderAbilityWarnings(p, rightW); w != "" {
		rightParts = append(rightParts, w)
	}
	if len(p.Varieties) > 1 {
		var forms []string
		for _, v := range p.Varieties {
			if v != p.Name {
				forms = append(forms, dex.DisplayName(v))
			}
		}
		rightParts = append(rightParts, sectionStyle.Render("Other forms")+"\n"+wrap(strings.Join(forms, labelStyle.Render(" · ")), rightW))
	}
	rightParts = append(rightParts, labelStyle.Render(wrap(fmt.Sprintf(
		"Learns %d different moves across %d games – see the Moves tab. "+
			"Press 4 to browse and open evolutions and forms.", p.UniqueMoveCount(), len(p.Learnsets)), rightW)))

	return twoPanels(strings.Join(leftParts, "\n\n"), leftW, strings.Join(rightParts, "\n\n"), rightW, 0)
}

func (m tuiModel) viewAbilities(totalW int) string {
	d := m.detail
	const leftW = 28
	rightW := max(30, totalW-leftW-9)

	var list strings.Builder
	for i, a := range d.p.Abilities {
		name := dex.DisplayName(a.Name)
		if info, ok := m.abilities[a.Name]; ok {
			name = info.DisplayName
		}
		line := "  " + name
		if i == d.abilitySel {
			line = selectedStyle.Render("▸ " + name)
		}
		if a.Hidden {
			line += labelStyle.Render("  hidden")
		}
		list.WriteString(line + "\n")
	}

	var right string
	if len(d.p.Abilities) > 0 {
		ref := d.p.Abilities[d.abilitySel]
		info, ok := m.abilities[ref.Name]
		switch {
		case !ok && m.pending["ability:"+ref.Name]:
			right = m.spinner.View() + " Loading…"
		case !ok:
			right = labelStyle.Render("No data.")
		default:
			right = renderAbilityInfo(info, ref.Hidden, rightW)
		}
	}
	return twoPanels(strings.TrimRight(list.String(), "\n"), leftW, right, rightW, 8)
}

func renderAbilityInfo(a dex.AbilityInfo, hidden bool, width int) string {
	title := headingStyle.Render(a.DisplayName)
	var tags []string
	if hidden {
		tags = append(tags, warnStyle.Render("Hidden ability"))
	}
	if a.Generation != "" {
		tags = append(tags, labelStyle.Render("Introduced in "+dex.GenLabel(a.Generation)))
	}
	parts := []string{title + "\n" + strings.Join(tags, labelStyle.Render(" · "))}
	if a.ShortEffect != "" {
		parts = append(parts, sectionStyle.Render("In short")+"\n"+wrap(a.ShortEffect, width))
	}
	if a.Effect != "" && a.Effect != a.ShortEffect {
		parts = append(parts, sectionStyle.Render("Effect")+"\n"+wrap(a.Effect, width))
	}
	if a.Flavor != "" {
		parts = append(parts, flavorStyle.Render(wrap(a.Flavor, width)))
	}
	return strings.Join(parts, "\n\n")
}

func (m tuiModel) viewMoves(totalW, bodyH int) string {
	d := m.detail
	const leftW = 44
	rightW := max(30, totalW-leftW-9)
	moves := m.filteredMoves()
	sel := clamp(d.moveSel, 0, len(moves)-1)

	var list strings.Builder
	game := labelStyle.Render("Game ") + headingStyle.Render(dex.DisplayName(d.learnset().VersionGroup))
	if n := len(d.p.Learnsets); n > 1 {
		game += labelStyle.Render(fmt.Sprintf("  %d/%d  [ ] change", d.game+1, n))
	}
	list.WriteString(game + "\n")
	if d.filtering || d.filter.Value() != "" {
		list.WriteString(d.filter.View() + "\n")
	} else {
		list.WriteString(labelStyle.Render("press / to filter") + "\n")
	}
	if loaded, total := m.movesLoaded(); loaded < total {
		list.WriteString(m.spinner.View() + labelStyle.Render(fmt.Sprintf(" loading move data %d/%d", loaded, total)) + "\n")
	} else {
		list.WriteString(labelStyle.Render(fmt.Sprintf("%d of %d entries", len(moves), len(d.learnset().Moves))) + "\n")
	}

	rows := max(5, bodyH-3)
	start := clamp(sel-rows/2, 0, max(0, len(moves)-rows))
	end := min(len(moves), start+rows)
	for i := start; i < end; i++ {
		list.WriteString(m.moveRow(moves[i], i == sel) + "\n")
	}
	if len(moves) == 0 {
		list.WriteString(labelStyle.Render("No moves match.") + "\n")
	}

	var right string
	if len(moves) > 0 {
		ref := moves[sel]
		if info, ok := m.moves[ref.Name]; ok {
			right = renderMoveInfo(info, ref, rightW)
		} else {
			right = headingStyle.Render(dex.DisplayName(ref.Name)) + "\n\n" + m.spinner.View() + " Loading…"
		}
	}
	return twoPanels(strings.TrimRight(list.String(), "\n"), leftW, right, rightW, bodyH)
}

func (m tuiModel) moveRow(ref dex.MoveRef, selected bool) string {
	name := dex.DisplayName(ref.Name)
	typ, class := strings.Repeat(" ", 10), "    "
	if info, ok := m.moves[ref.Name]; ok {
		name = info.DisplayName
		typ = typeBadgeFixed(info.Type, 10)
		class = classLabel(info.Class, true)
	}
	method := fmt.Sprintf("%-6s", dex.MethodLabel(ref))
	name = fmt.Sprintf("%-19s", truncate(name, 19))
	prefix := "  "
	if selected {
		prefix = selectedStyle.Render("▸ ")
		name = selectedStyle.Render(name)
	}
	return prefix + labelStyle.Render(method) + name + " " + typ + " " + class
}

func renderMoveInfo(mv dex.MoveInfo, ref dex.MoveRef, width int) string {
	title := headingStyle.Render(mv.DisplayName)
	badges := typeBadge(mv.Type) + "  " + classLabel(mv.Class, false)

	num := func(v *int, suffix string) string {
		if v == nil {
			return "—"
		}
		return fmt.Sprintf("%d%s", *v, suffix)
	}
	cell := func(label, value string) string {
		return labelStyle.Render(label+" ") + headingStyle.Render(value)
	}
	cells := []string{
		cell("Power", num(mv.Power, "")),
		cell("Accuracy", num(mv.Accuracy, "%")),
		cell("PP", num(mv.PP, "")),
		cell("Priority", fmt.Sprintf("%+d", mv.Priority)),
	}
	if mv.EffectChance != nil {
		cells = append(cells, cell("Effect chance", num(mv.EffectChance, "%")))
	}
	stats := strings.Join(cells, "   ")

	learn := dex.MethodLabel(ref)
	switch ref.Method {
	case "level-up":
		learn = fmt.Sprintf("Level %d", max(1, ref.Level))
	case "machine":
		learn = "TM / TR"
	case "egg":
		learn = "Egg move"
	case "tutor":
		learn = "Move tutor"
	}
	learned := labelStyle.Render("Learned by ") + learn
	if mv.Generation != "" {
		learned += labelStyle.Render(" · introduced in " + dex.GenLabel(mv.Generation))
	}

	parts := []string{title + "\n" + badges, wrap(stats, width), wrap(learned, width)}
	short := dex.WithChance(mv.ShortEffect, mv.EffectChance)
	long := dex.WithChance(mv.Effect, mv.EffectChance)
	if short != "" {
		parts = append(parts, sectionStyle.Render("In short")+"\n"+wrap(short, width))
	}
	if long != "" && long != short {
		parts = append(parts, sectionStyle.Render("Effect")+"\n"+wrap(long, width))
	}
	if mv.Flavor != "" {
		parts = append(parts, flavorStyle.Render(wrap(mv.Flavor, width)))
	}
	return strings.Join(parts, "\n\n")
}
