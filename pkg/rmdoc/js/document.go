package js

import (
	"github.com/dop251/goja"
	"github.com/google/uuid"
)

// Document represents an RMDoc document in JavaScript
type Document struct {
	vm       *goja.Runtime
	uuid     string
	title    string
	docType  string
	pages    []*Page
}

// NewDocument creates a new Document
func NewDocument(vm *goja.Runtime) *Document {
	return &Document{
		vm:      vm,
		uuid:    uuid.New().String(),
		title:   "Untitled",
		docType: "notebook",
		pages:   make([]*Page, 0),
	}
}

// ToObject converts the Document to a goja.Object with all methods bound
func (d *Document) ToObject(this *goja.Object) *goja.Object {
	obj := d.vm.NewObject()
	
	// Bind methods
	obj.Set("setTitle", d.setTitle)
	obj.Set("setType", d.setType)
	obj.Set("addPage", d.addPage)
	obj.Set("getPage", d.getPage)
	obj.Set("getPages", d.getPages)
	obj.Set("getPageCount", d.getPageCount)
	obj.Set("save", d.save)
	obj.Set("toBytes", d.toBytes)
	
	return obj
}

func (d *Document) setTitle(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.vm.NewTypeError("setTitle requires 1 argument"))
	}
	d.title = call.Arguments[0].String()
	return call.This
}

func (d *Document) setType(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.vm.NewTypeError("setType requires 1 argument"))
	}
	docType := call.Arguments[0].String()
	if docType != "notebook" && docType != "pdf" && docType != "epub" {
		panic(d.vm.NewTypeError("Invalid document type. Must be 'notebook', 'pdf', or 'epub'"))
	}
	d.docType = docType
	return call.This
}

func (d *Document) addPage(call goja.FunctionCall) goja.Value {
	page := NewPage(d.vm, d, len(d.pages))
	d.pages = append(d.pages, page)
	return page.ToValue()
}

func (d *Document) getPage(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.vm.NewTypeError("getPage requires 1 argument"))
	}
	index := int(call.Arguments[0].ToInteger())
	if index < 0 || index >= len(d.pages) {
		panic(d.vm.NewTypeError("Page index out of bounds"))
	}
	return d.pages[index].ToValue()
}

func (d *Document) getPages(call goja.FunctionCall) goja.Value {
	pages := make([]interface{}, len(d.pages))
	for i, page := range d.pages {
		pages[i] = page.ToValue()
	}
	return d.vm.ToValue(pages)
}

func (d *Document) getPageCount(call goja.FunctionCall) goja.Value {
	return d.vm.ToValue(len(d.pages))
}

func (d *Document) save(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(d.vm.NewTypeError("save requires 1 argument"))
	}
	path := call.Arguments[0].String()
	
	// Build and save the document
	err := BuildAndSave(d, path)
	if err != nil {
		panic(d.vm.NewGoError(err))
	}
	
	return goja.Undefined()
}

func (d *Document) toBytes(call goja.FunctionCall) goja.Value {
	// Build the document to bytes
	bytes, err := BuildToBytes(d)
	if err != nil {
		panic(d.vm.NewGoError(err))
	}
	
	// Convert to ArrayBuffer
	ab := d.vm.NewArrayBuffer(bytes)
	return d.vm.ToValue(ab)
}
