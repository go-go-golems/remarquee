package compile

import "github.com/go-go-golems/remarquee/pkg/rmdoc"

type CompiledDoc struct {
	UUID  string
	Name  string
	Pages []CompiledPage
}

type CompiledPage struct {
	ID       string
	Template string
	CanvasW  int
	CanvasH  int
	Layers   []CompiledLayer
	Text     *CompiledText
}

type CompiledLayer struct {
	Name  string
	Items []CompiledItem
}

type CompiledItemKind uint8

const (
	CompiledItemStroke CompiledItemKind = iota
	CompiledItemGlyph
)

type CompiledItem struct {
	Kind   CompiledItemKind
	Stroke *rmdoc.Stroke
	Glyph  *CompiledGlyph
}

type CompiledGlyph struct {
	Start  *uint32
	Length *uint32
	Text   string
	Color  rmdoc.PenColor
	Rects  []rmdoc.RMV6Rectangle
}

type CompiledText struct {
	Text  string
	Style uint8
	PosX  float64
	PosY  float64
	Width float64
}
