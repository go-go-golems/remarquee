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
}

type CompiledLayer struct {
	Name    string
	Strokes []rmdoc.Stroke
}
