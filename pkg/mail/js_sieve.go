package mail

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// makeSieveUse returns the mail.sieve.use(accountOrOptions, fn) function.
func makeSieveUse(vm *goja.Runtime, opts RuntimeOptions) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("mail.sieve.use(account, fn) requires 2 arguments"))
		}
		sieveOpts := resolveSieveOptions(vm, call.Arguments[0], opts)
		fn, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(vm.NewTypeError("mail.sieve.use: second argument must be a function"))
		}

		sc, err := ConnectSieve(sieveOpts)
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		defer func() { _ = sc.Logout() }()

		clientObj := buildSieveClientObject(vm, sc)
		_, err = fn(goja.Undefined(), clientObj)
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return goja.Undefined()
	}
}

// resolveSieveOptions resolves account name or inline options to SieveOptions.
func resolveSieveOptions(vm *goja.Runtime, arg goja.Value, opts RuntimeOptions) SieveOptions {
	switch arg.ExportType().Kind().String() {
	case "string":
		name := arg.String()
		acc, ok := opts.Accounts[name]
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("unknown account %q", name)))
		}
		return acc.Sieve
	default:
		obj := arg.ToObject(vm)
		sieveOpts := SieveOptions{}
		if v := obj.Get("host"); v != nil {
			sieveOpts.Host = v.String()
		}
		if v := obj.Get("port"); v != nil {
			sieveOpts.Port = int(v.ToInteger())
		}
		// Note: SieveOptions does not have TLS field (plain TCP for now)
		if v := obj.Get("auth"); v != nil {
			authObj := v.ToObject(vm)
			if u := authObj.Get("user"); u != nil {
				sieveOpts.Username = u.String()
			}
			if p := authObj.Get("pass"); p != nil {
				sieveOpts.Password = p.String()
			}
		}
		return sieveOpts
	}
}

// buildSieveClientObject builds the JS SieveClient object.
func buildSieveClientObject(vm *goja.Runtime, sc *SieveClient) *goja.Object {
	obj := vm.NewObject()

	// sieve.capabilities() → object
	_ = obj.Set("capabilities", func(call goja.FunctionCall) goja.Value {
		caps := sc.Capabilities()
		result := map[string]interface{}{
			"implementation": caps.Implementation,
			"sieve":          caps.Sieve,
			"notify":         caps.Notify,
			"sasl":           caps.SASL,
			"startTLS":       caps.StartTLS,
			"version":        caps.Version,
		}
		return vm.ToValue(result)
	})

	// sieve.listScripts() → [{name, active}, ...]
	_ = obj.Set("listScripts", func(call goja.FunctionCall) goja.Value {
		scripts, err := sc.ListScripts()
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		result := make([]interface{}, len(scripts))
		for i, s := range scripts {
			result[i] = map[string]interface{}{
				"name":   s.Name,
				"active": s.Active,
			}
		}
		return vm.ToValue(result)
	})

	// sieve.getScript(name) → string
	_ = obj.Set("getScript", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		content, err := sc.GetScript(name)
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return vm.ToValue(content)
	})

	// sieve.putScript(name, content, options?) → {ok: true}
	_ = obj.Set("putScript", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		content := call.Argument(1).String()
		activate := false
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Arguments[2]) {
			optsObj := call.Arguments[2].ToObject(vm)
			if v := optsObj.Get("activate"); v != nil {
				activate = v.ToBoolean()
			}
		}
		if err := sc.PutScript(name, content, false); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		if activate {
			if err := sc.Activate(name); err != nil {
				panic(vm.NewGoError(wrapSieveError(err)))
			}
		}
		return vm.ToValue(map[string]interface{}{"ok": true})
	})

	// sieve.activate(name)
	_ = obj.Set("activate", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := sc.Activate(name); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return goja.Undefined()
	})

	// sieve.deactivate()
	_ = obj.Set("deactivate", func(call goja.FunctionCall) goja.Value {
		if err := sc.Deactivate(); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return goja.Undefined()
	})

	// sieve.deleteScript(name)
	_ = obj.Set("deleteScript", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := sc.DeleteScript(name); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return goja.Undefined()
	})

	// sieve.renameScript(oldName, newName)
	_ = obj.Set("renameScript", func(call goja.FunctionCall) goja.Value {
		old := call.Argument(0).String()
		newName := call.Argument(1).String()
		if err := sc.RenameScript(old, newName); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return goja.Undefined()
	})

	// sieve.check(content) → {ok: true} or throws
	_ = obj.Set("check", func(call goja.FunctionCall) goja.Value {
		content := call.Argument(0).String()
		if err := sc.CheckScript(content); err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return vm.ToValue(map[string]interface{}{"ok": true})
	})

	// sieve.haveSpace(name, sizeBytes) → boolean
	_ = obj.Set("haveSpace", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		size := int(call.Argument(1).ToInteger())
		ok, err := sc.HaveSpace(name, size)
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return vm.ToValue(ok)
	})

	// sieve.build(fn) → Sieve script string (DSL builder)
	_ = obj.Set("build", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewTypeError("sieve.build: argument must be a function"))
		}
		builder := newSieveBuilder(vm)
		_, err := fn(goja.Undefined(), builder.obj)
		if err != nil {
			panic(vm.NewGoError(wrapSieveError(err)))
		}
		return vm.ToValue(builder.String())
	})

	return obj
}

