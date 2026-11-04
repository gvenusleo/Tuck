package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// tabKey 是标签的界面身份，与会话 ID 区分开，避免两类元素共用同一个 Key。
type tabKey struct{ sessionID uint64 }

// tabItem 是一帧构建标签栏所需的会话快照。菜单动作可能在构建期间结束会话，
// 先取快照，遍历就不会读到被移除的项。
type tabItem struct {
	id      uint64
	name    string
	tooltip string
}

func (a *app) tabBar(c *ui.Context) {
	items := make([]tabItem, len(a.sessions))
	for i, s := range a.sessions {
		items[i].id = s.id
		items[i].name, items[i].tooltip = s.name(i)
	}
	background, foreground := c.Theme().Background, c.Theme().Text
	if a.theme != nil {
		background, foreground = a.theme.Background, a.theme.Foreground
	}
	selected := a.sessionIndex(a.activeID)
	ui.ScrollHorizontal(c.Key("tab-scroll")).FillWidth().Height(22).Shrink(0).
		Background(background).Scrollbars(false).TrackScroll(&a.tabScroll).
		ContextMenu(func(m *ui.Menu) { a.sessionMenu(m, a.activeID) }).Children(func() {
		keepTabFocus := a.tabList.FocusWithin(c)
		tabs := ui.TabsBase(c.Key("tabs"), &selected, len(items))
		tabs.List.Bind(&a.tabList).KeepFocus().Label("终端标签页").Children(func() {
			for i, item := range items {
				c.Key(tabKey{item.id})
				tab := tabs.Tab(i).Height(22).MinWidth(80).MaxWidth(180).
					Padding(0, 6).AlignItems(ui.Center).Label(item.name).Tooltip(item.tooltip)
				text := background.Mix(foreground, 0.82)
				if i == selected {
					tab.Background(background.Mix(foreground, 0.08))
					text = foreground
					if a.revealTab {
						tab.ScrollIntoView()
						a.revealTab = false
					}
				} else if tab.Hovered() {
					tab.Background(background.Mix(foreground, 0.04))
				}
				tab.ContextMenu(func(m *ui.Menu) { a.sessionMenu(m, item.id) }).Children(func() {
					ui.Text(c, item.name).SingleLine().Ellipsis("…").FontSize(11).
						TextColor(text).TextAlign(ui.Center).Grow(1).MinWidth(0)
				})
			}
		})
		tabs.List.OnChange(func() {
			if selected >= 0 && selected < len(a.sessions) {
				a.selectSession(a.sessions[selected].id)
				if keepTabFocus {
					a.focusID = 0
				}
			}
		})
	})
}

func (a *app) sessionMenu(m *ui.Menu, target uint64) {
	if m.Item("新建标签页").Shortcut(ui.Cmd, ui.KeyT).Chosen() {
		a.newSession()
	}
	if m.Item("关闭标签页").Shortcut(ui.Cmd, ui.KeyW).Disabled(a.sessionIndex(target) < 0).Chosen() {
		a.closeSession(target)
	}
	if m.Item("上一个标签页").Shortcut(ui.Cmd|ui.Shift, ui.KeyBracketLeft).Disabled(len(a.sessions) < 2).Chosen() {
		a.stepSession(-1)
	}
	if m.Item("下一个标签页").Shortcut(ui.Cmd|ui.Shift, ui.KeyBracketRight).Disabled(len(a.sessions) < 2).Chosen() {
		a.stepSession(1)
	}
	m.Separator()
	for i, s := range a.sessions {
		name, _ := s.name(i)
		item := m.Item(fmt.Sprintf("%d  %s", i+1, name)).Checked(s.id == a.activeID)
		if i < 8 {
			item.Shortcut(ui.Cmd, ui.Key1+ui.Key(i))
		} else if i == len(a.sessions)-1 {
			item.Shortcut(ui.Cmd, ui.Key9)
		}
		if item.Chosen() {
			a.selectSession(s.id)
		}
	}
}
