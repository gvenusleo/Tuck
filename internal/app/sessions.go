package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
)

type session struct {
	id    uint64
	term  *terminal.Terminal
	title string
}

const (
	// terminalPaddingX 与终端插件的默认横向内边距一致；
	// terminalPaddingY 为负值，表示不留上下内边距。
	terminalPaddingX = 6
	terminalPaddingY = -1
)

// name 返回标签的显示名与悬停提示：目录末段（家目录为 ~）与完整路径，
// 没有目录时用终端标题，再回退到按位置编号的名称。
func (s *session) name(index int) (label, tooltip string) {
	if s.term != nil {
		if dir := s.term.Dir(); dir != "" {
			label = filepath.Base(dir)
			if home, err := os.UserHomeDir(); err == nil && filepath.Clean(dir) == filepath.Clean(home) {
				label = "~"
			}
			return label, dir
		}
	}
	if s.title != "" {
		return s.title, s.title
	}
	label = fmt.Sprintf("终端 %d", index+1)
	return label, label
}

func (a *app) sessionIndex(id uint64) int {
	return slices.IndexFunc(a.sessions, func(s *session) bool { return s.id == id })
}

func (a *app) newSession() {
	if a.quitting {
		return
	}
	// 会话 ID 同时用作界面 Key，必须高于现有会话，避免重复。
	for _, s := range a.sessions {
		a.nextSessionID = max(a.nextSessionID, s.id)
	}
	a.nextSessionID++
	s := &session{id: a.nextSessionID}
	term, err := terminal.New(terminal.Options{
		// 纵向不留白，文字紧贴标签栏；横向与插件默认值一致，和标签文字左对齐。
		Padding: terminal.Padding{X: terminalPaddingX, Y: terminalPaddingY},
		Font:    terminal.Font{Family: a.config.Font, Size: float32(a.config.FontSize)},
		Theme:   a.theme,
		OnTitle: func(title string) {
			go mygo.RunOnMain(func() { a.sessionTitleChanged(s.id, title) })
		},
		OnExit: func(code int) {
			// Close 会等待终端回调返回，因此回调不能等待主线程。
			go mygo.RunOnMain(func() { a.sessionExited(s.id, code) })
		},
	})
	if err != nil {
		sessionError(err)
		return
	}
	s.term = term
	a.sessions = append(a.sessions, s)
	a.selectSession(s.id)
}

func (a *app) selectSession(id uint64) {
	if a.quitting || a.sessionIndex(id) < 0 {
		return
	}
	a.activeID, a.focusID = id, id
	a.revealTab = true
	a.invalidatePanel()
}

func (a *app) stepSession(delta int) {
	if n := len(a.sessions); n > 0 {
		i := (a.sessionIndex(a.activeID) + delta + n) % n
		a.selectSession(a.sessions[i].id)
	}
}

func (a *app) closeSession(id uint64) {
	i := a.sessionIndex(id)
	if i < 0 {
		return
	}
	s := a.sessions[i]
	// 先移除身份，忽略 Close 引发或已经排队的退出回调。
	a.sessions = slices.Delete(a.sessions, i, i+1)
	if len(a.sessions) == 0 {
		a.activeID, a.focusID = 0, 0
		a.hide()
	} else if a.activeID == id {
		a.selectSession(a.sessions[min(i, len(a.sessions)-1)].id)
	}
	if s.term != nil {
		if err := s.term.Close(); err != nil {
			log.Printf("关闭终端失败：%v", err)
		}
	}
	a.revealTab = true
	a.invalidatePanel()
}

func (a *app) sessionTitleChanged(id uint64, title string) {
	if i := a.sessionIndex(id); !a.quitting && i >= 0 {
		a.sessions[i].title = strings.Join(strings.Fields(title), " ")
		a.revealTab = true
		a.invalidatePanel()
	}
}

func (a *app) sessionExited(id uint64, code int) {
	i := a.sessionIndex(id)
	if a.quitting || i < 0 {
		return
	}
	var err error
	if code < 0 {
		err = errors.New(a.sessions[i].term.Text())
	}
	a.closeSession(id)
	if err != nil {
		sessionError(err)
	}
}

func sessionError(err error) {
	showError("无法启动终端", err, "好")
}

func (a *app) invalidatePanel() {
	if a.panel != nil {
		a.panel.Invalidate()
	}
}
