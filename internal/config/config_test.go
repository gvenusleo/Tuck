package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, tc := range []struct {
		name string
		xdg  string
		want string
	}{
		{"default", "", filepath.Join(home, ".config", "tuck", "config.json")},
		{"xdg", filepath.Join(home, "custom"), filepath.Join(home, "custom", "tuck", "config.json")},
		{"relative xdg ignored", "relative", filepath.Join(home, ".config", "tuck", "config.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tc.xdg)
			got, err := Path()
			if err != nil || got != tc.want {
				t.Fatalf("Path() = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want Config
		fail bool
	}{
		{"missing", "", Default(), false},
		{"partial", `{"terminal_font_size": 16}`, Config{Theme: Default().Theme, Font: Default().Font, FontSize: 16, Width: 720, Height: 420, TabPosition: TabsTop}, false},
		{"invalid tab position", `{"terminal_tab_position": "left"}`, Default(), true},
		{"malformed", `{"terminal_theme":`, Default(), true},
		{"empty theme", `{"terminal_theme": ""}`, Default(), true},
		{"empty font", `{"terminal_font": " "}`, Default(), true},
		{"small font", `{"terminal_font_size": 0}`, Default(), true},
		{"large font", `{"terminal_font_size": 100}`, Default(), true},
		{"zero width", `{"terminal_width": 0}`, Default(), true},
		{"negative height", `{"terminal_height": -1}`, Default(), true},
		{"fractional width", `{"terminal_width": 720.5}`, Default(), true},
		{"fractional height", `{"terminal_height": 420.5}`, Default(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if tc.data != "" {
				if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Load(path)
			if (err != nil) != tc.fail || got != tc.want {
				t.Fatalf("Load() = %+v, %v; want %+v, failure %v", got, err, tc.want, tc.fail)
			}
		})
	}
}

func TestSaveRoundTripAndPreserveOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuck", "config.json")
	want := Config{Theme: "Catppuccin Latte", Font: "Menlo", FontSize: 16, Width: 10000, Height: 8000, TabPosition: TabsBottom}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want {
		t.Fatalf("round trip = %+v, %v; want %+v", got, err, want)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invalid := want
	invalid.FontSize = 100
	if err := Save(path, invalid); err == nil {
		t.Fatal("invalid configuration was saved")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatalf("failed save changed original: %q, %v", after, err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "config.json" {
		t.Fatalf("temporary files left after save: %v, %v", entries, err)
	}
	if err := Save(filepath.Join(path, "child"), want); err == nil {
		t.Fatal("saving under a file unexpectedly succeeded")
	}
}
