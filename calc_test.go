package main

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
	eff := effectiveness(chart, []string{"fire", "flying"})
	want := map[string]float64{"rock": 4, "water": 2, "electric": 2, "fire": 0.5, "grass": 0.25, "ground": 0}
	for atk, w := range want {
		if eff[atk] != w {
			t.Errorf("%s vs fire/flying = %v, want %v", atk, eff[atk], w)
		}
	}
}
