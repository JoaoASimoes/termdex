// Package cli parses the command line and runs either a subcommand or the TUI.
package cli

import (
	"bufio"
	"fmt"
	"github.com/JoaoASimoes/pokedex/internal/dex"
	"github.com/JoaoASimoes/pokedex/internal/fuzzy"
	"github.com/JoaoASimoes/pokedex/internal/sprite"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// ----------------------------------------------------------------------------- colors

var useColor = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

var typeColors256 = map[string]int{
	"normal": 250, "fire": 202, "water": 33, "grass": 70, "electric": 220, "ice": 117,
	"fighting": 160, "poison": 129, "ground": 179, "flying": 147, "psychic": 205,
	"bug": 106, "rock": 137, "ghost": 97, "dragon": 63, "dark": 94, "steel": 145, "fairy": 218,
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func sgr(text, code string) string {
	if !useColor {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func dim(s string) string    { return sgr(s, "2") }
func bold(s string) string   { return sgr(s, "1") }
func red(s string) string    { return sgr(s, "31") }
func yellow(s string) string { return sgr(s, "33") }

func badge(t string) string {
	color, ok := typeColors256[t]
	if !ok {
		color = 250
	}
	return sgr(dex.Capitalize(t), "38;5;"+strconv.Itoa(color))
}

func badges(ts []string) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = badge(t)
	}
	return out
}

// ----------------------------------------------------------------------------- prompting

// prompter reads the user's answer when several suggestions are offered.
// It is nil when input is not interactive.
type prompter func(prompt string) (string, error)

func stdinPrompter() prompter {
	if !isTerminal(os.Stdin) {
		return nil
	}
	reader := bufio.NewReader(os.Stdin)
	return func(prompt string) (string, error) {
		fmt.Print(prompt)
		return reader.ReadString('\n')
	}
}

func choose(query string, options []string, what string, ask prompter) string {
	if len(options) == 0 {
		fmt.Println(red(fmt.Sprintf("No %s found for '%s'.", what, query)))
		return ""
	}
	if len(options) == 1 {
		fmt.Println(yellow(fmt.Sprintf("'%s' not found; using '%s'.", query, options[0])))
		return options[0]
	}
	fmt.Println(yellow(fmt.Sprintf("'%s' not found. Did you mean:", query)))
	for i, o := range options {
		fmt.Printf("  %d. %s\n", i+1, o)
	}
	if ask == nil {
		return ""
	}
	ans, err := ask("Number (Enter to cancel): ")
	if err != nil {
		return ""
	}
	if i, err := strconv.Atoi(strings.TrimSpace(ans)); err == nil && i >= 1 && i <= len(options) {
		return options[i-1]
	}
	return ""
}

func resolveType(d *dex.Client, query string, ask prompter) (string, error) {
	chart, err := d.Types()
	if err != nil {
		return "", err
	}
	q := fuzzy.Normalize(query)
	if _, ok := chart[q]; ok {
		return q, nil
	}
	return choose(query, fuzzy.Suggest(q, d.TypeNames(), 8), "type", ask), nil
}

func resolvePokemon(d *dex.Client, query string, ask prompter) (string, error) {
	names, err := d.PokemonNames()
	if err != nil {
		return "", err
	}
	q := fuzzy.Normalize(query)
	if d.IsPokemon(q) {
		return q, nil
	}
	return choose(query, fuzzy.Suggest(q, names, 8), "Pokémon", ask), nil
}

// ----------------------------------------------------------------------------- commands

func printChart(chart dex.TypeChart, defTypes []string) {
	fmt.Println("Type(s): " + strings.Join(badges(defTypes), " / "))
	fmt.Println()
	for _, g := range dex.GroupedEffectiveness(chart, defTypes) {
		label := fmt.Sprintf("%-5s%-17s", g.Label, g.Desc)
		fmt.Printf("  %s %s\n", sgr(label, g.ANSI), strings.Join(badges(g.Attackers), ", "))
	}
	fmt.Println()
}

func cmdPokemon(d *dex.Client, query string, ask prompter) (bool, error) {
	name, err := resolvePokemon(d, query, ask)
	if err != nil || name == "" {
		return false, err
	}
	p, err := d.Pokemon(name)
	if err != nil {
		return false, err
	}
	chart, err := d.Types()
	if err != nil {
		return false, err
	}
	fmt.Println()
	if useColor { // sprites need a colour terminal
		if img, err := d.Sprite(p.Name, p.Sprite); err == nil {
			lipgloss.Println(sprite.Render(img, 32, 16))
		}
	}
	fmt.Println(bold(fmt.Sprintf("#%d %s", p.ID, dex.DisplayName(p.Name))))
	printChart(chart, p.Types)
	if notes := dex.AbilityWarnings(p.AbilityNames()); len(notes) > 0 {
		fmt.Println(yellow("  Note – abilities that may change this: ") + strings.Join(notes, "; "))
		fmt.Println()
	}
	return true, nil
}

func cmdType(d *dex.Client, queries []string, ask prompter) (bool, error) {
	if len(queries) < 1 || len(queries) > 2 {
		fmt.Println(red("Provide one or two types."))
		return false, nil
	}
	chart, err := d.Types()
	if err != nil {
		return false, err
	}
	var resolved []string
	for _, q := range queries {
		t, err := resolveType(d, q, ask)
		if err != nil || t == "" {
			return false, err
		}
		resolved = append(resolved, t)
	}
	fmt.Println()
	printChart(chart, dex.UniqueTypes(resolved...))
	return true, nil
}

func cmdTypes(d *dex.Client) error {
	if _, err := d.Types(); err != nil {
		return err
	}
	fmt.Println(strings.Join(badges(d.TypeNames()), ", "))
	return nil
}
