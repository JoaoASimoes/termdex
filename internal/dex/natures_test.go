package dex

import (
	"reflect"
	"testing"
)

var testNatures = []Nature{
	{Name: "adamant", Increased: "attack", Decreased: "special-attack", Likes: "spicy", Hates: "dry"},
	{Name: "hardy", Increased: "attack", Decreased: "attack"},
	{Name: "jolly", Increased: "speed", Decreased: "special-attack", Likes: "sweet", Hates: "dry"},
	{Name: "timid", Increased: "speed", Decreased: "attack", Likes: "sweet", Hates: "spicy"},
}

func TestSuggestNaturesByStat(t *testing.T) {
	if got, want := SuggestNatures("speed", testNatures), []string{"jolly", "timid"}; !reflect.DeepEqual(got, want) {
		t.Errorf("speed: got %v, want %v", got, want)
	}
	if got, want := SuggestNatures("-spa", testNatures), []string{"adamant", "jolly"}; !reflect.DeepEqual(got, want) {
		t.Errorf("-spa: got %v, want %v", got, want)
	}
	if got := SuggestNatures("adamnt", testNatures); len(got) == 0 || got[0] != "adamant" {
		t.Errorf("adamnt: got %v", got)
	}
	if got := SuggestNatures("", testNatures); len(got) != len(testNatures) {
		t.Errorf("empty query should list all natures, got %v", got)
	}
}

func TestNatureAt(t *testing.T) {
	if n, ok := NatureAt(testNatures, "speed", "attack"); !ok || n.Name != "timid" {
		t.Errorf("NatureAt(speed, attack) = %v, %v", n, ok)
	}
	if n, ok := NatureAt(testNatures, "attack", "attack"); !ok || !n.Neutral() {
		t.Errorf("diagonal should be neutral, got %v, %v", n, ok)
	}
}
