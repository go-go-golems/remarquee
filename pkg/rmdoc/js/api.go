package js

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

// NewRuntime creates a new goja runtime with the Remarquee.js API exposed
func NewRuntime() *goja.Runtime {
	vm := goja.New()
	
	// Set field name mapper for natural JavaScript naming
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	
	// Expose the RMDoc constructor
	vm.Set("RMDoc", func(call goja.ConstructorCall) *goja.Object {
		doc := NewDocument(vm)
		return doc.ToObject(call.This)
	})
	
	// Expose the Stroke constructor
	vm.Set("Stroke", func(call goja.ConstructorCall) *goja.Object {
		stroke := NewStroke(vm)
		return stroke.ToObject(call.This)
	})
	
	// Expose the Colors module
	vm.Set("Colors", NewColorsModule(vm))
	
	// Expose console.log for debugging
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			println(arg.String())
		}
		return goja.Undefined()
	})
	vm.Set("console", console)
	
	return vm
}

// RunScript executes a JavaScript file in the runtime
func RunScript(vm *goja.Runtime, script string) (goja.Value, error) {
	val, err := vm.RunString(script)
	if err != nil {
		if jsErr, ok := err.(*goja.Exception); ok {
			return nil, errors.Errorf("JavaScript error: %s", jsErr.String())
		}
		return nil, errors.Wrap(err, "failed to run script")
	}
	return val, nil
}
