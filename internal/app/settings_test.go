package app

import (
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

func TestPreferences(t *testing.T) {
	p := preferences{
		draft:  config.Default(),
		themes: []string{config.Default().Theme, "Catppuccin Latte"},
		fonts:  []string{config.Default().Font, "Menlo"},
	}
	changes, loginChanges := 0, 0
	p.save = func() { changes++ }
	p.setOpenAtLogin = func() { loginChanges++ }
	tt := ui.NewTester(p.view, settingsWidth, settingsHeight)
	for _, field := range []struct{ label, value string }{{"终端主题", "Catppuccin Latte"}, {"终端字体", "Menlo"}, {"终端字号", "16"}} {
		if err := tt.Click(field.label); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type(field.value)
		if field.label != "终端字号" {
			tt.Key(0, ui.KeyDown)
			tt.Key(0, ui.KeyEnter)
		}
	}
	want := config.Config{Theme: "Catppuccin Latte", Font: "Menlo", FontSize: 16}
	if p.draft != want || changes != 3 {
		t.Fatalf("preferences = %+v, changes = %d; want %+v, 3", p.draft, changes, want)
	}
	if err := tt.Click("通用"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("开机自启动") || tt.HasText("终端主题") {
		t.Fatalf("unexpected general tab: %v", tt.Texts())
	}
	if err := tt.Click("开机自启动"); err != nil {
		t.Fatal(err)
	}
	if !p.openAtLogin || loginChanges != 1 {
		t.Fatalf("login switch = %v, callback count = %d", p.openAtLogin, loginChanges)
	}
	if err := tt.Click("外观"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("终端主题") || p.draft != want {
		t.Fatalf("tab switch lost preferences: %+v", p.draft)
	}
}

func TestPreferencesSaveFailure(t *testing.T) {
	for _, field := range []struct{ label, value, original string }{
		{"终端主题", "Catppuccin Latte", config.Default().Theme},
		{"终端字体", "Menlo", config.Default().Font},
		{"终端字号", "16", "12"},
	} {
		t.Run(field.label, func(t *testing.T) {
			a := &app{config: config.Default()}
			a.prefs = preferences{
				draft:  a.config,
				themes: []string{a.config.Theme, "Catppuccin Latte"},
				fonts:  []string{a.config.Font, "Menlo"},
				save:   func() { a.prefs.busy = true },
			}
			tt := ui.NewTester(a.prefs.view, settingsWidth, settingsHeight)
			if err := tt.Click(field.label); err != nil {
				t.Fatal(err)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type(field.value)
			if field.label != "终端字号" {
				tt.Key(0, ui.KeyDown)
				tt.Key(0, ui.KeyEnter)
			}
			if !a.prefs.busy || a.prefs.draft == a.config {
				t.Fatal("edit did not start saving")
			}
			a.finishPreferencesSave(a.prefs.draft, nil, os.ErrPermission)
			tt.Frame()
			if a.prefs.busy || a.prefs.draft != a.config || !tt.HasText(a.prefs.errorText) || a.prefs.errorText == "" {
				t.Fatalf("failed save did not restore preferences: %+v", a.prefs)
			}
			if !tt.Focused(field.label) {
				if err := tt.Click(field.label); err != nil {
					t.Fatal(err)
				}
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Key(ui.Cmd, ui.KeyC)
			if got := tt.Clipboard(); got != field.original {
				t.Fatalf("displayed value = %q; want %q after rollback", got, field.original)
			}
		})
	}
}

func TestLoadConfigFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Save(path, config.Config{Theme: "Tuck missing theme", Font: "Menlo", FontSize: 16}); err != nil {
		t.Fatal(err)
	}
	a := &app{}
	if err := a.loadConfig(); err != nil {
		t.Fatal(err)
	}
	if a.config.Theme != config.Default().Theme || a.config.Font != "Menlo" || a.theme == nil || a.prefs.errorText == "" {
		t.Fatalf("invalid theme fallback: %+v, error %q", a.config, a.prefs.errorText)
	}
	saved, err := config.Load(path)
	if err != nil || saved.Theme != "Tuck missing theme" {
		t.Fatalf("startup overwrote user's configuration: %+v, %v", saved, err)
	}
}
