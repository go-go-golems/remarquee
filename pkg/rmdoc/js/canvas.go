package js

import (
	"math"
	
	"github.com/dop251/goja"
)

// Canvas provides a canvas-like drawing interface
type Canvas struct {
	vm    *goja.Runtime
	page  *Page
	obj   *goja.Object
	
	// Current pen state
	tool      uint32
	color     uint32
	thickness float64
	
	// Current path
	currentPath []Point
	pathStarted bool
}

// Point represents a point in the current path
type Point struct {
	X         float64
	Y         float64
	Pressure  float64
	Speed     float64
	Direction float64
	Width     float64
}

// NewCanvas creates a new Canvas
func NewCanvas(vm *goja.Runtime, page *Page) *Canvas {
	return &Canvas{
		vm:          vm,
		page:        page,
		tool:        2, // Pen
		color:       0, // Black
		thickness:   2.0,
		currentPath: make([]Point, 0),
		pathStarted: false,
	}
}

// ToValue converts the Canvas to a goja.Value
func (c *Canvas) ToValue() goja.Value {
	if c.obj == nil {
		c.obj = c.vm.NewObject()
		
		// Pen configuration
		c.obj.Set("setPen", c.setPen)
		c.obj.Set("setTool", c.setTool)
		c.obj.Set("setColor", c.setColor)
		c.obj.Set("setThickness", c.setThickness)
		
		// Path building
		c.obj.Set("beginPath", c.beginPath)
		c.obj.Set("moveTo", c.moveTo)
		c.obj.Set("lineTo", c.lineTo)
		c.obj.Set("closePath", c.closePath)
		c.obj.Set("stroke", c.stroke)
		
		// Drawing primitives
		c.obj.Set("drawLine", c.drawLine)
		c.obj.Set("drawRect", c.drawRect)
		c.obj.Set("drawCircle", c.drawCircle)
	}
	return c.obj
}

func (c *Canvas) setPen(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(c.vm.NewTypeError("setPen requires 1 argument"))
	}
	
	options := call.Arguments[0].ToObject(c.vm)
	if options == nil {
		panic(c.vm.NewTypeError("setPen requires an object"))
	}
	
	if tool := options.Get("tool"); tool != nil && tool != goja.Undefined() {
		c.tool = toolNameToID(tool.String())
	}
	if color := options.Get("color"); color != nil && color != goja.Undefined() {
		c.color = colorNameToID(color.String())
	}
	if thickness := options.Get("thickness"); thickness != nil && thickness != goja.Undefined() {
		c.thickness = thickness.ToFloat()
	}
	
	return call.This
}

func (c *Canvas) setTool(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(c.vm.NewTypeError("setTool requires 1 argument"))
	}
	c.tool = toolNameToID(call.Arguments[0].String())
	return call.This
}

func (c *Canvas) setColor(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(c.vm.NewTypeError("setColor requires 1 argument"))
	}
	
	arg := call.Arguments[0]
	if arg.ExportType().Kind().String() == "string" {
		c.color = colorNameToID(arg.String())
	} else {
		// Assume it's a color object
		obj := arg.ToObject(c.vm)
		if obj != nil {
			// For now, just use black
			c.color = 0
		}
	}
	return call.This
}

func (c *Canvas) setThickness(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(c.vm.NewTypeError("setThickness requires 1 argument"))
	}
	c.thickness = call.Arguments[0].ToFloat()
	return call.This
}

func (c *Canvas) beginPath(call goja.FunctionCall) goja.Value {
	c.currentPath = make([]Point, 0)
	c.pathStarted = true
	return goja.Undefined()
}

func (c *Canvas) moveTo(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(c.vm.NewTypeError("moveTo requires 2 arguments"))
	}
	
	x := call.Arguments[0].ToFloat()
	y := call.Arguments[1].ToFloat()
	
	// Start a new sub-path
	c.currentPath = append(c.currentPath, Point{
		X:        x,
		Y:        y,
		Pressure: 128,
		Width:    c.thickness,
	})
	
	return goja.Undefined()
}

