package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultIsValid(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}
	if !c.Keys.Search.Down.Matches("ctrl+j") || !c.Keys.Global.Search.Matches("ctrl+f") {
		t.Errorf("unexpected defaults: %+v", c.Keys)
	}
	if c.General.MaxRequests != 6 || !c.General.Sprites {
		t.Errorf("unexpected general defaults: %+v", c.General)
	}
}

func TestMissingFileGivesDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c, Default()) {
		t.Error("missing file should give the defaults")
	}
}

func TestOverrideKeepsOtherDefaults(t *testing.T) {
	c, err := Load(write(t, "[general]\nsprites = false\n[keys.search]\ndown = [\"ctrl+d\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.General.Sprites {
		t.Error("sprites should be off")
	}
	if !reflect.DeepEqual(c.Keys.Search.Down, Binding{"ctrl+d"}) {
		t.Errorf("search.down = %v, want [ctrl+d] (replaces the defaults)", c.Keys.Search.Down)
	}
	if !c.Keys.Search.Up.Matches("ctrl+k") || c.General.MaxRequests != 6 {
		t.Error("settings that were not set must keep their defaults")
	}
}

func TestUnknownSettingIsAnError(t *testing.T) {
	_, err := Load(write(t, "[keys.search]\nfly = [\"x\"]\n"))
	if err == nil || !strings.Contains(err.Error(), "keys.search.fly") {
		t.Fatalf("want unknown-setting error, got %v", err)
	}
}

func TestConflictingKeysAreAnError(t *testing.T) {
	// ctrl+f is the global "search" key, so it cannot also move down in the list.
	_, err := Load(write(t, "[keys.search]\ndown = [\"ctrl+f\"]\n"))
	if err == nil || !strings.Contains(err.Error(), `"ctrl+f"`) {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestBadValuesAreAnError(t *testing.T) {
	if _, err := Load(write(t, "[general]\nmax_requests = 0\n")); err == nil {
		t.Fatal("max_requests = 0 should be rejected")
	}
}

func TestHelpLabel(t *testing.T) {
	if got := (Binding{"up", "ctrl+k", "ctrl+p"}).Help(); got != "↑/ctrl+k" {
		t.Errorf("Help = %q", got)
	}
	if got := (Binding{}).Help(); got != "(unbound)" {
		t.Errorf("Help = %q", got)
	}
}

// configs/config.example.toml is the copy published in the repository; it must stay
// identical to the embedded defaults.
func TestExampleConfigMatchesDefaults(t *testing.T) {
	example, err := os.ReadFile("../../configs/config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	if string(example) != string(DefaultTOML) {
		t.Fatal("configs/config.example.toml differs from internal/config/default.toml; copy it over")
	}
}
