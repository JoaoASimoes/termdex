package fuzzy

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"  Mr. Mime ": "mr-mime",
		"Farfetch'd":  "farfetchd",
		"GARCHOMP":    "garchomp",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0}, {"fire", "fire", 0}, {"fary", "fairy", 1}, {"kitten", "sitting", 3},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSuggest(t *testing.T) {
	names := []string{"charizard", "charizard-mega-x", "charizard-mega-y", "giratina-altered", "giratina-origin", "pikachu"}

	if got := Suggest("charizrd", names, 8); len(got) == 0 || got[0] != "charizard" {
		t.Errorf("Suggest(charizrd) = %v, want charizard first", got)
	}
	got := Suggest("giratina", names, 8)
	want := []string{"giratina-origin", "giratina-altered"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Suggest(giratina) = %v, want %v", got, want)
	}
	if got := Suggest("xyzxyz", names, 8); len(got) != 0 {
		t.Errorf("Suggest(xyzxyz) = %v, want none", got)
	}
}
