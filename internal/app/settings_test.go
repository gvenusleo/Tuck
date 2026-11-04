package app

import (
	"math"
	"os"
	"testing"

	"github.com/egoist/mygo"
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
	for _, field := range []struct{ label, value string }{
		{"终端主题", "Catppuccin Latte"},
		{"终端字体", "Menlo"},
		{"终端字号", "16"},
		{"面板宽度（DIP）", "10000"},
		{"面板高度（DIP）", "8000"},
	} {
		if err := tt.Click(field.label); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type(field.value)
		if field.label == "终端主题" || field.label == "终端字体" {
			tt.Key(0, ui.KeyDown)
			tt.Key(0, ui.KeyEnter)
		}
	}
	if err := tt.Click("下方"); err != nil {
		t.Fatal(err)
	}
	want := config.Config{Theme: "Catppuccin Latte", Font: "Menlo", FontSize: 16, Width: 10000, Height: 8000, TabPosition: config.TabsBottom}
	if p.draft != want || changes != 6 {
		t.Fatalf("preferences = %+v, changes = %d; want %+v, 6", p.draft, changes, want)
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
		{"面板宽度（DIP）", "1000", "720"},
	} {
		t.Run(field.label, func(t *testing.T) {
			a := &app{config: config.Default()}
			a.prefs = preferences{
				draft:  a.config,
				themes: []string{a.config.Theme, "Catppuccin Latte"},
				save:   func() { a.prefs.busy = true },
			}
			tt := ui.NewTester(a.prefs.view, settingsWidth, settingsHeight)
			if err := tt.Click(field.label); err != nil {
				t.Fatal(err)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Type(field.value)
			if field.label == "终端主题" {
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
	t.Run("标签栏位置", func(t *testing.T) {
		a := &app{config: config.Default()}
		a.prefs = preferences{draft: a.config, save: func() { a.prefs.busy = true }}
		tt := ui.NewTester(a.prefs.view, settingsWidth, settingsHeight)
		if err := tt.Click("下方"); err != nil {
			t.Fatal(err)
		}
		a.finishPreferencesSave(a.prefs.draft, nil, os.ErrPermission)
		tt.Frame()
		if a.prefs.draft.TabPosition != config.TabsTop || a.prefs.errorText == "" {
			t.Fatal("failed save did not restore tab position")
		}
		if err := tt.Click("上方"); err != nil {
			t.Fatal(err)
		}
		if a.prefs.busy {
			t.Fatal("restored segmented control still showed the unsaved choice")
		}
	})
}

func TestPanelSize(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height float64
		wantW, wantH  int
	}{
		{"default", 720, 420, 720, 420},
		{"wide", 10000, 420, 1408, 420},
		{"tall", 720, 8000, 720, 843},
		{"both", math.MaxFloat64, math.MaxFloat64, 1408, 843},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := config.Default()
			c.Width, c.Height = tc.width, tc.height
			w, h := panelSize(c, mygo.Rectangle{X: -1440, Y: 25, Width: 1440, Height: 875})
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("panel size = %d × %d; want %d × %d", w, h, tc.wantW, tc.wantH)
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
	c := config.Default()
	c.Theme, c.Font, c.FontSize = "Tuck missing theme", "Menlo", 16
	if err := config.Save(path, c); err != nil {
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
