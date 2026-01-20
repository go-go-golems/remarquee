package js

import (
	"github.com/dop251/goja"
)

// NewColorsModule creates the Colors module
func NewColorsModule(vm *goja.Runtime) *goja.Object {
	colors := vm.NewObject()
	
	// Predefined colors
	colors.Set("BLACK", 0)
	colors.Set("GRAY", 1)
	colors.Set("WHITE", 2)
	colors.Set("RED", 3)
	colors.Set("BLUE", 4)
	colors.Set("GREEN", 5)
	colors.Set("YELLOW", 6)
	
	// Color creation functions
	colors.Set("rgb", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewTypeError("rgb requires 3 arguments"))
		}
		
		obj := vm.NewObject()
		obj.Set("r", call.Arguments[0].ToInteger())
		obj.Set("g", call.Arguments[1].ToInteger())
		obj.Set("b", call.Arguments[2].ToInteger())
		obj.Set("a", 255)
		return obj
	})
	
	colors.Set("rgba", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 4 {
			panic(vm.NewTypeError("rgba requires 4 arguments"))
		}
		
		obj := vm.NewObject()
		obj.Set("r", call.Arguments[0].ToInteger())
		obj.Set("g", call.Arguments[1].ToInteger())
		obj.Set("b", call.Arguments[2].ToInteger())
		obj.Set("a", call.Arguments[3].ToInteger())
		return obj
	})
	
	return colors
}

// toolNameToID converts a tool name to its numeric ID
func toolNameToID(name string) uint32 {
	switch name {
	case "pen":
		return 2
	case "pencil":
		return 4
	case "marker":
		return 3
	case "highlighter":
		return 5
	case "eraser":
		return 6
	default:
		return 2 // Default to pen
	}
}

// colorNameToID converts a color name to its numeric ID
func colorNameToID(name string) uint32 {
	switch name {
	case "black":
		return 0
	case "gray":
		return 1
	case "white":
		return 2
	case "red":
		return 3
	case "blue":
		return 4
	case "green":
		return 5
	case "yellow":
		return 6
	default:
		return 0 // Default to black
	}
}
