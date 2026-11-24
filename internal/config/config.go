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
	Keys    Keys    `toml:"keys"`
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
}

type SearchKeys struct {
	Up        Binding `toml:"up"`
	Down      Binding `toml:"down"`
	Confirm   Binding `toml:"confirm"`
	Clear     Binding `toml:"clear"`
	NextField Binding `toml:"next_field"`
	PrevField Binding `toml:"prev_field"`
	NextTab   Binding `toml:"next_tab"`
	PrevTab   Binding `toml:"prev_tab"`
}

type EntryKeys struct {
	Up        Binding `toml:"up"`
	Down      Binding `toml:"down"`
	PageUp    Binding `toml:"page_up"`
	PageDown  Binding `toml:"page_down"`
	Top       Binding `toml:"top"`
	Bottom    Binding `toml:"bottom"`
	NextTab   Binding `toml:"next_tab"`
	PrevTab   Binding `toml:"prev_tab"`
	Overview  Binding `toml:"overview"`
	Abilities Binding `toml:"abilities"`
	Moves     Binding `toml:"moves"`
	Family    Binding `toml:"family"`
	NextGame  Binding `toml:"next_game"`
	PrevGame  Binding `toml:"prev_game"`
	Filter    Binding `toml:"filter"`
	Open      Binding `toml:"open"`
	Back      Binding `toml:"back"`
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
