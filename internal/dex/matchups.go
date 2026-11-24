package dex

import (
	"sort"
	"strings"
)

// Effectiveness returns, for every attacking type, the damage multiplier against defTypes.
func Effectiveness(chart TypeChart, defTypes []string) map[string]float64 {
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

// Group is one row of the result (e.g. all 2× weaknesses).
type Group struct {
	Mult      float64
	Label     string
	Desc      string
	ANSI      string // SGR code for the CLI
	Color     string // hex color for the TUI
	Attackers []string
}

var groupDefs = []Group{
	{Mult: 4, Label: "4×", Desc: "Very weak", ANSI: "1;31", Color: "#FF5F5F"},
	{Mult: 2, Label: "2×", Desc: "Weak", ANSI: "31", Color: "#FF9F7F"},
	{Mult: 0.5, Label: "½×", Desc: "Resists", ANSI: "32", Color: "#9FDF7F"},
	{Mult: 0.25, Label: "¼×", Desc: "Strongly resists", ANSI: "1;32", Color: "#5FD75F"},
	{Mult: 0, Label: "0×", Desc: "Immune", ANSI: "36", Color: "#5FD7FF"},
}

// GroupedEffectiveness returns the non-empty groups for defTypes, attackers sorted by name.
func GroupedEffectiveness(chart TypeChart, defTypes []string) []Group {
	eff := Effectiveness(chart, defTypes)
	var out []Group
	for _, g := range groupDefs {
		for atk, m := range eff {
			if m == g.Mult {
				g.Attackers = append(g.Attackers, atk)
			}
		}
		if len(g.Attackers) > 0 {
			sort.Strings(g.Attackers)
			out = append(out, g)
		}
	}
	return out
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

// AbilityWarnings lists "ability (effect)" for abilities that change matchups.
func AbilityWarnings(abilities []string) []string {
	var notes []string
	for _, a := range abilities {
		if n, ok := abilityNotes[a]; ok {
			notes = append(notes, a+" ("+n+")")
		}
	}
	return notes
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// DisplayName turns "charizard-mega-x" into "Charizard Mega X".
func DisplayName(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		words[i] = Capitalize(w)
	}
	return strings.Join(words, " ")
}

// UniqueTypes drops empty and duplicate entries, keeping order.
func UniqueTypes(ts ...string) []string {
	var out []string
	for _, t := range ts {
		if t == "" {
			continue
		}
		dup := false
		for _, o := range out {
			dup = dup || o == t
		}
		if !dup {
			out = append(out, t)
		}
	}
	return out
}
