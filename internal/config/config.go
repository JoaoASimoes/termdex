// Package config loads the user's configuration (general options and key bindings)
// from a TOML file, on top of the defaults embedded in default.toml.
package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultTOML is the commented default configuration; `pokedex config init` writes it.
//
//go:embed default.toml
var DefaultTOML []byte

// EnvPath names the environment variable that overrides the config file location.
const EnvPath = "POKEDEX_CONFIG"

type Config struct {
	General General `toml:"general"`
	Colors  Colors  `toml:"colors"`
	Keys    Keys    `toml:"keys"`
}

// Colors is the TUI theme. Each value is "#RRGGBB" or an ANSI colour number "0"–"255".
type Colors struct {
	Accent     string            `toml:"accent"`      // title, active tab, selection, focused border
	AccentText string            `toml:"accent_text"` // text drawn on the accent colour
	Heading    string            `toml:"heading"`     // section titles
	Muted      string            `toml:"muted"`       // labels, help line, borders
	Error      string            `toml:"error"`
	Warning    string            `toml:"warning"`
	Flavor     string            `toml:"flavor"` // Pokédex and move descriptions
	Types      map[string]string `toml:"types"`  // type badge backgrounds
}

// TypeNames are the 18 types whose badge colour can be set in [colors.types].
var TypeNames = []string{
	"normal", "fire", "water", "electric", "grass", "ice", "fighting", "poison", "ground",
	"flying", "psychic", "bug", "rock", "ghost", "dragon", "dark", "steel", "fairy",
}

// ValidColor reports whether c is "#RRGGBB" or an ANSI colour number 0–255.
func ValidColor(c string) bool {
	if len(c) == 7 && c[0] == '#' {
		for _, r := range c[1:] {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return false
			}
		}
		return true
	}
	n, err := strconv.Atoi(c)
	return err == nil && n >= 0 && n <= 255 && c == strconv.Itoa(n)
}

type General struct {
	CacheTTLDays   int  `toml:"cache_ttl_days"`
	MaxRequests    int  `toml:"max_requests"`
	PreviewDelayMS int  `toml:"preview_delay_ms"`
	Sprites        bool `toml:"sprites"`
}

func (g General) CacheTTL() time.Duration { return time.Duration(g.CacheTTLDays) * 24 * time.Hour }
func (g General) PreviewDelay() time.Duration {
	return time.Duration(g.PreviewDelayMS) * time.Millisecond
}

// Binding is the list of keys that trigger an action, as Bubble Tea names them
// ("ctrl+k", "enter", "G", …).
type Binding []string

// Matches reports whether key (a tea.KeyPressMsg.String()) triggers the binding.
func (b Binding) Matches(key string) bool {
	for _, k := range b {
		if k == key {
			return true
		}
	}
	return false
}

// Help is a short label for help lines: the first one or two keys, e.g. "↑/ctrl+k".
func (b Binding) Help() string {
	var out []string
	for _, k := range b {
		if len(out) == 2 {
			break
		}
		out = append(out, prettyKey(k))
	}
	if len(out) == 0 {
		return "(unbound)"
	}
	return strings.Join(out, "/")
}

// Key returns the i-th key in a readable form ("↑" for "up"), or "" if there is none.
func (b Binding) Key(i int) string {
	if i < 0 || i >= len(b) {
		return ""
	}
	return prettyKey(b[i])
}

var arrows = map[string]string{"up": "↑", "down": "↓", "left": "←", "right": "→"}

func prettyKey(k string) string {
	if a, ok := arrows[k]; ok {
		return a
	}
	return k
}

type Keys struct {
	Global GlobalKeys `toml:"global"`
	Search SearchKeys `toml:"search"`
	Entry  EntryKeys  `toml:"entry"`
	Filter FilterKeys `toml:"filter"`
}

type GlobalKeys struct {
	Quit         Binding `toml:"quit"`
	Search       Binding `toml:"search"`
	TypeMatchups Binding `toml:"type_matchups"`
	Help         Binding `toml:"help"`
	Retry        Binding `toml:"retry"`
}

type SearchKeys struct {
	Up         Binding `toml:"up"`
	Down       Binding `toml:"down"`
	Confirm    Binding `toml:"confirm"`
	Clear      Binding `toml:"clear"`
	NextField  Binding `toml:"next_field"`
	PrevField  Binding `toml:"prev_field"`
	NextTab    Binding `toml:"next_tab"`
	PrevTab    Binding `toml:"prev_tab"`
	ScrollUp   Binding `toml:"scroll_up"`
	ScrollDown Binding `toml:"scroll_down"`
}

