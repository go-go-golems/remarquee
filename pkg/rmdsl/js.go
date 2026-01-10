package rmdsl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

// rmPreludeJS implements the minimal "rm" builder API as pure JS, returning plain objects.
// Keeping this in JS (instead of Go-exposed method chains) makes it cheap to evolve.
//
// IMPORTANT: This is not meant to be a general-purpose Node runtime.
// We only expose what fixture generation needs.
const rmPreludeJS = `
// Minimal RMDoc-DSL builder prelude.
// The user's script is expected to define one of:
//   - function main(params) { ...; return <dsl object>; }
//   - exports.default = function(params) { ... }
//   - module.exports = function(params) { ... }
//
// This prelude defines ` + "`rm`" + ` and some CommonJS-ish globals.

var exports = exports || {};
var module = module || { exports: exports };

(function(global){
  function deepClone(x) { return JSON.parse(JSON.stringify(x)); }

  function DocBuilder(name) {
    this._doc = { rm_dsl: "v0", document: { name: name, kind: "notebook", format: "v6", pages: [] } };
    this._curPage = null;
    this._curLayer = null;
    this._curItem = null;
  }

  DocBuilder.prototype.notebook = function(){ this._doc.document.kind = "notebook"; return this; };
  DocBuilder.prototype.pdf = function(){ this._doc.document.kind = "pdf"; return this; };
  DocBuilder.prototype.mixed = function(){ this._doc.document.kind = "mixed"; return this; };
  DocBuilder.prototype.v6 = function(){ this._doc.document.format = "v6"; return this; };
  DocBuilder.prototype.v5 = function(){ this._doc.document.format = "v5"; return this; };
  DocBuilder.prototype.v3 = function(){ this._doc.document.format = "v3"; return this; };

  DocBuilder.prototype.page = function(id){
    var p = { id: id || "", canvas: { space: "rm_screen_v6", width: 1404, height: 1872 }, layers: [] };
    this._doc.document.pages.push(p);
    this._curPage = p;
    this._curLayer = null;
    return this;
  };

  DocBuilder.prototype.canvas = function(space, w, h){
    if (!this._curPage) throw new Error("rm.doc(...).page(...).canvas(...) requires an active page");
    this._curPage.canvas = { space: space || "rm_screen_v6", width: w || 1404, height: h || 1872 };
    return this;
  };

  DocBuilder.prototype.layer = function(name){
    if (!this._curPage) throw new Error("rm.doc(...).page(...).layer(...) requires an active page");
    var l = { name: name || "", items: [] };
    this._curPage.layers.push(l);
    this._curLayer = l;
    return this;
  };

  DocBuilder.prototype._pushItem = function(item){
    if (!this._curLayer) throw new Error("layer(...) must be called before adding items");
    this._curLayer.items.push(item);
    this._curItem = item;
    return this;
  };

  DocBuilder.prototype.stroke = function(tool, color, width){
    var it = { kind: "stroke", stroke: { tool: tool, color: color, width: width || 1 }, points: [] };
    this._pushItem(it);
    // return a small builder facade for stroke-specific operations
    var self = this;
    return {
      polyline: function(points){
        it.points = deepClone(points || []);
        return self;
      }
    };
  };

  DocBuilder.prototype.ellipse = function(center, rx, ry){
    var it = { kind: "shape", shape: "ellipse", center: deepClone(center), rx: rx, ry: ry, stroke: { tool: "fineliner_2", color: "black", width: 1 } };
    this._pushItem(it);
    return this._shapeFacade(it);
  };

  DocBuilder.prototype.rect = function(rect){
    var it = { kind: "shape", shape: "rect", rect: deepClone(rect), rotate_deg: 0, stroke: { tool: "fineliner_2", color: "black", width: 1 } };
    this._pushItem(it);
    return this._shapeFacade(it);
  };

  DocBuilder.prototype._shapeFacade = function(it){
    var self = this;
    return {
      rotateDeg: function(deg){ it.rotate_deg = deg || 0; return this; },
      stroke: function(tool, color, width){ it.stroke = { tool: tool, color: color, width: width || 1 }; return self; }
    };
  };

  DocBuilder.prototype.done = function(){
    return deepClone(this._doc);
  };

  global.rm = global.rm || {};
  global.rm.doc = function(name){ return new DocBuilder(name); };

  // enums (string-valued for now)
  global.rm.space = { rm_screen_v6: "rm_screen_v6" };
  global.rm.tool = {
    fineliner_2: "fineliner_2",
    highlighter_2: "highlighter_2",
    marker_2: "marker_2",
    ballpoint_2: "ballpoint_2",
    pencil_2: "pencil_2"
  };
  global.rm.color = {
    black: "black",
    red: "red",
    green: "green",
    blue: "blue",
    highlight_pink: "highlight_pink",
    highlight_green: "highlight_green"
  };

  // include() is bound from Go (rm.include(path)).
})(this);
`