// ---- Sieve DSL Builder ----

// sieveBuilder accumulates Sieve script lines.
type sieveBuilder struct {
	vm      *goja.Runtime
	obj     *goja.Object
	lines   []string
	indent  int
	actions *sieveActionBuilder
}

type sieveActionBuilder struct {
	vm     *goja.Runtime
	obj    *goja.Object
	parent *sieveBuilder
}

func newSieveBuilder(vm *goja.Runtime) *sieveBuilder {
	b := &sieveBuilder{vm: vm}
	b.obj = vm.NewObject()

	// r.require(["fileinto", "imap4flags"])
	_ = b.obj.Set("require", func(call goja.FunctionCall) goja.Value {
		exts := jsToStringSlice(vm, call.Argument(0))
		quoted := make([]string, len(exts))
		for i, e := range exts {
			quoted[i] = fmt.Sprintf("%q", e)
		}
		b.lines = append(b.lines, fmt.Sprintf("require [%s];", strings.Join(quoted, ", ")))
		return goja.Undefined()
	})

	// r.if(condition, actionFn) → adds if block
	_ = b.obj.Set("if", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("r.if(condition, actionFn) requires 2 arguments"))
		}
		cond := call.Arguments[0].String()
		actionFn, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(vm.NewTypeError("r.if: second argument must be a function"))
		}
		b.lines = append(b.lines, fmt.Sprintf("if %s {", cond))
		b.indent++
		ab := newSieveActionBuilder(vm, b)
		_, err := actionFn(goja.Undefined(), ab.obj)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		b.indent--
		b.lines = append(b.lines, "}")
		return goja.Undefined()
	})

	// r.all(c1, c2, ...) → "allof(...)" condition string
	_ = b.obj.Set("all", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = a.String()
		}
		return vm.ToValue(fmt.Sprintf("allof(%s)", strings.Join(parts, ", ")))
	})

	// r.any(c1, c2, ...) → "anyof(...)" condition string
	_ = b.obj.Set("any", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = a.String()
		}
		return vm.ToValue(fmt.Sprintf("anyof(%s)", strings.Join(parts, ", ")))
	})

	// r.not(condition) → "not condition" string
	_ = b.obj.Set("not", func(call goja.FunctionCall) goja.Value {
		cond := call.Argument(0).String()
		return vm.ToValue(fmt.Sprintf("not %s", cond))
	})

	// r.headerContains(header, value) → condition string
	_ = b.obj.Set("headerContains", func(call goja.FunctionCall) goja.Value {
		header := call.Argument(0).String()
		value := call.Argument(1).String()
		return vm.ToValue(fmt.Sprintf("header :contains %q %q", header, value))
	})

	// r.headerMatches(header, pattern) → condition string
	_ = b.obj.Set("headerMatches", func(call goja.FunctionCall) goja.Value {
		header := call.Argument(0).String()
		patternVal := call.Argument(1)
		pattern := jsStringOrRegex(patternVal)
		return vm.ToValue(fmt.Sprintf("header :matches %q %q", header, pattern))
	})

	// r.headerIs(header, value) → condition string
	_ = b.obj.Set("headerIs", func(call goja.FunctionCall) goja.Value {
		header := call.Argument(0).String()
		value := call.Argument(1).String()
		return vm.ToValue(fmt.Sprintf("header :is %q %q", header, value))
	})

	// r.address(part, header, value) → condition string
	_ = b.obj.Set("address", func(call goja.FunctionCall) goja.Value {
		part := call.Argument(0).String()
		header := call.Argument(1).String()
		value := call.Argument(2).String()
		return vm.ToValue(fmt.Sprintf("address :%s %q %q", part, header, value))
	})

	// r.size(comparator, bytes) → condition string
	_ = b.obj.Set("size", func(call goja.FunctionCall) goja.Value {
		comp := call.Argument(0).String() // "over" or "under"
		bytes_ := call.Argument(1).ToInteger()
		return vm.ToValue(fmt.Sprintf("size :%s %d", comp, bytes_))
	})

	// r.true() → "true" condition
	_ = b.obj.Set("true", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue("true")
	})

	// r.false() → "false" condition
	_ = b.obj.Set("false", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue("false")
	})

	return b
}

