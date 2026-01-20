package js

import (
	"github.com/dop251/goja"
)

// Stroke represents a low-level stroke object
type Stroke struct {
	vm        *goja.Runtime
	tool      uint32
	color     uint32
	thickness float64
	points    []StrokePoint
	obj       *goja.Object
}

// StrokePoint represents a single point in a stroke
type StrokePoint struct {
	X         float64
	Y         float64
	Pressure  float64
	Speed     float64
	Direction float64
	Width     float64
}

// NewStroke creates a new Stroke
func NewStroke(vm *goja.Runtime) *Stroke {
	return &Stroke{
		vm:        vm,
		tool:      2, // Default to pen
		color:     0, // Default to black
		thickness: 2.0,
		points:    make([]StrokePoint, 0),
	}
}

// ToObject converts the Stroke to a goja.Object with all methods bound
func (s *Stroke) ToObject(this *goja.Object) *goja.Object {
	obj := s.vm.NewObject()
	
	// Store internal reference
	obj.Set("_internal", s.vm.ToValue(s))
	
	// Bind methods
	obj.Set("setTool", s.setTool)
	obj.Set("setColor", s.setColor)
	obj.Set("setThickness", s.setThickness)
	obj.Set("addPoint", s.addPoint)
	obj.Set("addPoints", s.addPoints)
	
	return obj
}

func (s *Stroke) setTool(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(s.vm.NewTypeError("setTool requires 1 argument"))
	}
	s.tool = uint32(call.Arguments[0].ToInteger())
	return call.This
}

func (s *Stroke) setColor(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(s.vm.NewTypeError("setColor requires 1 argument"))
	}
	s.color = uint32(call.Arguments[0].ToInteger())
	return call.This
}

func (s *Stroke) setThickness(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(s.vm.NewTypeError("setThickness requires 1 argument"))
	}
	s.thickness = call.Arguments[0].ToFloat()
	return call.This
}

func (s *Stroke) addPoint(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(s.vm.NewTypeError("addPoint requires 1 argument"))
	}
	
	pointObj := call.Arguments[0].ToObject(s.vm)
	if pointObj == nil {
		panic(s.vm.NewTypeError("addPoint requires an object"))
	}
	
	point := StrokePoint{
		X:         getFloat(pointObj, "x", 0),
		Y:         getFloat(pointObj, "y", 0),
		Pressure:  getFloat(pointObj, "pressure", 128),
		Speed:     getFloat(pointObj, "speed", 10),
		Direction: getFloat(pointObj, "direction", 0),
		Width:     getFloat(pointObj, "width", s.thickness),
	}
	
	s.points = append(s.points, point)
	return call.This
}

func (s *Stroke) addPoints(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(s.vm.NewTypeError("addPoints requires 1 argument"))
	}
	
	pointsVal := call.Arguments[0]
	pointsObj := pointsVal.ToObject(s.vm)
	if pointsObj == nil {
		panic(s.vm.NewTypeError("addPoints requires an array"))
	}
	
	// Get array length
	lengthVal := pointsObj.Get("length")
	if lengthVal == nil {
		panic(s.vm.NewTypeError("addPoints requires an array"))
	}
	length := int(lengthVal.ToInteger())
	
	// Iterate through array
	for i := 0; i < length; i++ {
		pointVal := pointsObj.Get(string(rune(i)))
		if pointVal == nil {
			continue
		}
		pointObj := pointVal.ToObject(s.vm)
		if pointObj == nil {
			continue
		}
		
		point := StrokePoint{
			X:         getFloat(pointObj, "x", 0),
			Y:         getFloat(pointObj, "y", 0),
			Pressure:  getFloat(pointObj, "pressure", 128),
			Speed:     getFloat(pointObj, "speed", 10),
			Direction: getFloat(pointObj, "direction", 0),
			Width:     getFloat(pointObj, "width", s.thickness),
		}
		
		s.points = append(s.points, point)
	}
	
	return call.This
}

func getFloat(obj *goja.Object, key string, defaultVal float64) float64 {
	val := obj.Get(key)
	if val == nil || val == goja.Undefined() {
		return defaultVal
	}
	return val.ToFloat()
}