type EntryKeys struct {
	Up         Binding `toml:"up"`
	Down       Binding `toml:"down"`
	PageUp     Binding `toml:"page_up"`
	PageDown   Binding `toml:"page_down"`
	Top        Binding `toml:"top"`
	Bottom     Binding `toml:"bottom"`
	NextTab    Binding `toml:"next_tab"`
	PrevTab    Binding `toml:"prev_tab"`
	Overview   Binding `toml:"overview"`
	Abilities  Binding `toml:"abilities"`
	Moves      Binding `toml:"moves"`
	Family     Binding `toml:"family"`
	NextGame   Binding `toml:"next_game"`
	PrevGame   Binding `toml:"prev_game"`
	Filter     Binding `toml:"filter"`
	Open       Binding `toml:"open"`
	Back       Binding `toml:"back"`
	ScrollUp   Binding `toml:"scroll_up"`
	ScrollDown Binding `toml:"scroll_down"`
}

type FilterKeys struct {
	Up      Binding `toml:"up"`
	Down    Binding `toml:"down"`
	Apply   Binding `toml:"apply"`
	Cancel  Binding `toml:"cancel"`
	NextTab Binding `toml:"next_tab"`
	PrevTab Binding `toml:"prev_tab"`
}

// Default returns the built-in configuration (parsed from default.toml).
func Default() Config {
	var c Config
	if _, err := toml.NewDecoder(bytes.NewReader(DefaultTOML)).Decode(&c); err != nil {
		panic("config: invalid embedded default.toml: " + err.Error())
	}
	return c
}

// DefaultPath is where the config file is looked up when no path is given.
func DefaultPath() string {
	if p := os.Getenv(EnvPath); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(dir, "pokedex", "config.toml")
}