func (c *Canvas) lineTo(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(c.vm.NewTypeError("lineTo requires 2 arguments"))
	}
	
	x := call.Arguments[0].ToFloat()
	y := call.Arguments[1].ToFloat()
	
	if len(c.currentPath) == 0 {
		// If no current point, treat as moveTo
		c.currentPath = append(c.currentPath, Point{
			X:        x,
			Y:        y,
			Pressure: 128,
			Width:    c.thickness,
		})
	} else {
		// Add line from last point to new point
		lastPoint := c.currentPath[len(c.currentPath)-1]
		
		// Interpolate points for smoother line
		dx := x - lastPoint.X
		dy := y - lastPoint.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		steps := int(distance / 5.0) // One point every 5 pixels
		if steps < 1 {
			steps = 1
		}
		
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			c.currentPath = append(c.currentPath, Point{
				X:        lastPoint.X + dx*t,
				Y:        lastPoint.Y + dy*t,
				Pressure: 128,
				Width:    c.thickness,
			})
		}
	}
	
	return goja.Undefined()
}

func (c *Canvas) closePath(call goja.FunctionCall) goja.Value {
	if len(c.currentPath) > 0 {
		// Draw line back to start
		firstPoint := c.currentPath[0]
		c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(firstPoint.X), c.vm.ToValue(firstPoint.Y)}})
	}
	return goja.Undefined()
}

func (c *Canvas) stroke(call goja.FunctionCall) goja.Value {
	if len(c.currentPath) == 0 {
		return goja.Undefined()
	}
	
	// Create a stroke from the current path
	stroke := NewStroke(c.vm)
	stroke.tool = c.tool
	stroke.color = c.color
	stroke.thickness = c.thickness
	
	for _, pt := range c.currentPath {
		stroke.points = append(stroke.points, StrokePoint{
			X:        pt.X,
			Y:        pt.Y,
			Pressure: pt.Pressure,
			Speed:    10,
			Width:    pt.Width,
		})
	}
	
	// Add stroke to page
	c.page.AddStrokeInternal(stroke)
	
	// Clear the path
	c.currentPath = make([]Point, 0)
	
	return goja.Undefined()
}

func (c *Canvas) drawLine(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 4 {
		panic(c.vm.NewTypeError("drawLine requires 4 arguments"))
	}
	
	x1 := call.Arguments[0].ToFloat()
	y1 := call.Arguments[1].ToFloat()
	x2 := call.Arguments[2].ToFloat()
	y2 := call.Arguments[3].ToFloat()
	
	c.beginPath(goja.FunctionCall{})
	c.moveTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x1), c.vm.ToValue(y1)}})
	c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x2), c.vm.ToValue(y2)}})
	c.stroke(goja.FunctionCall{})
	
	return goja.Undefined()
}

func (c *Canvas) drawRect(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 4 {
		panic(c.vm.NewTypeError("drawRect requires 4 arguments"))
	}
	
	x := call.Arguments[0].ToFloat()
	y := call.Arguments[1].ToFloat()
	width := call.Arguments[2].ToFloat()
	height := call.Arguments[3].ToFloat()
	
	c.beginPath(goja.FunctionCall{})
	c.moveTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x), c.vm.ToValue(y)}})
	c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x + width), c.vm.ToValue(y)}})
	c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x + width), c.vm.ToValue(y + height)}})
	c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x), c.vm.ToValue(y + height)}})
	c.closePath(goja.FunctionCall{})
	c.stroke(goja.FunctionCall{})
	
	return goja.Undefined()
}

func (c *Canvas) drawCircle(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 3 {
		panic(c.vm.NewTypeError("drawCircle requires 3 arguments"))
	}
	
	cx := call.Arguments[0].ToFloat()
	cy := call.Arguments[1].ToFloat()
	radius := call.Arguments[2].ToFloat()
	
	c.beginPath(goja.FunctionCall{})
	
	// Draw circle using line segments
	segments := 64
	for i := 0; i <= segments; i++ {
		angle := 2.0 * math.Pi * float64(i) / float64(segments)
		x := cx + radius*math.Cos(angle)
		y := cy + radius*math.Sin(angle)
		
		if i == 0 {
			c.moveTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x), c.vm.ToValue(y)}})
		} else {
			c.lineTo(goja.FunctionCall{Arguments: []goja.Value{c.vm.ToValue(x), c.vm.ToValue(y)}})
		}
	}
	
	c.stroke(goja.FunctionCall{})
	
	return goja.Undefined()
}