func newSieveActionBuilder(vm *goja.Runtime, parent *sieveBuilder) *sieveActionBuilder {
	ab := &sieveActionBuilder{vm: vm, parent: parent}
	ab.obj = vm.NewObject()

	addLine := func(line string) {
		indent := strings.Repeat("  ", parent.indent)
		parent.lines = append(parent.lines, indent+line)
	}

	// a.fileInto(folder)
	_ = ab.obj.Set("fileInto", func(call goja.FunctionCall) goja.Value {
		folder := call.Argument(0).String()
		addLine(fmt.Sprintf("fileinto %q;", folder))
		return goja.Undefined()
	})

	// a.redirect(address)
	_ = ab.obj.Set("redirect", func(call goja.FunctionCall) goja.Value {
		addr := call.Argument(0).String()
		addLine(fmt.Sprintf("redirect %q;", addr))
		return goja.Undefined()
	})

	// a.keep()
	_ = ab.obj.Set("keep", func(call goja.FunctionCall) goja.Value {
		addLine("keep;")
		return goja.Undefined()
	})

	// a.discard()
	_ = ab.obj.Set("discard", func(call goja.FunctionCall) goja.Value {
		addLine("discard;")
		return goja.Undefined()
	})

	// a.stop()
	_ = ab.obj.Set("stop", func(call goja.FunctionCall) goja.Value {
		addLine("stop;")
		return goja.Undefined()
	})

	// a.addFlag(flag)
	_ = ab.obj.Set("addFlag", func(call goja.FunctionCall) goja.Value {
		flag := call.Argument(0).String()
		addLine(fmt.Sprintf("addflag %q;", flag))
		return goja.Undefined()
	})

	// a.removeFlag(flag)
	_ = ab.obj.Set("removeFlag", func(call goja.FunctionCall) goja.Value {
		flag := call.Argument(0).String()
		addLine(fmt.Sprintf("removeflag %q;", flag))
		return goja.Undefined()
	})

	// a.setFlag(flag)
	_ = ab.obj.Set("setFlag", func(call goja.FunctionCall) goja.Value {
		flag := call.Argument(0).String()
		addLine(fmt.Sprintf("setflag %q;", flag))
		return goja.Undefined()
	})

	// a.setVariable(name, value)
	_ = ab.obj.Set("setVariable", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		value := call.Argument(1).String()
		addLine(fmt.Sprintf("set %q %q;", name, value))
		return goja.Undefined()
	})

	// a.vacation(options)
	_ = ab.obj.Set("vacation", func(call goja.FunctionCall) goja.Value {
		optsObj := call.Argument(0).ToObject(vm)
		var parts []string
		if v := optsObj.Get("days"); v != nil && !goja.IsUndefined(v) {
			parts = append(parts, fmt.Sprintf(":days %d", v.ToInteger()))
		}
		if v := optsObj.Get("subject"); v != nil && !goja.IsUndefined(v) {
			parts = append(parts, fmt.Sprintf(":subject %q", v.String()))
		}
		msg := ""
		if v := optsObj.Get("message"); v != nil && !goja.IsUndefined(v) {
			msg = v.String()
		}
		addLine(fmt.Sprintf("vacation %s %q;", strings.Join(parts, " "), msg))
		return goja.Undefined()
	})

	// a.raw(line) - escape hatch for raw Sieve lines
	_ = ab.obj.Set("raw", func(call goja.FunctionCall) goja.Value {
		line := call.Argument(0).String()
		addLine(line)
		return goja.Undefined()
	})

	return ab
}

func (b *sieveBuilder) String() string {
	return strings.Join(b.lines, "\n") + "\n"
}

// jsToStringSlice converts a JS array or string to []string.
func jsToStringSlice(vm *goja.Runtime, val goja.Value) []string {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}
	exported := val.Export()
	switch v := exported.(type) {
	case []interface{}:
		strs := make([]string, len(v))
		for i, el := range v {
			strs[i] = fmt.Sprintf("%v", el)
		}
		return strs
	case string:
		return []string{v}
	default:
		if obj, ok := val.(*goja.Object); ok {
			if lenV := obj.Get("length"); lenV != nil {
				length := int(lenV.ToInteger())
				strs := make([]string, length)
				for i := 0; i < length; i++ {
					el := obj.Get(fmt.Sprintf("%d", i))
					if el != nil {
						strs[i] = el.String()
					}
				}
				return strs
			}
		}
		return []string{fmt.Sprintf("%v", exported)}
	}
}

// wrapSieveError wraps a Go error into a MailError with source "sieve".
func wrapSieveError(err error) error {
	if err == nil {
		return nil
	}
	var me *MailError
	if err != nil {
		if me2, ok := err.(*MailError); ok {
			_ = me2
			return err
		}
	}
	_ = me
	return &MailError{
		Name:    "SieveError",
		Message: err.Error(),
		Source:  "sieve",
	}
}
