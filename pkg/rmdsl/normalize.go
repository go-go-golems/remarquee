package rmdsl

import (
	"fmt"
)

const (
	DSLVersionV0    = "v0"
	CanvasSpaceV6   = "rm_screen_v6"
	DefaultWidthV6  = 1404
	DefaultHeightV6 = 1872
)

// Normalize fills defaults and performs basic validation.
// It is intentionally strict about structure; it is intentionally lax about enums (tool/color),
// because those will evolve as we add renderer features.
func Normalize(d *Doc) error {
	if d == nil {
		return fmt.Errorf("doc is nil")
	}
	if d.RMDSL == "" {
		d.RMDSL = DSLVersionV0
	}
	if d.RMDSL != DSLVersionV0 {
		return fmt.Errorf("unsupported rm_dsl %q (supported: %q)", d.RMDSL, DSLVersionV0)
	}
	if d.Document.Name == "" {
		return fmt.Errorf("document.name is required")
	}
	if len(d.Document.Pages) == 0 {
		return fmt.Errorf("document.pages is empty")
	}
	for pi := range d.Document.Pages {
		p := &d.Document.Pages[pi]
		if p.ID == "" {
			p.ID = fmt.Sprintf("page-%d", pi+1)
		}
		if p.Canvas.Space == "" {
			p.Canvas.Space = CanvasSpaceV6
		}
		if p.Canvas.Space != CanvasSpaceV6 {
			return fmt.Errorf("document.pages[%d].canvas.space=%q unsupported (supported: %q)", pi, p.Canvas.Space, CanvasSpaceV6)
		}
		if p.Canvas.Width == 0 {
			p.Canvas.Width = DefaultWidthV6
		}
		if p.Canvas.Height == 0 {
			p.Canvas.Height = DefaultHeightV6
		}
	}
	return nil
}
