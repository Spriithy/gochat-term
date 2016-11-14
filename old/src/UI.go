package old_server

import (
	"fmt"
	"log"
	"sync"

	"github.com/Spriithy/gochat-term/server/src/ui"
	"github.com/jroimartin/gocui"
)

type ServerUI struct {
	g       *gocui.Gui
	in, out *ui.Panel

	sync.RWMutex
	input string
}

func NewServerUI() *ServerUI {
	return new(ServerUI)
}

func (s *ServerUI) Print(a ...interface{}) {
	s.out.Append(a...)
	v, err := s.g.View("out")
	if err != nil {
		panic(err)
	}
	t := s.out.GetBuffer()
	if len(t) > 0 {
		fmt.Fprintf(v, t)
	}
}

func (s *ServerUI) Println(a ...interface{}) {
	s.out.Appendln(a...)
	v, err := s.g.View("out")
	if err != nil {
		panic(err)
	}
	t := s.out.GetBuffer()
	if len(t) > 0 {
		fmt.Fprintf(v, t)
	}
}

func (s *ServerUI) Start() {
	var err error
	s.g, err = gocui.NewGui()

	if err != nil {
		log.Panicln(err)
	}
	defer s.g.Close()

	s.g.Cursor = true

	s.g.BgColor = gocui.ColorDefault
	s.g.FgColor = gocui.ColorWhite

	_, my := s.g.Size()
	s.in = ui.NewPanel("in", "", 2, my-3, 1, 1)
	s.in.SetBordered(false)
	s.in.SetEditable(true)
	s.in.OnUpdate(func(g *gocui.Gui) error {
		if err != nil {
			return err
		}
		if err = g.SetKeybinding("in", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
			s.Lock()
			defer s.Unlock()
			s.input = v.Buffer()
			s.Println(s.input)
			if err := v.SetCursor(0, 0); err != nil {
				return err
			}
			if err := v.SetOrigin(0, 0); err != nil {
				return err
			}
			v.Clear()
			return nil
		}); err != nil {
			return err
		}

		if _, err := g.SetCurrentView("in"); err != nil {
			return err
		}

		return nil
	})

	s.out = ui.NewPanel("out", "", 1, 1, 1, 3)
	s.out.SetTitle("[ " + "Chatroom" + " ]")
	s.out.SetBordered(true)
	s.out.SetEditable(false)
	s.out.SetWrappable(true)
	s.out.SetScrollable(true)
	s.out.OnUpdate(func(g *gocui.Gui) error {
		v, err := g.View("out")
		if err != nil {
			return err
		}
		t := s.out.GetBuffer()
		if len(t) > 0 {
			fmt.Fprintf(v, t)
		}

		return nil
	})

	s.g.SetManager(s.in, s.out)

	if err := s.g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := s.g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}

func (s *ServerUI) FlushInput() string {
	s.RLock()
	defer s.RUnlock()
	t := s.input
	s.input = ""
	return t
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
