package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Pick struct {
	Pick func(sizeAvail image.Point) UI `json:"-"`

	ui UI
}

func (ui *Pick) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout("Pick", self)

	if self.Layout == Clean && !force {
		return
	}

	oui := ui.ui
	ui.ui = ui.Pick(sizeAvail)
	ui.ui.Layout(dui, self, sizeAvail, force || oui != ui.ui)
}

func (ui *Pick) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw("Pick", self)
	ui.ui.Draw(dui, self, img, orig, m, force)
}

func (ui *Pick) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	return ui.ui.Mouse(dui, self, m, origM, orig)
}

func (ui *Pick) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	return ui.ui.Key(dui, self, k, m, orig)
}

func (ui *Pick) FirstFocus(dui *DUI, self *Kid) (warp *image.Point) {
	return ui.ui.FirstFocus(dui, self)
}

func (ui *Pick) Focus(dui *DUI, self *Kid, o UI) (warp *image.Point) {
	return ui.ui.Focus(dui, self, o)
}

func (ui *Pick) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return ui.ui.Mark(self, o, forLayout)
}

func (ui *Pick) Print(self *Kid, indent int) {
	PrintUI("Pick", self, indent)
	if ui.ui != nil {
		ui.ui.Print(self, indent+1)
	}
}
