package app

import (
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	terminal.View(c.Key("terminal"), a.term).Fill().AutoFocus()
}
