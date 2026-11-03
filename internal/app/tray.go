package app

import (
	_ "embed"
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

const toggleShortcut = "Alt+Space"

//go:embed resources/tray.png
var trayIcon []byte

func (a *app) ready() {
	var err error
	a.tray, err = mygo.NewTray(mygo.TrayOptions{
		Icon:           trayIcon,
		IconIsTemplate: true,
		ToolTip:        "Tuck",
	})
	if err != nil {
		a.fail("无法创建托盘图标", err)
		return
	}
	if err := mygo.GlobalShortcut.Register(toggleShortcut, a.toggle); err != nil {
		log.Printf("注册 Option+空格快捷键失败：%v", err)
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开 Tuck", Click: func(*mygo.MenuItem, *mygo.Window) { a.show() }},
		mygo.Separator(),
		{Label: "退出 Tuck", Role: mygo.RoleQuit},
	})
	if runtime.GOOS == "linux" {
		a.tray.SetMenu(menu)
		return
	}
	a.tray.OnClick(a.toggle)
	a.tray.OnRightClick(func() {
		a.hide()
		// macOS 设置常驻菜单会接管左键点击，因此只在右键弹出期间设置。
		a.tray.SetMenu(menu)
		defer a.tray.SetMenu(nil)
		a.tray.PopUpMenu()
	})
}

func (a *app) toggle() {
	if a.panel != nil {
		a.hide()
		return
	}
	a.show()
}

func (a *app) show() {
	if a.quitting || a.tray == nil || a.panel != nil {
		return
	}
	if a.term == nil {
		theme, err := terminal.GhosttyTheme("Ghostty Default Style Dark")
		if err != nil {
			a.fail("无法加载终端主题", err)
			return
		}
		term, err := terminal.New(terminal.Options{
			Font:   terminal.Font{Family: "JetBrainsMono Nerd Font", Size: 12},
			Theme:  theme,
			OnExit: a.terminalExited,
		})
		if err != nil {
			a.fail("无法启动终端", err)
			return
		}
		a.term = term
	}
	panel := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     720,
		Height:    420,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.view),
	})
	a.panel = panel
	panel.OnClosed(func() {
		if a.panel == panel {
			a.panel = nil
		}
	})
}

func (a *app) hide() {
	if a.panel != nil {
		a.panel.Close()
	}
}
