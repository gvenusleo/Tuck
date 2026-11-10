package app

import (
	_ "embed"
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

const toggleShortcut = "Alt+Space"

//go:embed resources/tray.png
var trayIcon []byte

func (a *app) ready() {
	// 面板收起后禁用 App Nap，仍允许系统正常休眠。
	release := mygo.Power.KeepRunning("保持 Tuck 后台终端运行")
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { release() })
	if runtime.GOOS == "darwin" {
		// 原生菜单先于视图处理 ⌘W；按收到命令的窗口区分标签与设置。
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{{
				Label: "关闭", Accelerator: "CmdOrCtrl+W",
				Click: func(_ *mygo.MenuItem, w *mygo.Window) { a.closeFromMenu(w) },
			}}},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleViewMenu},
			{Role: mygo.RoleWindowMenu},
		}))
	}
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
		{Label: "设置", Click: func(*mygo.MenuItem, *mygo.Window) { a.showSettings() }},
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

func (a *app) closeFromMenu(w *mygo.Window) {
	// NSPopover 不登记为 MyGo 的 FocusedWindow，原生菜单可能传入 nil。
	if w == nil {
		w = a.panel
	}
	if w != nil && w == a.panel {
		a.closeSession(a.activeID)
	} else if w != nil {
		w.Close()
	}
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
	if len(a.sessions) == 0 {
		a.newSession()
		if len(a.sessions) == 0 {
			return
		}
	}
	a.focusID = a.activeID
	a.revealTab = true
	anchor := a.tray.Bounds()
	display := mygo.Screen.DisplayNearestPoint(mygo.Point{
		X: anchor.X + anchor.Width/2,
		Y: anchor.Y + anchor.Height/2,
	})
	if anchor.Width == 0 || anchor.Height == 0 {
		display = mygo.Screen.PrimaryDisplay()
	}
	width, height := panelSize(a.config, display.WorkArea)
	panel := mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray,
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     width,
		Height:    height,
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

func panelSize(c config.Config, area mygo.Rectangle) (int, int) {
	// 为系统浮窗的箭头和边框在可用区域四周各预留 16 DIP。
	const inset = 16
	width := min(c.Width, float64(max(1, area.Width-2*inset)))
	height := min(c.Height, float64(max(1, area.Height-2*inset)))
	return int(width), int(height)
}

func (a *app) hide() {
	if a.panel != nil {
		a.panel.Close()
	}
}