// Load reads path on top of the defaults. A missing file is not an error: the
// defaults are returned. Unknown settings and conflicting keys are reported.
func Load(path string) (Config, error) {
	c := Default()
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	md, err := toml.NewDecoder(bytes.NewReader(raw)).Decode(&c)
	if err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if und := md.Undecoded(); len(und) > 0 {
		var names []string
		for _, k := range und {
			names = append(names, k.String())
		}
		return c, fmt.Errorf("%s: unknown setting(s): %s", path, strings.Join(names, ", "))
	}
	if err := c.Validate(); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Validate checks value ranges and that no key triggers two actions on the same screen.
func (c Config) Validate() error {
	var problems []string
	g := c.General
	if g.CacheTTLDays < 1 {
		problems = append(problems, "general.cache_ttl_days must be 1 or more")
	}
	if g.MaxRequests < 1 || g.MaxRequests > 20 {
		problems = append(problems, "general.max_requests must be between 1 and 20")
	}
	if g.PreviewDelayMS < 0 || g.PreviewDelayMS > 5000 {
		problems = append(problems, "general.preview_delay_ms must be between 0 and 5000")
	}
	col := c.Colors
	for _, f := range []struct{ name, value string }{
		{"accent", col.Accent}, {"accent_text", col.AccentText}, {"heading", col.Heading},
		{"muted", col.Muted}, {"error", col.Error}, {"warning", col.Warning}, {"flavor", col.Flavor},
	} {
		if !ValidColor(f.value) {
			problems = append(problems, fmt.Sprintf("colors.%s = %q is not a colour (use \"#RRGGBB\" or \"0\"-\"255\")", f.name, f.value))
		}
	}
	known := map[string]bool{}
	for _, t := range TypeNames {
		known[t] = true
	}
	var typeKeys []string
	for t := range col.Types {
		typeKeys = append(typeKeys, t)
	}
	sort.Strings(typeKeys)
	for _, t := range typeKeys {
		switch {
		case !known[t]:
			problems = append(problems, fmt.Sprintf("colors.types.%s: unknown type", t))
		case !ValidColor(col.Types[t]):
			problems = append(problems, fmt.Sprintf("colors.types.%s = %q is not a colour", t, col.Types[t]))
		}
	}

	// Global keys are active on every screen, so they are checked with each one.
	for _, screen := range []struct {
		name    string
		section any
	}{{"search", c.Keys.Search}, {"entry", c.Keys.Entry}, {"filter", c.Keys.Filter}} {
		problems = append(problems, conflicts(
			bindingsOf("global", c.Keys.Global), bindingsOf(screen.name, screen.section))...)
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// bindingsOf maps "section.action" to its binding, using the TOML names.
func bindingsOf(section string, keys any) map[string]Binding {
	out := map[string]Binding{}
	v := reflect.ValueOf(keys)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		out[section+"."+t.Field(i).Tag.Get("toml")] = v.Field(i).Interface().(Binding)
	}
	return out
}

func conflicts(sets ...map[string]Binding) []string {
	owner := map[string]string{}
	var problems []string
	var actions []string
	all := map[string]Binding{}
	for _, s := range sets {
		for a, b := range s {
			all[a] = b
			actions = append(actions, a)
		}
	}
	sort.Strings(actions)
	for _, a := range actions {
		for _, k := range all[a] {
			if prev, ok := owner[k]; ok && prev != a {
				problems = append(problems, fmt.Sprintf("key %q is bound to both keys.%s and keys.%s", k, prev, a))
				continue
			}
			owner[k] = a
		}
	}
	return problems
}

// Write writes the commented default configuration to path, creating its directory.
// It refuses to overwrite an existing file unless force is set.
func Write(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, DefaultTOML, 0o644)
}

// TOML renders the effective configuration.
func (c Config) TOML() string {
	var b bytes.Buffer
	_ = toml.NewEncoder(&b).Encode(c)
	return b.String()
}

// ----------------------------------------------------------------------------- help screen

// What each action does, keyed "section.action"; shown by the in-app help screen.
var actionDesc = map[string]string{
	"global.quit":          "quit",
	"global.search":        "new Pokémon search",
	"global.type_matchups": "go to Type Matchups",
	"global.help":          "show / hide this help",
	"global.retry":         "retry after a network error",

	"search.up":          "previous suggestion",
	"search.down":        "next suggestion",
	"search.confirm":     "open the entry / list (Pokédex, Moves, Items) or pick (Natures, Types)",
	"search.clear":       "clear the box; on an empty box, quit",
	"search.next_field":  "next box in the same tab (Type 1 ↔ Type 2)",
	"search.prev_field":  "previous box in the same tab",
	"search.next_tab":    "next tab",
	"search.prev_tab":    "previous tab",
	"search.scroll_up":   "scroll the right panel up",
	"search.scroll_down": "scroll the right panel down",

	"entry.up":          "previous row",
	"entry.down":        "next row",
	"entry.page_up":     "up 10 rows",
	"entry.page_down":   "down 10 rows",
	"entry.top":         "first row",
	"entry.bottom":      "last row",
	"entry.next_tab":    "next tab",
	"entry.prev_tab":    "previous tab",
	"entry.overview":    "Overview tab",
	"entry.abilities":   "Abilities tab",
	"entry.moves":       "Moves tab",
	"entry.family":      "Family tab",
	"entry.next_game":   "Moves: next game",
	"entry.prev_game":   "Moves: previous game",
	"entry.filter":      "Moves: filter the list",
	"entry.open":        "open the selected Pokémon, or the Pokémon with that move/ability",
	"entry.back":        "back to the previous page",
	"entry.scroll_up":   "scroll the panels up",
	"entry.scroll_down": "scroll the panels down",

	"filter.up":       "previous move",
	"filter.down":     "next move",
	"filter.apply":    "keep the filter",
	"filter.cancel":   "clear the filter",
	"filter.next_tab": "next tab",
	"filter.prev_tab": "previous tab",
}

// HelpRow is one line of the help screen.
type HelpRow struct {
	Keys Binding
	Desc string
}

// HelpSection groups the rows of one [keys.*] section.
type HelpSection struct {
	Title string
	Rows  []HelpRow
}

// HelpFor returns the key help for a screen: "search", "entry" or "list".
func (k Keys) HelpFor(screen string) []HelpSection {
	section := func(title, name string, keys any, only ...string) HelpSection {
		want := map[string]bool{}
		for _, o := range only {
			want[o] = true
		}
		v := reflect.ValueOf(keys)
		t := v.Type()
		hs := HelpSection{Title: title}
		for i := 0; i < t.NumField(); i++ {
			action := t.Field(i).Tag.Get("toml")
			if len(only) > 0 && !want[action] {
				continue
			}
			hs.Rows = append(hs.Rows, HelpRow{v.Field(i).Interface().(Binding), actionDesc[name+"."+action]})
		}
		return hs
	}
	global := section("Everywhere", "global", k.Global)
	switch screen {
	case "search":
		return []HelpSection{section("Search screen", "search", k.Search), global}
	case "list":
		return []HelpSection{section("Pokémon list", "entry", k.Entry,
			"up", "down", "page_up", "page_down", "top", "bottom", "open", "back", "scroll_up", "scroll_down"), global}
	default:
		return []HelpSection{section("Pokémon entry", "entry", k.Entry),
			section("While typing the move filter", "filter", k.Filter), global}
	}
}
