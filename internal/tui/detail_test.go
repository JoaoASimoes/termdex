package tui

import (
	"testing"

	"github.com/JoaoASimoes/termdex/internal/dex"
)

func TestDefaultGameSkipsSpinOffs(t *testing.T) {
	p := dex.Pokemon{Learnsets: []dex.Learnset{
		{VersionGroup: "champions", Moves: []dex.MoveRef{{Name: "bite", Method: "train"}}},
		{VersionGroup: "scarlet-violet", Moves: []dex.MoveRef{{Name: "bite", Method: "level-up", Level: 1}}},
	}}
	if g := defaultGame(p); g != 1 {
		t.Errorf("defaultGame = %d, want 1", g)
	}
}