func loadJS(ctx context.Context, path string, opts LoadOptions) (*Doc, error) {
	entryAbs, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "abs path")
	}

	caseRoot := opts.CaseRoot
	if caseRoot == "" {
		caseRoot = filepath.Dir(entryAbs)
	}
	caseRootAbs, err := filepath.Abs(caseRoot)
	if err != nil {
		return nil, errors.Wrap(err, "abs caseRoot")
	}

	vm := goja.New()

	// Deadline/interrupt.
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		<-ctx.Done()
		// Interrupt with a recognizable error.
		vm.Interrupt(errors.Wrap(ctx.Err(), "js execution interrupted"))
	}()

	// We keep an include stack so relative includes resolve correctly.
	includeStack := []string{filepath.Dir(entryAbs)}

	rmObj := vm.NewObject()
	_ = rmObj.Set("include", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("rm.include(path) requires a path"))
		}
		rel := call.Arguments[0].String()
		base := includeStack[len(includeStack)-1]
		inc := rel
		if !filepath.IsAbs(inc) {
			inc = filepath.Join(base, inc)
		}
		incAbs, _ := filepath.Abs(inc)

		// Restrict to caseRoot.
		// Note: filepath.Rel is the simplest safe check.
		relToRoot, err := filepath.Rel(caseRootAbs, incAbs)
		if err != nil || strings.HasPrefix(relToRoot, "..") || relToRoot == "." && incAbs == caseRootAbs {
			// Allow exact root file? No, that's weird. Be strict.
			panic(vm.NewGoError(errors.Errorf("rm.include(%q): path outside caseRoot %q", incAbs, caseRootAbs)))
		}

		b, err := os.ReadFile(incAbs)
		if err != nil {
			panic(vm.NewGoError(errors.Wrapf(err, "rm.include read %s", incAbs)))
		}

		includeStack = append(includeStack, filepath.Dir(incAbs))
		defer func() { includeStack = includeStack[:len(includeStack)-1] }()

		_, err = vm.RunString(string(b))
		if err != nil {
			panic(vm.NewGoError(errors.Wrapf(err, "rm.include exec %s", incAbs)))
		}
		return goja.Undefined()
	})

	// Provide rm early so the prelude can attach to it and scripts can call rm.include().
	if err := vm.Set("rm", rmObj); err != nil {
		return nil, errors.Wrap(err, "set rm")
	}

	// Prelude: builder + CommonJS-ish exports/module.
	if _, err := vm.RunString(rmPreludeJS); err != nil {
		return nil, errors.Wrap(err, "run rm prelude")
	}

	// Execute entry script.
	entryBytes, err := os.ReadFile(entryAbs)
	if err != nil {
		return nil, errors.Wrap(err, "read js")
	}

	// A tiny guardrail: if a script loops forever without checking time, the ctx interrupt still stops it,
	// but we can also set a default timeout for safety if the caller doesn't.
	if _, ok := ctx.Deadline(); !ok {
		_ = time.AfterFunc(3*time.Second, func() {
			vm.Interrupt(errors.New("js execution timeout (no ctx deadline set)"))
		})
	}

	if _, err := vm.RunString(string(entryBytes)); err != nil {
		return nil, errors.Wrap(err, "exec js")
	}

	// Find the entry point function.
	var fn goja.Callable
	if v := vm.Get("main"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if c, ok := goja.AssertFunction(v); ok {
			fn = c
		}
	}
	if fn == nil {
		// exports.default?
		if ex := vm.Get("exports"); ex != nil && !goja.IsUndefined(ex) && !goja.IsNull(ex) {
			if o := ex.ToObject(vm); o != nil {
				if dv := o.Get("default"); dv != nil && !goja.IsUndefined(dv) && !goja.IsNull(dv) {
					if c, ok := goja.AssertFunction(dv); ok {
						fn = c
					}
				}
			}
		}
	}
	if fn == nil {
		// module.exports?
		if mv := vm.Get("module"); mv != nil && !goja.IsUndefined(mv) && !goja.IsNull(mv) {
			if mo := mv.ToObject(vm); mo != nil {
				if ev := mo.Get("exports"); ev != nil && !goja.IsUndefined(ev) && !goja.IsNull(ev) {
					if c, ok := goja.AssertFunction(ev); ok {
						fn = c
					}
				}
			}
		}
	}
	if fn == nil {
		return nil, errors.New("no JS entry point found (expected: function main(params), exports.default = fn, or module.exports = fn)")
	}

	// Call it.
	params := opts.Params
	if params == nil {
		params = map[string]any{}
	}
	ret, err := fn(goja.Undefined(), vm.ToValue(params))
	if err != nil {
		return nil, errors.Wrap(err, "call main(params)")
	}

	// Convert result to Go via JSON so we can rely on json tags (rm_dsl, rotate_deg, ...).
	// goja.ExportTo does not reliably honor struct tags for our nested schema shape.
	var d Doc
	raw := ret.Export()
	jb, err := json.Marshal(raw)
	if err != nil {
		return nil, errors.Wrap(err, "marshal JS result")
	}
	if err := json.Unmarshal(jb, &d); err != nil {
		return nil, errors.Wrap(err, "unmarshal JS result")
	}
	// JS default: tolerate missing rm_dsl.
	if d.RMDSL == "" {
		d.RMDSL = DSLVersionV0
	}
	if err := Normalize(&d); err != nil {
		return nil, errors.Wrap(err, "normalize")
	}
	return &d, nil
}


