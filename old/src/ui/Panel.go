package old_ui

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

type UpdateFunc func(*gocui.Gui) error

func doUpdate(g *gocui.Gui) error {
	return nil
}

type Panel struct {
	name  string
	title string

	x, y int
	z, t int

	scroll bool
	border bool
	edit   bool
	wrap   bool

	bg gocui.Attribute
	fg gocui.Attribute

	update UpdateFunc

	buf string
}

func NewPanel(name, title string, x, y, z, t int) *Panel {
	return &Panel{
		name, title,
		x, y, z, t,
		false, true, false, false,
		gocui.ColorDefault,
		gocui.ColorDefault,
		doUpdate,
		""}
}

func (p *Panel) Layout(g *gocui.Gui) error {
	mx, my := g.Size()
	v, err := g.SetView(p.name, 0+p.x, 0+p.y, mx-p.z, my-p.t)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = p.title

		v.Autoscroll = p.scroll
		v.Frame = p.border
		v.Editable = p.edit
		v.Wrap = p.wrap

		v.BgColor = p.bg
		v.FgColor = p.fg

		err = p.update(g)
		if err != nil {
			return err
		}

		if v.Editable {
			if _, err := g.SetCurrentView(p.name); err != nil {
				return err
			}
		}

		fmt.Fprint(v, p.buf)
		p.ClearBuffer()
	}
	return nil
}

func (p *Panel) OnUpdate(u UpdateFunc) {
	p.update = u
}

func (p *Panel) SetTitle(s string) {
	p.title = s
}

func (p *Panel) SetScrollable(b bool) {
	p.scroll = b
}

func (p *Panel) SetBordered(b bool) {
	p.border = b
}

func (p *Panel) SetEditable(b bool) {
	p.edit = b
}

func (p *Panel) SetWrappable(b bool) {
	p.wrap = b
}

func (p *Panel) SetBackgroundColor(col gocui.Attribute) {
	p.bg = col
}

func (p *Panel) SetForegroundColor(col gocui.Attribute) {
	p.fg = col
}

func (p *Panel) ClearBuffer() {
	p.buf = ""
}

func (p *Panel) GetBuffer() string {
	defer p.ClearBuffer()
	return p.buf
}

func (p *Panel) Append(a ...interface{}) {
	var buf string
	for _, x := range a {
		buf = fmt.Sprintf("%s%s", buf, x)
	}
	p.buf += buf
}

func (p *Panel) Appendln(a ...interface{}) {
	var buf string
	for _, x := range a {
		buf = fmt.Sprintf("%s %s", buf, x)
	}
	p.buf += buf + "\n"
}
