// Package fuzzy provides typo-tolerant name matching.
package fuzzy

import (
	"sort"
	"strings"
)

func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), "-")
	return strings.NewReplacer("'", "", ".", "").Replace(s)
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

// similarity is a 0..1 score based on edit distance.
func similarity(a, b string) float64 {
	la, lb := len([]rune(a)), len([]rune(b))
	if la+lb == 0 {
		return 1
	}
	return 1 - float64(levenshtein(a, b))/float64(max(la, lb))
}

// Suggest returns up to n candidates: prefix matches, then similar names, then substring matches.
func Suggest(query string, candidates []string, n int) []string {
	q := Normalize(query)
	if q == "" {
		return nil
	}
	var prefix, substr []string
	type scored struct {
		name  string
		score float64
	}
	var close []scored
	for _, c := range candidates {
		switch {
		case strings.HasPrefix(c, q):
			prefix = append(prefix, c)
		case strings.Contains(c, q):
			substr = append(substr, c)
		default:
			if s := similarity(q, c); s >= 0.6 {
				close = append(close, scored{c, s})
			}
		}
	}
	// Base forms first (e.g. charizard before charizard-mega-x)
	sort.SliceStable(prefix, func(i, j int) bool {
		di, dj := strings.Count(prefix[i], "-"), strings.Count(prefix[j], "-")
		if di != dj {
			return di < dj
		}
		return len(prefix[i]) < len(prefix[j])
	})
	sort.SliceStable(close, func(i, j int) bool { return close[i].score > close[j].score })

	out := append([]string{}, prefix...)
	for _, c := range close {
		out = append(out, c.name)
	}
	out = append(out, substr...)
	if len(out) > n {
		out = out[:n]
	}
	return out
}
