package app

import (
	"log"
	"math"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

const (
	settingsWidth  = 520
	settingsHeight = 360
)

type preferences struct {
	draft          config.Config
	tab            int
	formRevision   int
	numberInputs   [3]ui.Handle
	editingNumber  bool
	pendingSave    bool
	themes         []string
	fonts          []string
	openAtLogin    bool
	loading        bool
	busy           bool
	errorText      string
	save           func()
	setOpenAtLogin func()
}

func (a *app) loadConfig() error {
	var err error
	a.configPath, err = config.Path()
	if err != nil {
		return err
	}
	a.config, err = config.Load(a.configPath)
	if err != nil {
		a.prefs.errorText = "无法读取配置，已使用默认设置。"
		log.Printf("读取设置失败：%v", err)
	}
	a.theme, err = terminal.GhosttyTheme(a.config.Theme)
	if err != nil {
		a.prefs.errorText = "无法加载终端主题，已使用默认主题。"
		log.Printf("加载终端主题失败：%v", err)
		a.config.Theme = config.Default().Theme
		a.theme, err = terminal.GhosttyTheme(a.config.Theme)
		if err != nil {
			return err
		}
	}
	a.prefs.draft = a.config
	a.prefs.save = a.savePreferences
	a.prefs.setOpenAtLogin = a.setOpenAtLogin
	return nil
}

func (a *app) showSettings() {
	if a.quitting {
		return
	}
	a.hide()
	if a.settingsWindow != nil {
		a.settingsWindow.Show()
		a.settingsWindow.Focus()
		return
	}
	a.prefs.openAtLogin = mygo.App.OpenAtLogin()
	w := mygo.NewWindow(mygo.WindowOptions{
		Title:             "Tuck 设置",
		Width:             settingsWidth,
		Height:            settingsHeight,
		UseContentSize:    true,
		DisableResize:     true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableFullScreen: true,
		Content:           ui.View(a.prefs.view),
	})
	a.settingsWindow = w
	w.OnBlur(a.flushPreferences)
	w.OnClose(func(*mygo.CloseEvent) { a.flushPreferences() })
	w.OnClosed(func() {
		if a.settingsWindow == w {
			a.settingsWindow = nil
		}
	})
	if a.prefs.fonts == nil && !a.prefs.loading {
		a.prefs.loading = true
		go func() {
			themes := terminal.GhosttyThemes()
			fonts := []string{config.Default().Font}
			for _, f := range ui.FontFamilies() {
				if f.Monospaced {
					fonts = append(fonts, f.Name)
				}
			}
			slices.Sort(fonts)
			fonts = slices.Compact(fonts)
			mygo.RunOnMain(func() {
				a.prefs.themes, a.prefs.fonts = themes, fonts
				a.prefs.loading = false
				a.invalidateSettings()
			})
		}()
	}
}

func (p *preferences) view(c *ui.Context) {
	ui.Column(c).Fill().Padding(24).Gap(24).Children(func() {
		ui.Row(c).FillWidth().Center().Children(func() {
			ui.Segmented(c.Key("settings-tabs"), &p.tab, "外观", "通用").Label("设置分类")
		})
		ui.Form(c.Key([2]int{p.tab, p.formRevision}), func() {
			if p.tab == 0 {
				ui.Field(c, "终端主题", func() {
					ui.Combobox(c.Key("theme"), &p.draft.Theme, p.themes).Width(320).OnChange(p.save)
				})
				ui.Field(c, "终端字体", func() {
					ui.Combobox(c.Key("font"), &p.draft.Font, p.fonts).Width(320).OnChange(p.save)
				})
				ui.Field(c, "终端字号", func() {
					ui.NumberInput(c.Key("font-size"), &p.draft.FontSize, config.MinFontSize, config.MaxFontSize, 1).
						Width(100).Bind(&p.numberInputs[0]).OnChange(p.save)
				})
				ui.Field(c, "终端宽度（DIP）", func() {
					ui.NumberInput(c.Key("terminal-width"), &p.draft.Width, 1, math.MaxFloat64, 10).
						Width(100).Bind(&p.numberInputs[1]).OnChange(p.save)
				})
				ui.Field(c, "终端高度（DIP）", func() {
					ui.NumberInput(c.Key("terminal-height"), &p.draft.Height, 1, math.MaxFloat64, 10).
						Width(100).Bind(&p.numberInputs[2]).OnChange(p.save)
				})
			} else {
				ui.Field(c, "开机自启动", func() {
					ui.Switch(c.Key("open-at-login"), &p.openAtLogin).OnChange(p.setOpenAtLogin)
				})
			}
		}).AlignSelf(ui.Center).Disabled(p.loading || p.busy)
		p.editingNumber = false
		for _, input := range p.numberInputs {
			p.editingNumber = p.editingNumber || input.FocusWithin(c)
		}
		if p.pendingSave && !p.editingNumber {
			p.pendingSave = false
			p.save()
		}
		ui.Spacer(c)
		switch {
		case p.errorText != "":
			ui.Text(c, p.errorText).TextColor(c.Theme().Danger).Wrap()
		case p.loading:
			ui.Text(c, "正在加载设置…").TextColor(c.Theme().TextMuted)
		case p.busy:
			ui.Text(c, "正在保存…").TextColor(c.Theme().TextMuted)
		}
	})
}

func (a *app) flushPreferences() {
	a.prefs.editingNumber = false
	if a.prefs.pendingSave {
		a.savePreferences()
	}
}

func (a *app) savePreferences() {
	if a.prefs.busy || a.quitting {
		return
	}
	if a.prefs.editingNumber {
		a.prefs.pendingSave = true
		return
	}
	a.prefs.pendingSave = false
	next, path := a.prefs.draft, a.configPath
	a.prefs.busy, a.prefs.errorText = true, ""
	a.configWrites.Add(1)
	go func() {
		theme, err := terminal.GhosttyTheme(next.Theme)
		if err == nil {
			err = config.Save(path, next)
		}
		a.configWrites.Done()
		mygo.RunOnMain(func() { a.finishPreferencesSave(next, theme, err) })
	}()
}

func (a *app) finishPreferencesSave(next config.Config, theme *terminal.Theme, err error) {
	a.prefs.busy = false
	if err != nil {
		a.prefs.draft = a.config
		// 重建表单，清除原生输入框持有的未保存文字。
		a.prefs.formRevision++
		a.prefs.errorText = "设置未保存，请检查主题或配置文件权限。"
		log.Printf("保存设置失败：%v", err)
	} else {
		previous := a.config
		a.config, a.theme = next, theme
		if a.term != nil {
			if next.Font != previous.Font || next.FontSize != previous.FontSize {
				a.term.SetFont(terminal.Font{Family: next.Font, Size: float32(next.FontSize)})
			}
			if next.Theme != previous.Theme {
				a.term.SetTheme(theme, nil)
			}
		}
	}
	a.invalidateSettings()
}

func (a *app) setOpenAtLogin() {
	if a.prefs.busy || a.quitting {
		return
	}
	enabled := a.prefs.openAtLogin
	a.prefs.busy, a.prefs.errorText = true, ""
	go func() {
		err := mygo.App.SetOpenAtLogin(enabled)
		actual := mygo.App.OpenAtLogin()
		mygo.RunOnMain(func() {
			a.prefs.busy = false
			a.prefs.openAtLogin = actual
			if err != nil {
				a.prefs.errorText = "无法更改开机启动，请在系统设置中检查登录项。"
				log.Printf("更改开机启动失败：%v", err)
			}
			a.invalidateSettings()
		})
	}()
}

func (a *app) invalidateSettings() {
	if a.settingsWindow != nil {
		a.settingsWindow.Invalidate()
	}
}
