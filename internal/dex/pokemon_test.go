package dex

import "testing"

func TestSortMoves(t *testing.T) {
	moves := []MoveRef{
		{Name: "earthquake", Method: "machine"},
		{Name: "outrage", Method: "egg"},
		{Name: "dragon-claw", Method: "level-up", Level: 33},
		{Name: "tackle", Method: "level-up", Level: 1},
		{Name: "draco-meteor", Method: "tutor"},
	}
	sortMoves(moves)
	want := []string{"tackle", "dragon-claw", "earthquake", "draco-meteor", "outrage"}
	for i, w := range want {
		if moves[i].Name != w {
			t.Fatalf("position %d = %s, want %s (got %v)", i, moves[i].Name, w, moves)
		}
	}
}

func TestWithChance(t *testing.T) {
	ten := 10
	if got := WithChance("Has a $effect_chance% chance to burn.", &ten); got != "Has a 10% chance to burn." {
		t.Errorf("got %q", got)
	}
	if got := WithChance("Has a $effect_chance% chance to burn.", nil); got != "Has a chance to burn." {
		t.Errorf("got %q", got)
	}
}

func TestCleanText(t *testing.T) {
	if got := cleanText("A strange seed was\nplanted\fon its back."); got != "A strange seed was planted on its back." {
		t.Errorf("got %q", got)
	}
}

func TestDescribeEvolution(t *testing.T) {
	lv := 36
	happy := 160
	cases := []struct {
		d    apiEvoDetail
		want string
	}{
		{apiEvoDetail{Trigger: namedURL{Name: "level-up"}, MinLevel: &lv}, "Level 36"},
		{apiEvoDetail{Trigger: namedURL{Name: "use-item"}, Item: &namedURL{Name: "water-stone"}}, "Use Water Stone"},
		{apiEvoDetail{Trigger: namedURL{Name: "trade"}, HeldItem: &namedURL{Name: "metal-coat"}}, "Trade holding Metal Coat"},
		{apiEvoDetail{Trigger: namedURL{Name: "level-up"}, MinHappiness: &happy, TimeOfDay: "night"}, "Level up with high friendship during the night"},
		{apiEvoDetail{Trigger: namedURL{Name: "level-up"}, Location: &namedURL{Name: "eterna-forest"}}, "Level up at a special location"},
	}
	for _, c := range cases {
		if got := describeEvolution(c.d); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestFindEvo(t *testing.T) {
	root := &EvoNode{Species: "charmander", Next: []EvoNode{
		{Species: "charmeleon", Next: []EvoNode{{Species: "charizard"}}},
	}}
	node, parent := FindEvo(root, "charizard")
	if node == nil || parent == nil || parent.Species != "charmeleon" {
		t.Fatalf("FindEvo(charizard) = %v, %v", node, parent)
	}
	if node, parent := FindEvo(root, "charmander"); node == nil || parent != nil {
		t.Fatalf("FindEvo(charmander) = %v, %v", node, parent)
	}
	if got := HowText([]string{"Level up at a special location", "Use Leaf Stone"}); got != "Level up at a special location or use Leaf Stone" {
		t.Errorf("HowText = %q", got)
	}
}
