package rmdsl

// NOTE: This is intentionally small and corresponds to RMDoc-DSL v0 as defined in:
// ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/reference/06-yaml-dsl-rmdoc-spec-and-design.md

type Doc struct {
	RMDSL    string   `yaml:"rm_dsl" json:"rm_dsl"`
	Document Document `yaml:"document" json:"document"`
}

type Document struct {
	Name   string `yaml:"name" json:"name"`
	Kind   string `yaml:"kind" json:"kind"`
	Format string `yaml:"format" json:"format"`
	Pages  []Page `yaml:"pages" json:"pages"`
}

type Page struct {
	ID       string  `yaml:"id" json:"id"`
	Template string  `yaml:"template,omitempty" json:"template,omitempty"`
	Canvas   Canvas  `yaml:"canvas" json:"canvas"`
	Layers   []Layer `yaml:"layers" json:"layers"`
}

type Canvas struct {
	Space  string `yaml:"space" json:"space"`
	Width  int    `yaml:"width" json:"width"`
	Height int    `yaml:"height" json:"height"`
}

type Layer struct {
	Name  string `yaml:"name" json:"name"`
	Items []Item `yaml:"items" json:"items"`
}

type Item struct {
	Kind string `yaml:"kind" json:"kind"`

	// stroke
	Stroke *StrokeStyle `yaml:"stroke,omitempty" json:"stroke,omitempty"`
	Points []Point      `yaml:"points,omitempty" json:"points,omitempty"`

	// shape
	Shape     string   `yaml:"shape,omitempty" json:"shape,omitempty"` // rect|ellipse|line
	Center    *PointXY `yaml:"center,omitempty" json:"center,omitempty"`
	Rx        float64  `yaml:"rx,omitempty" json:"rx,omitempty"`
	Ry        float64  `yaml:"ry,omitempty" json:"ry,omitempty"`
	Rect      *Rect    `yaml:"rect,omitempty" json:"rect,omitempty"`
	RotateDeg float64  `yaml:"rotate_deg,omitempty" json:"rotate_deg,omitempty"`
}

type StrokeStyle struct {
	Tool  string  `yaml:"tool" json:"tool"`
	Color string  `yaml:"color" json:"color"`
	Width float64 `yaml:"width" json:"width"`
}

type Point struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

type PointXY struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

type Rect struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
	W float64 `yaml:"w" json:"w"`
	H float64 `yaml:"h" json:"h"`
}


