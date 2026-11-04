package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

// pickerWidth 容纳主题与字体名称，超长时由输入框自行截断。
const pickerWidth = 220

func (a *app) settingsView(c *ui.Context) {
	c.OnShortcut(ui.Cmd, ui.KeyW, func() {
		if a.settingsWindow != nil {
			a.settingsWindow.Close()
		}
	})
	a.prefs.view(c)
}

func (p *preferences) view(c *ui.Context) {
	ui.Column(c).Fill().Padding(24).Gap(24).Children(func() {
		ui.Row(c).FillWidth().Center().Children(func() {
			ui.Segmented(c.Key("settings-tabs"), &p.tab, "外观", "通用").Label("设置分类")
		})
		ui.Form(c.Key([2]int{p.tab, p.formRevision}), func() {
			if p.tab == 0 {
				ui.Field(c, "终端主题", func() {
					ui.Combobox(c.Key("theme"), &p.draft.Theme, p.themes).Width(pickerWidth).OnChange(p.save)
				})
				ui.Field(c, "终端字体", func() {
					ui.Combobox(c.Key("font"), &p.draft.Font, p.fonts).Width(pickerWidth).OnChange(p.save)
				})
				ui.Field(c, "终端字号", func() {
					ui.NumberInput(c.Key("font-size"), &p.draft.FontSize, config.MinFontSize, config.MaxFontSize, 1).
						Width(100).Bind(&p.numberInputs[0]).OnChange(p.save)
				})
				ui.Field(c, "标签栏位置", func() {
					ui.Segmented(c.Key("tab-position"), &p.tabPosition, "上方", "下方").AlignSelf(ui.Start).OnChange(func() {
						p.draft.TabPosition = tabPositionFor(p.tabPosition)
						p.save()
					})
				})
				ui.Field(c, "面板宽度（DIP）", func() {
					ui.NumberInput(c.Key("terminal-width"), &p.draft.Width, 1, math.MaxFloat64, 10).
						Width(100).Bind(&p.numberInputs[1]).OnChange(p.save)
				})
				ui.Field(c, "面板高度（DIP）", func() {
					ui.NumberInput(c.Key("terminal-height"), &p.draft.Height, 1, math.MaxFloat64, 10).
						Width(100).Bind(&p.numberInputs[2]).OnChange(p.save)
				})
			} else {
				ui.Field(c, "开机自启动", func() {
					ui.Switch(c.Key("open-at-login"), &p.openAtLogin).OnChange(p.setOpenAtLogin)
				})
			}
		}).AlignSelf(ui.Center).Disabled(p.loading)
		p.editingNumber = false
		for _, input := range p.numberInputs {
			p.editingNumber = p.editingNumber || input.FocusWithin(c)
		}
		if p.pendingSave && !p.editingNumber && !p.busy {
			p.pendingSave = false
			p.save()
		}
		ui.Spacer(c)
		switch {
		case p.errorText != "":
			ui.Text(c, p.errorText).TextColor(c.Theme().Danger).Wrap()
		case p.loading:
			ui.Text(c, "正在加载设置…").TextColor(c.Theme().TextMuted)
		}
	})
}
