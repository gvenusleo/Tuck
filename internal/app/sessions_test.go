package app

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"liuyu.xin/tuck/internal/config"
)

func TestSessionLifecycle(t *testing.T) {
	a := &app{sessions: []*session{{id: 1}, {id: 2}, {id: 3}, {id: 4}}, activeID: 2}
	a.closeSession(1)
	if a.activeID != 2 {
		t.Fatal("closing a background session changed the active session")
	}
	a.closeSession(2)
	if a.activeID != 3 {
		t.Fatal("closing the active session did not select its right neighbor")
	}
	a.selectSession(4)
	a.closeSession(4)
	if a.activeID != 3 {
		t.Fatal("closing the last tab did not select its left neighbor")
	}
	a.sessionExited(2, -1)
	a.sessionTitleChanged(2, "stale title")
	label, _ := a.sessions[0].name(0)
	if len(a.sessions) != 1 || a.sessions[0].id != 3 || label != "终端 1" {
		t.Fatal("stale callbacks changed a surviving session")
	}
	a.sessionExited(3, 0)
	if len(a.sessions) != 0 || a.activeID != 0 || a.focusID != 0 {
		t.Fatal("last session exit left an active session")
	}
}

func TestTabsView(t *testing.T) {
	for _, position := range []string{config.TabsTop, config.TabsBottom} {
		t.Run(position, func(t *testing.T) {
			a := &app{config: config.Default(), activeID: 1, focusID: 1}
			a.config.TabPosition = position
			var peers []net.Conn
			for id := uint64(1); id <= 2; id++ {
				conn, peer := net.Pipe()
				term, err := terminal.New(terminal.Options{Conn: conn})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { term.Close(); peer.Close() })
				a.sessions = append(a.sessions, &session{id: id, term: term})
				peers = append(peers, peer)
			}
			tt := ui.NewTester(a.view, 720, 420)
			tab, _ := tt.Find("终端 1")
			term, _ := tt.Find("当前终端")
			if (tab.Y < term.Y) != (position == config.TabsTop) {
				t.Fatalf("wrong tab position: tab %+v, terminal %+v", tab, term)
			}
			if err := tt.Click("终端 2"); err != nil {
				t.Fatal(err)
			}
			if a.activeID != 2 || !tt.Focused("当前终端") {
				t.Fatal("click did not select and focus the terminal")
			}
			tt.Type("echo")
			peers[1].SetReadDeadline(time.Now().Add(time.Second))
			buf := make([]byte, 4)
			if _, err := io.ReadFull(peers[1], buf); err != nil || string(buf) != "echo" {
				t.Fatalf("input did not reach the selected session: %q, %v", buf, err)
			}
			tt.Key(ui.Cmd, ui.KeyW)
			if len(a.sessions) != 1 || a.activeID != 1 || !tt.Focused("当前终端") {
				t.Fatal("closing the active tab lost the surviving session or its focus")
			}
			if _, visible := tt.Find("终端标签页"); visible {
				t.Fatal("single session still showed the tab bar")
			}
			content, found := tt.Find("当前终端")
			if !found || content.Y != 0 || content.H != 420 {
				t.Fatal("single session did not fill the panel")
			}
			tt.Key(ui.Cmd, ui.KeyW)
			if len(a.sessions) != 0 || a.activeID != 0 {
				t.Fatal("close shortcut did not remove the last session")
			}
		})
	}
}
