package app

import (
	"errors"
	"log"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"liuyu.xin/tuck/internal/config"
)

type app struct {
	tray           *mygo.Tray
	panel          *mygo.Window
	settingsWindow *mygo.Window
	term           *terminal.Terminal
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
	a.closeTerminal()
	if a.tray != nil {
		a.tray.Destroy()
		a.tray = nil
	}
}

func (a *app) closeTerminal() {
	if a.term != nil {
		term := a.term
		a.term = nil
		if err := term.Close(); err != nil {
			log.Printf("关闭终端失败：%v", err)
		}
	}
}

func (a *app) terminalExited(code int) {
	// 终端回调不等待主线程，清理和错误对话框由主线程处理。
	go mygo.RunOnMain(func() {
		if a.quitting {
			return
		}
		if code < 0 {
			a.fail("无法启动终端", errors.New(a.term.Text()))
			return
		}
		a.hide()
		a.closeTerminal()
	})
}

func (a *app) fail(message string, err error) {
	// 错误对话框运行嵌套事件循环，期间不再接受打开请求。
	a.quitting = true
	log.Printf("%s：%v", message, err)
	if _, dialogErr := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   "Tuck",
		Message: message,
		Detail:  err.Error(),
		Buttons: []string{"退出"},
	}); dialogErr != nil {
		log.Printf("显示错误对话框失败：%v", dialogErr)
	}
	mygo.App.Quit()
}

// Run 启动 Tuck 并运行原生事件循环。
func Run() error {
	return (&app{}).run()
}
