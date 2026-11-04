package app

import (
	"log"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

type app struct {
	tray           *mygo.Tray
	panel          *mygo.Window
	settingsWindow *mygo.Window
	sessions       []*session
	activeID       uint64
	nextSessionID  uint64
	focusID        uint64
	tabList        ui.Handle
	tabScroll      ui.ScrollState
	revealTab      bool
	theme          *terminal.Theme
	config         config.Config
	configPath     string
	configWrites   sync.WaitGroup
	prefs          preferences
	quitting       bool
}

func (a *app) run() error {
	mygo.App.SetName("Tuck")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if err := a.loadConfig(); err != nil {
		return err
	}
	mygo.App.OnSecondInstance(func([]string, string) { mygo.App.WhenReady(a.show) })
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			mygo.App.WhenReady(a.show)
		}
	})
	mygo.App.OnDidBecomeActive(func() {
		if a.settingsWindow != nil && !a.prefs.busy {
			a.prefs.openAtLogin = mygo.App.OpenAtLogin()
			a.invalidateSettings()
		}
	})
	// 浮窗关闭后，会话和托盘仍需继续运行。
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.flushPreferences()
		a.quitting = true
	})
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.close() })
	mygo.App.WhenReady(a.ready)
	return mygo.App.Run()
}

func (a *app) close() {
	a.quitting = true
	a.configWrites.Wait()
	mygo.GlobalShortcut.Unregister(toggleShortcut)
	a.hide()
	for len(a.sessions) > 0 {
		a.closeSession(a.sessions[0].id)
	}
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
}

func (a *app) fail(message string, err error) {
	// 错误对话框运行嵌套事件循环，期间不再接受打开请求。
	a.quitting = true
	showError(message, err, "退出")
	mygo.App.Quit()
}

// showError 记录错误原因并显示系统对话框。
func showError(message string, err error, buttons ...string) {
	log.Printf("%s：%v", message, err)
	if _, dialogErr := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   "Tuck",
		Message: message,
		Detail:  err.Error(),
		Buttons: buttons,
	}); dialogErr != nil {
		log.Printf("显示错误对话框失败：%v", dialogErr)
	}
}

// Run 启动 Tuck 并运行原生事件循环。
func Run() error {
	return (&app{}).run()
}
