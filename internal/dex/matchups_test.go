package dex

import "testing"

func TestEffectiveness(t *testing.T) {
	chart := TypeChart{
		"fire":     {"water": 2, "rock": 2, "fire": 0.5, "grass": 0.5, "ground": 2},
		"flying":   {"rock": 2, "electric": 2, "grass": 0.5, "ground": 0},
		"water":    {},
		"rock":     {},
		"grass":    {},
		"ground":   {},
		"electric": {},
	}
	eff := Effectiveness(chart, []string{"fire", "flying"})
	want := map[string]float64{"rock": 4, "water": 2, "electric": 2, "fire": 0.5, "grass": 0.25, "ground": 0}
	for atk, w := range want {
		if eff[atk] != w {
			t.Errorf("%s vs fire/flying = %v, want %v", atk, eff[atk], w)
		}
	}
}

func TestCapitalize(t *testing.T) {
	cases := map[string]string{
		"":                       "",
		"fire":                   "Fire",
		"élan":                   "Élan",
		"— none (single type) —": "— none (single type) —", // multi-byte first rune stays intact
	}
	for in, want := range cases {
		if got := Capitalize(in); got != want {
			t.Errorf("Capitalize(%q) = %q, want %q", in, got, want)
		}
	}
}
