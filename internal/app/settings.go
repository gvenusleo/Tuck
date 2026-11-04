package app

import (
	"log"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

const (
	settingsWidth  = 520
	settingsHeight = 400
)

type preferences struct {
	draft          config.Config
	tab            int
	tabPosition    int
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
	a.prefs.tabPosition = tabPositionIndex(a.config.TabPosition)
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
		Content:           ui.View(a.settingsView),
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

// tabPositionIndex 与 tabPositionFor 在配置与分段控件之间转换：配置保存
// 可读的 top/bottom，控件使用 0/1 下标。
func tabPositionIndex(position string) int {
	if position == config.TabsBottom {
		return 1
	}
	return 0
}

func tabPositionFor(index int) string {
	if index == 1 {
		return config.TabsBottom
	}
	return config.TabsTop
}

func (a *app) flushPreferences() {
	a.prefs.editingNumber = false
	if a.prefs.pendingSave {
		a.savePreferences()
	}
}

func (a *app) savePreferences() {
	if a.quitting {
		return
	}
	if a.prefs.busy {
		// 保存进行中：登记这次改动，等当前保存结束后再写一次。
		a.prefs.pendingSave = true
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
		a.prefs.tabPosition = tabPositionIndex(a.config.TabPosition)
		a.prefs.pendingSave = false
		// 重建表单，清除原生输入框持有的未保存文字。
		a.prefs.formRevision++
		a.prefs.errorText = "设置未保存，请检查主题或配置文件权限。"
		log.Printf("保存设置失败：%v", err)
	} else {
		previous := a.config
		a.config, a.theme = next, theme
		for _, s := range a.sessions {
			if next.Font != previous.Font || next.FontSize != previous.FontSize {
				s.term.SetFont(terminal.Font{Family: next.Font, Size: float32(next.FontSize)})
			}
			if next.Theme != previous.Theme {
				s.term.SetTheme(theme, nil)
			}
		}
		if next.TabPosition != previous.TabPosition {
			a.revealTab = true
			a.invalidatePanel()
		}
	}
	a.invalidateSettings()
	if a.prefs.pendingSave {
		a.prefs.pendingSave = false
		a.savePreferences()
	}
}

func (a *app) setOpenAtLogin() {
	if a.quitting {
		return
	}
	if a.prefs.busy {
		// 保存进行中：开关恢复为系统实际状态，避免显示未生效的选择。
		a.prefs.openAtLogin = mygo.App.OpenAtLogin()
		a.invalidateSettings()
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
