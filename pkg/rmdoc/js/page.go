package js

import (
	"github.com/dop251/goja"
	"github.com/google/uuid"
)

// Page represents a single page in an RMDoc document
type Page struct {
	vm       *goja.Runtime
	doc      *Document
	pageID   string
	index    int
	template string
	canvas   *Canvas
	strokes  []*Stroke
	obj      *goja.Object
}

// NewPage creates a new Page
func NewPage(vm *goja.Runtime, doc *Document, index int) *Page {
	p := &Page{
		vm:       vm,
		doc:      doc,
		pageID:   uuid.New().String(),
		index:    index,
		template: "Blank",
		strokes:  make([]*Stroke, 0),
	}
	p.canvas = NewCanvas(vm, p)
	return p
}

// ToValue converts the Page to a goja.Value
func (p *Page) ToValue() goja.Value {
	if p.obj == nil {
		p.obj = p.vm.NewObject()
		p.obj.Set("setTemplate", p.setTemplate)
		p.obj.Set("getCanvas", p.getCanvas)
		p.obj.Set("addStroke", p.addStroke)
	}
	return p.obj
}

func (p *Page) setTemplate(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(p.vm.NewTypeError("setTemplate requires 1 argument"))
	}
	p.template = call.Arguments[0].String()
	return call.This
}

func (p *Page) getCanvas(call goja.FunctionCall) goja.Value {
	return p.canvas.ToValue()
}

func (p *Page) addStroke(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(p.vm.NewTypeError("addStroke requires 1 argument"))
	}
	
	// Extract the Stroke object from the argument
	strokeObj := call.Arguments[0].ToObject(p.vm)
	if strokeObj == nil {
		panic(p.vm.NewTypeError("addStroke requires a Stroke object"))
	}
	
	// Get the internal stroke reference
	strokeVal := strokeObj.Get("_internal")
	if strokeVal == nil {
		panic(p.vm.NewTypeError("Invalid Stroke object"))
	}
	
	stroke, ok := strokeVal.Export().(*Stroke)
	if !ok {
		panic(p.vm.NewTypeError("Invalid Stroke object"))
	}
	
	p.strokes = append(p.strokes, stroke)
	return call.This
}

// AddStrokeInternal adds a stroke directly (called from Canvas)
func (p *Page) AddStrokeInternal(stroke *Stroke) {
	p.strokes = append(p.strokes, stroke)
}
