// Type Calc - calculate the weaknesses/resistances of a Pokémon or a type combination.
//
// Data comes from PokeAPI (https://pokeapi.co) and is cached locally.
//
// Usage:
//
//	type_calc                        interactive mode (TAB autocompletion)
//	type_calc pokemon <name>         weaknesses of a Pokémon
//	type_calc type <type1> [type2]   weaknesses of 1 or 2 types
//	type_calc types                  list available types
//	type_calc --refresh ...          ignore the cache and download the data again
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/peterh/liner"
	"golang.org/x/term"
)

// ----------------------------------------------------------------------------- colors

var useColor = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

var typeColors = map[string]int{
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

func fg256(text string, n int) string { return sgr(text, "38;5;"+strconv.Itoa(n)) }
func dim(s string) string             { return sgr(s, "2") }
func bold(s string) string            { return sgr(s, "1") }
func red(s string) string             { return sgr(s, "31") }
func yellow(s string) string          { return sgr(s, "33") }

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func badge(t string) string {
	color, ok := typeColors[t]
	if !ok {
		color = 250
	}
	return fg256(capitalize(t), color)
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

func resolveType(d *PokeData, query string, ask prompter) (string, error) {
	chart, err := d.Types()
	if err != nil {
		return "", err
	}
	q := normalize(query)
	if _, ok := chart[q]; ok {
		return q, nil
	}
	return choose(query, suggest(q, d.TypeNames(), 8), "type", ask), nil
}

func resolvePokemon(d *PokeData, query string, ask prompter) (string, error) {
	names, err := d.PokemonNames()
	if err != nil {
		return "", err
	}
	q := normalize(query)
	if d.pokemonSet[q] {
		return q, nil
	}
	return choose(query, suggest(q, names, 8), "Pokémon", ask), nil
}

// ----------------------------------------------------------------------------- calculation

func effectiveness(chart TypeChart, defTypes []string) map[string]float64 {
	result := map[string]float64{}
	for atk := range chart {
		m := 1.0
		for _, def := range defTypes {
			if v, ok := chart[def][atk]; ok {
				m *= v
			}
		}
		result[atk] = m
	}
	return result
}

var groups = []struct {
	mult  float64
	label string
	color string
	desc  string
}{
	{4, "4×", "1;31", "Very weak"},
	{2, "2×", "31", "Weak"},
	{0.5, "½×", "32", "Resists"},
	{0.25, "¼×", "1;32", "Strongly resists"},
	{0, "0×", "36", "Immune"},
}

// Abilities that change type matchups (shown as a warning only).
var abilityNotes = map[string]string{
	"levitate": "immune to Ground", "flash-fire": "immune to Fire", "water-absorb": "immune to Water",
	"storm-drain": "immune to Water", "dry-skin": "immune to Water, +25% Fire damage",
	"volt-absorb": "immune to Electric", "lightning-rod": "immune to Electric",
	"motor-drive": "immune to Electric", "sap-sipper": "immune to Grass",
	"thick-fat": "½ Fire and Ice damage", "heatproof": "½ Fire damage",
	"water-bubble": "½ Fire damage", "purifying-salt": "½ Ghost damage",
	"earth-eater": "immune to Ground", "well-baked-body": "immune to Fire",
	"wonder-guard": "only hit by super effective moves",
	"fluffy":       "½ contact damage, 2× Fire damage", "filter": "¾ super effective damage",
	"solid-rock": "¾ super effective damage", "prism-armor": "¾ super effective damage",
	"tera-shell": "moves are not very effective at full HP",
}

func printChart(chart TypeChart, defTypes []string) {
	eff := effectiveness(chart, defTypes)
	fmt.Println("Type(s): " + strings.Join(badges(defTypes), " / "))
	fmt.Println()
	for _, g := range groups {
		var attackers []string
		for atk, m := range eff {
			if m == g.mult {
				attackers = append(attackers, atk)
			}
		}
		if len(attackers) == 0 {
			continue
		}
		sort.Strings(attackers)
		label := fmt.Sprintf("%-5s%-17s", g.label, g.desc)
		fmt.Printf("  %s %s\n", sgr(label, g.color), strings.Join(badges(attackers), ", "))
	}
	fmt.Println()
}

func cmdPokemon(d *PokeData, query string, ask prompter) (bool, error) {
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
	title := strings.Join(strings.Split(p.Name, "-"), " ")
	words := strings.Fields(title)
	for i, w := range words {
		words[i] = capitalize(w)
	}
	fmt.Println(bold(fmt.Sprintf("#%d %s", p.ID, strings.Join(words, " "))))
	printChart(chart, p.Types)

	var notes []string
	for _, a := range p.Abilities {
		if n, ok := abilityNotes[a]; ok {
			notes = append(notes, fmt.Sprintf("%s (%s)", a, n))
		}
	}
	if len(notes) > 0 {
		fmt.Println(yellow("  Note – abilities that may change this: ") + strings.Join(notes, "; "))
		fmt.Println()
	}
	return true, nil
}

func cmdType(d *PokeData, queries []string, ask prompter) (bool, error) {
	if len(queries) < 1 || len(queries) > 2 {
		fmt.Println(red("Provide one or two types."))
		return false, nil
	}
	chart, err := d.Types()
	if err != nil {
		return false, err
	}
	var types []string
	for _, q := range queries {
		t, err := resolveType(d, q, ask)
		if err != nil || t == "" {
			return false, err
		}
		if len(types) == 0 || types[0] != t {
			types = append(types, t)
		}
	}
	fmt.Println()
	printChart(chart, types)
	return true, nil
}

func cmdTypes(d *PokeData) error {
	if _, err := d.Types(); err != nil {
		return err
	}
	fmt.Println(strings.Join(badges(d.TypeNames()), ", "))
	return nil
}

// ----------------------------------------------------------------------------- interactive mode

const helpText = `Commands:
  p <name>            weaknesses of a Pokémon    (e.g. p garchomp)
  t <type> [type2]    weaknesses of 1 or 2 types (e.g. t fire flying)
  types               list available types
  help                show this help
  quit                exit
TAB autocompletes Pokémon names and types.`

var commands = []string{"p", "pokemon", "t", "type", "types", "help", "quit"}

func completer(d *PokeData) liner.Completer {
	return func(line string) []string {
		parts := strings.Fields(line)
		trailingSpace := strings.HasSuffix(line, " ")
		var pool []string
		var prefix, word string
		switch {
		case len(parts) == 0 || (len(parts) == 1 && !trailingSpace):
			pool = commands
			if len(parts) == 1 {
				word = parts[0]
			}
		default:
			switch strings.ToLower(parts[0]) {
			case "p", "pokemon":
				pool = d.pokemonNames
			case "t", "type":
				pool = d.TypeNames()
			}
			if trailingSpace {
				prefix = line
			} else {
				word = parts[len(parts)-1]
				prefix = line[:len(line)-len(word)]
			}
		}
		word = strings.ToLower(word)
		var out []string
		for _, c := range pool {
			if strings.HasPrefix(c, word) {
				out = append(out, prefix+c)
			}
		}
		return out
	}
}

func interactive(d *PokeData) error {
	fmt.Println(bold("Type Calc – Pokémon weakness calculator"))
	fmt.Println(helpText)
	fmt.Println()
	// Preload data for autocompletion.
	if _, err := d.Types(); err != nil {
		return err
	}
	if _, err := d.PokemonNames(); err != nil {
		return err
	}

	var readLine func(prompt string) (string, error)
	if isTerminal(os.Stdin) {
		ln := liner.NewLiner()
		defer ln.Close()
		ln.SetCtrlCAborts(true)
		ln.SetTabCompletionStyle(liner.TabPrints)
		ln.SetCompleter(completer(d))
		readLine = func(prompt string) (string, error) {
			s, err := ln.Prompt(prompt)
			if err == nil && strings.TrimSpace(s) != "" {
				ln.AppendHistory(s)
			}
			return s, err
		}
	} else {
		sc := bufio.NewScanner(os.Stdin)
		readLine = func(prompt string) (string, error) {
			fmt.Print(prompt)
			if !sc.Scan() {
				return "", io.EOF
			}
			return sc.Text(), nil
		}
	}
	ask := prompter(readLine)

	for {
		line, err := readLine("> ")
		if err != nil {
			fmt.Println()
			return nil
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		cmd, args := strings.ToLower(parts[0]), parts[1:]
		switch {
		case cmd == "quit" || cmd == "exit" || cmd == "q":
			return nil
		case cmd == "help" || cmd == "h" || cmd == "?":
			fmt.Println(helpText)
		case cmd == "types":
			err = cmdTypes(d)
		case (cmd == "p" || cmd == "pokemon") && len(args) > 0:
			_, err = cmdPokemon(d, strings.Join(args, "-"), ask)
		case (cmd == "t" || cmd == "type") && len(args) > 0:
			_, err = cmdType(d, args, ask)
		default:
			fmt.Println("Invalid command. Type 'help'.")
		}
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
		}
	}
}

// ----------------------------------------------------------------------------- main

func usage() {
	fmt.Fprintf(os.Stderr, `Calculate Pokémon / type weaknesses (PokeAPI).

Usage:
  %[1]s [--refresh]                       interactive mode
  %[1]s [--refresh] pokemon|p <name>       weaknesses of a Pokémon
  %[1]s [--refresh] type|t <type> [type2]  weaknesses of 1 or 2 types
  %[1]s [--refresh] types                  list available types

Options:
`, os.Args[0])
	flag.PrintDefaults()
}

func main() {
	refresh := flag.Bool("refresh", false, "ignore the local cache")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	d := NewPokeData(*refresh)
	var ask prompter
	if isTerminal(os.Stdin) {
		reader := bufio.NewReader(os.Stdin)
		ask = func(prompt string) (string, error) {
			fmt.Print(prompt)
			return reader.ReadString('\n')
		}
	}

	ok := true
	var err error
	if len(args) == 0 {
		err = interactive(d)
	} else {
		switch args[0] {
		case "pokemon", "p":
			if len(args) < 2 {
				usage()
				os.Exit(2)
			}
			ok, err = cmdPokemon(d, strings.Join(args[1:], "-"), ask)
		case "type", "t":
			ok, err = cmdType(d, args[1:], ask)
		case "types":
			err = cmdTypes(d)
		case "help", "-h", "--help":
			usage()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command '%s'.\n\n", args[0])
			usage()
			os.Exit(2)
		}
	}
	if err != nil {
		if errors.Is(err, errNotFound) {
			err = errors.New("not found on PokeAPI")
		}
		fmt.Fprintln(os.Stderr, red("Error: "+err.Error()))
		os.Exit(1)
	}
	if !ok {
		os.Exit(1)
	}
}
