package app

import (
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

func (a *app) view(c *ui.Context) {
	c.OnShortcut(ui.Cmd, ui.KeyT, a.newSession)
	c.OnShortcut(ui.Cmd, ui.KeyW, func() { a.closeSession(a.activeID) })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyBracketLeft, func() { a.stepSession(-1) })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyBracketRight, func() { a.stepSession(1) })
	c.OnShortcut(ui.Cmd, ui.KeyComma, a.showSettings)
	for i := 0; i < 9; i++ {
		c.OnShortcut(ui.Cmd, ui.Key1+ui.Key(i), func() {
			index := i
			if i == 8 {
				index = len(a.sessions) - 1
			}
			if index >= 0 && index < len(a.sessions) {
				a.selectSession(a.sessions[index].id)
			}
		})
	}
	ui.Column(c).Fill().Children(func() {
		if len(a.sessions) > 1 && a.config.TabPosition != config.TabsBottom {
			a.tabBar(c)
		}
		if i := a.sessionIndex(a.activeID); i >= 0 {
			s := a.sessions[i]
			terminal.View(c.Key(s.id), s.term).Fill().MinHeight(0).
				Label("当前终端").FocusBind(&a.focusID, s.id)
		}
		if len(a.sessions) > 1 && a.config.TabPosition == config.TabsBottom {
			a.tabBar(c)
		}
	})
}
