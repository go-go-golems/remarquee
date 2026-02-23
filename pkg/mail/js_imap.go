// Package mail provides goja JS bindings for IMAP and ManageSieve operations.
package mail

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/emersion/go-imap/v2"
	"github.com/pkg/errors"
)

// AccountConfig holds named account credentials for the JS runtime.
type AccountConfig struct {
	IMAP  IMAPOptions
	Sieve SieveOptions
}

// RuntimeOptions configures the JS mail runtime.
type RuntimeOptions struct {
	// Accounts maps account names to their configs.
	Accounts map[string]AccountConfig
	// Secrets maps secret names to their values (for mail.secret()).
	Secrets map[string]string
	// Context is the parent context for all connections.
	Context context.Context
}

// RegisterMailModule registers the `mail` object into a goja runtime.
// After calling this, JS scripts can do: mail.imap.use("account", fn)
func RegisterMailModule(vm *goja.Runtime, opts RuntimeOptions) error {
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Accounts == nil {
		opts.Accounts = make(map[string]AccountConfig)
	}
	if opts.Secrets == nil {
		opts.Secrets = make(map[string]string)
	}

	mailObj := vm.NewObject()

	// mail.secret(name) → string
	if err := mailObj.Set("secret", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		val, ok := opts.Secrets[name]
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("secret %q not found", name)))
		}
		return vm.ToValue(val)
	}); err != nil {
		return errors.Wrap(err, "set mail.secret")
	}

	// mail.imap object
	imapObj := vm.NewObject()
	if err := imapObj.Set("use", makeImapUse(vm, opts)); err != nil {
		return errors.Wrap(err, "set mail.imap.use")
	}
	if err := mailObj.Set("imap", imapObj); err != nil {
		return errors.Wrap(err, "set mail.imap")
	}

	// mail.sieve object
	sieveObj := vm.NewObject()
	if err := sieveObj.Set("use", makeSieveUse(vm, opts)); err != nil {
		return errors.Wrap(err, "set mail.sieve.use")
	}
	if err := mailObj.Set("sieve", sieveObj); err != nil {
		return errors.Wrap(err, "set mail.sieve")
	}

	// Set mail as a global and also as require("mail") via a simple require shim
	if err := vm.Set("mail", mailObj); err != nil {
		return errors.Wrap(err, "set mail global")
	}

	// Simple require() shim for `var mail = require("mail")`
	requireFn := func(call goja.FunctionCall) goja.Value {
		mod := call.Argument(0).String()
		if mod == "mail" {
			return mailObj
		}
		panic(vm.NewGoError(fmt.Errorf("require: unknown module %q", mod)))
	}
	if err := vm.Set("require", requireFn); err != nil {
		return errors.Wrap(err, "set require")
	}

	return nil
}

// makeImapUse returns the mail.imap.use(accountOrOptions, fn) function.
func makeImapUse(vm *goja.Runtime, opts RuntimeOptions) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("mail.imap.use(account, fn) requires 2 arguments"))
		}
		imapOpts := resolveIMAPOptions(vm, call.Arguments[0], opts)
		fn, ok := goja.AssertFunction(call.Arguments[1])
		if !ok {
			panic(vm.NewTypeError("mail.imap.use: second argument must be a function"))
		}

		ic, err := Connect(opts.Context, imapOpts)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		defer func() { _ = ic.Logout() }()

		clientObj := buildImapClientObject(vm, ic, opts)
		_, err = fn(goja.Undefined(), clientObj)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	}
}

// resolveIMAPOptions resolves account name or inline options object to IMAPOptions.
func resolveIMAPOptions(vm *goja.Runtime, arg goja.Value, opts RuntimeOptions) IMAPOptions {
	switch arg.ExportType().Kind().String() {
	case "string":
		name := arg.String()
		acc, ok := opts.Accounts[name]
		if !ok {
			panic(vm.NewGoError(fmt.Errorf("unknown account %q", name)))
		}
		return acc.IMAP
	default:
		// Inline options object: { host, port, tls, auth: { user, pass } }
		obj := arg.ToObject(vm)
		imapOpts := IMAPOptions{}
		if v := obj.Get("host"); v != nil {
			imapOpts.Host = v.String()
		}
		if v := obj.Get("port"); v != nil {
			imapOpts.Port = int(v.ToInteger())
		}
		if v := obj.Get("tls"); v != nil {
			imapOpts.TLS = v.ToBoolean()
		}
		if v := obj.Get("auth"); v != nil {
			authObj := v.ToObject(vm)
			if u := authObj.Get("user"); u != nil {
				imapOpts.Username = u.String()
			}
			if p := authObj.Get("pass"); p != nil {
				imapOpts.Password = p.String()
			}
		}
		return imapOpts
	}
}

// buildImapClientObject builds the JS ImapClient object.
func buildImapClientObject(vm *goja.Runtime, ic *IMAPClient, opts RuntimeOptions) *goja.Object {
	obj := vm.NewObject()

	// imap.capabilities → object
	_ = obj.Set("capabilities", func(call goja.FunctionCall) goja.Value {
		caps := vm.NewObject()
		for k, v := range ic.Capabilities() {
			_ = caps.Set(k, v)
		}
		return caps
	})

	// imap.list(pattern?) → [{name, flags, delimiter}, ...]
	_ = obj.Set("list", func(call goja.FunctionCall) goja.Value {
		pattern := "*"
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			pattern = call.Arguments[0].String()
		}
		boxes, err := ic.List(pattern)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		result := make([]interface{}, len(boxes))
		for i, b := range boxes {
			result[i] = map[string]interface{}{
				"name":      b.Name,
				"flags":     b.Flags,
				"delimiter": b.Delimiter,
			}
		}
		return vm.ToValue(result)
	})

	// imap.status(name, items?) → {messages, unseen, uidNext, uidValidity, ...}
	_ = obj.Set("status", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("imap.status(name) requires a mailbox name"))
		}
		name := call.Arguments[0].String()
		st, err := ic.Status(name)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{
			"messages":    st.Messages,
			"unseen":      st.Unseen,
			"uidNext":     st.UIDNext,
			"uidValidity": st.UIDValidity,
			"recent":      st.Recent,
		})
	})

	// imap.mailbox(name) → Mailbox handle (lazy)
	_ = obj.Set("mailbox", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("imap.mailbox(name) requires a name"))
		}
		name := call.Arguments[0].String()
		return buildMailboxObject(vm, ic, name, false, nil)
	})

	// imap.withMailbox(name, options?, fn) → opens, runs fn(mbox), closes
	_ = obj.Set("withMailbox", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("imap.withMailbox(name, fn) requires at least 2 arguments"))
		}
		name := call.Arguments[0].String()
		readOnly := false
		var fn goja.Callable

		if len(call.Arguments) == 2 {
			var ok bool
			fn, ok = goja.AssertFunction(call.Arguments[1])
			if !ok {
				panic(vm.NewTypeError("imap.withMailbox: last argument must be a function"))
			}
		} else {
			// options object in position 1
			optsObj := call.Arguments[1].ToObject(vm)
			if v := optsObj.Get("readOnly"); v != nil {
				readOnly = v.ToBoolean()
			}
			if v := optsObj.Get("examine"); v != nil {
				readOnly = v.ToBoolean()
			}
			var ok bool
			fn, ok = goja.AssertFunction(call.Arguments[2])
			if !ok {
				panic(vm.NewTypeError("imap.withMailbox: last argument must be a function"))
			}
		}

		selectData, err := ic.SelectMailbox(name, readOnly)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		defer func() { _ = ic.UnselectMailbox() }()

		mboxObj := buildMailboxObject(vm, ic, name, readOnly, selectData)
		_, err = fn(goja.Undefined(), mboxObj)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})

	// imap.batch(fn) → groups commands (synchronous execution in sequence)
	_ = obj.Set("batch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("imap.batch(fn) requires a function"))
		}
		fn, ok := goja.AssertFunction(call.Arguments[0])
		if !ok {
			panic(vm.NewTypeError("imap.batch: argument must be a function"))
		}
		batchObj := buildBatchObject(vm, ic)
		ret, err := fn(goja.Undefined(), batchObj)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return ret
	})

	// imap.create(name), imap.rename(old, new), imap.drop(name)
	_ = obj.Set("create", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := ic.CreateMailbox(name); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("rename", func(call goja.FunctionCall) goja.Value {
		old := call.Argument(0).String()
		newName := call.Argument(1).String()
		if err := ic.RenameMailbox(old, newName); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("drop", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := ic.DeleteMailbox(name); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("subscribe", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := ic.Subscribe(name); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("unsubscribe", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if err := ic.Unsubscribe(name); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})

	return obj
}

// buildMailboxObject builds the JS Mailbox object.
func buildMailboxObject(vm *goja.Runtime, ic *IMAPClient, name string, readOnly bool, selectData *imap.SelectData) *goja.Object {
	obj := vm.NewObject()

	_ = obj.Set("name", name)
	_ = obj.Set("readOnly", readOnly)

	if selectData != nil {
		_ = obj.Set("uidValidity", selectData.UIDValidity)
		if selectData.UIDNext != 0 {
			_ = obj.Set("uidNext", uint32(selectData.UIDNext))
		}
		_ = obj.Set("numMessages", selectData.NumMessages)
	}

	// mbox.search(criteria) → [uid, ...]
	_ = obj.Set("search", func(call goja.FunctionCall) goja.Value {
		criteria := &SearchCriteria{}
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			criteria = parseCriteria(vm, call.Arguments[0])
		}
		uids, err := ic.Search(criteria)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		result := make([]interface{}, len(uids))
		for i, uid := range uids {
			result[i] = uint32(uid)
		}
		return vm.ToValue(result)
	})

	// mbox.query(criteria?) → Query builder
	_ = obj.Set("query", func(call goja.FunctionCall) goja.Value {
		criteria := &SearchCriteria{}
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			criteria = parseCriteria(vm, call.Arguments[0])
		}
		return buildQueryObject(vm, ic, criteria)
	})

	// mbox.get(uid) → Message (lightweight handle, lazy fetch)
	_ = obj.Set("get", func(call goja.FunctionCall) goja.Value {
		uid := imap.UID(call.Argument(0).ToInteger())
		return buildMessageObject(vm, ic, uid, nil)
	})

	// mbox.fetch(uids, fields, options?) → Message[]
	_ = obj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		uids := jsToUIDs(vm, call.Argument(0))
		fields := jsToFetchFields(vm, call.Argument(1))
		msgs, err := ic.Fetch(uids, fields)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		result := make([]interface{}, len(msgs))
		for i, m := range msgs {
			result[i] = fetchedMessageToJS(vm, ic, m)
		}
		return vm.ToValue(result)
	})

	// mbox.move(uids, destMailbox) → {moved: n}
	_ = obj.Set("move", func(call goja.FunctionCall) goja.Value {
		uids := jsToUIDs(vm, call.Argument(0))
		dest := call.Argument(1).String()
		if err := ic.MoveUIDs(uids, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"moved": len(uids)})
	})

	// mbox.copy(uids, destMailbox)
	_ = obj.Set("copy", func(call goja.FunctionCall) goja.Value {
		uids := jsToUIDs(vm, call.Argument(0))
		dest := call.Argument(1).String()
		if err := ic.CopyUIDs(uids, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"copied": len(uids)})
	})

	// mbox.delete(uids, options?) → marks \Deleted + expunge
	_ = obj.Set("delete", func(call goja.FunctionCall) goja.Value {
		uids := jsToUIDs(vm, call.Argument(0))
		expunge := true
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			optsObj := call.Arguments[1].ToObject(vm)
			if v := optsObj.Get("expunge"); v != nil {
				expunge = v.ToBoolean()
			}
		}
		if err := ic.DeleteUIDs(uids, expunge); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"deleted": len(uids)})
	})

	// mbox.expunge(uids?)
	_ = obj.Set("expunge", func(call goja.FunctionCall) goja.Value {
		var uids []imap.UID
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			uids = jsToUIDs(vm, call.Arguments[0])
		}
		if err := ic.Expunge(uids); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})

	// mbox.append(destMailbox?, rfc822, flags?, date?) → uid
	_ = obj.Set("append", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("mbox.append requires at least rfc822 content"))
		}
		// Detect if first arg is mailbox name or message content
		var mailboxName string
		var msgBytes []byte
		var flags []imap.Flag
		var date *time.Time

		argIdx := 0
		// If first arg is a string that looks like a mailbox name (no \r\n), treat as dest
		first := call.Arguments[0]
		if first.ExportType().Kind().String() == "string" && !strings.Contains(first.String(), "\r\n") && !strings.Contains(first.String(), "\n") {
			mailboxName = first.String()
			argIdx = 1
		} else {
			mailboxName = ic.SelectedMailbox()
		}

		if argIdx < len(call.Arguments) {
			raw := call.Arguments[argIdx].Export()
			switch v := raw.(type) {
			case string:
				msgBytes = []byte(v)
			case []byte:
				msgBytes = v
			default:
				msgBytes = []byte(fmt.Sprintf("%v", v))
			}
			argIdx++
		}

		if argIdx < len(call.Arguments) && !goja.IsUndefined(call.Arguments[argIdx]) {
			flags = jsToFlags(vm, call.Arguments[argIdx])
			argIdx++
		}

		if argIdx < len(call.Arguments) && !goja.IsUndefined(call.Arguments[argIdx]) {
			t := jsToTime(vm, call.Arguments[argIdx])
			date = &t
		}

		uid, err := ic.Append(mailboxName, msgBytes, flags, date)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(uint32(uid))
	})

	// mbox.create(name), mbox.rename(old, new), mbox.drop(name)
	_ = obj.Set("create", func(call goja.FunctionCall) goja.Value {
		if err := ic.CreateMailbox(call.Argument(0).String()); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("rename", func(call goja.FunctionCall) goja.Value {
		if err := ic.RenameMailbox(call.Argument(0).String(), call.Argument(1).String()); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("drop", func(call goja.FunctionCall) goja.Value {
		if err := ic.DeleteMailbox(call.Argument(0).String()); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("subscribe", func(call goja.FunctionCall) goja.Value {
		if err := ic.Subscribe(call.Argument(0).String()); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	_ = obj.Set("unsubscribe", func(call goja.FunctionCall) goja.Value {
		if err := ic.Unsubscribe(call.Argument(0).String()); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})

	return obj
}

// buildQueryObject builds the JS Query builder object.
func buildQueryObject(vm *goja.Runtime, ic *IMAPClient, criteria *SearchCriteria) *goja.Object {
	q := &jsQuery{
		ic:       ic,
		criteria: criteria,
		fields:   []FetchField{FetchUID, FetchFlags, FetchEnvelope, FetchInternalDate, FetchSize},
		limitN:   -1,
	}

	obj := vm.NewObject()

	// q.limit(n) → q
	_ = obj.Set("limit", func(call goja.FunctionCall) goja.Value {
		q.limitN = int(call.Argument(0).ToInteger())
		return obj
	})

	// q.peek(bool) → q
	_ = obj.Set("peek", func(call goja.FunctionCall) goja.Value {
		q.peek = call.Argument(0).ToBoolean()
		return obj
	})

	// q.fetch(fields) → q
	_ = obj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		q.fields = jsToFetchFields(vm, call.Argument(0))
		return obj
	})

	// q.uids() → [uid, ...]
	_ = obj.Set("uids", func(call goja.FunctionCall) goja.Value {
		uids, err := ic.Search(q.criteria)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if q.limitN >= 0 && len(uids) > q.limitN {
			uids = uids[:q.limitN]
		}
		result := make([]interface{}, len(uids))
		for i, uid := range uids {
			result[i] = uint32(uid)
		}
		return vm.ToValue(result)
	})

	// q.list() → Message[]
	_ = obj.Set("list", func(call goja.FunctionCall) goja.Value {
		msgs := q.execute(vm)
		return vm.ToValue(msgs)
	})

	// q.each(fn) → executes and calls fn(msg) per message
	_ = obj.Set("each", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.NewTypeError("q.each: argument must be a function"))
		}
		msgs := q.execute(vm)
		for _, msg := range msgs {
			_, err := fn(goja.Undefined(), vm.ToValue(msg))
			if err != nil {
				panic(vm.NewGoError(wrapMailError(err)))
			}
		}
		return goja.Undefined()
	})

	return obj
}

type jsQuery struct {
	ic       *IMAPClient
	criteria *SearchCriteria
	fields   []FetchField
	limitN   int
	peek     bool
}

func (q *jsQuery) execute(vm *goja.Runtime) []interface{} {
	uids, err := q.ic.Search(q.criteria)
	if err != nil {
		panic(vm.NewGoError(wrapMailError(err)))
	}
	if q.limitN >= 0 && len(uids) > q.limitN {
		uids = uids[:q.limitN]
	}
	if len(uids) == 0 {
		return nil
	}
	msgs, err := q.ic.Fetch(uids, q.fields)
	if err != nil {
		panic(vm.NewGoError(wrapMailError(err)))
	}
	result := make([]interface{}, len(msgs))
	for i, m := range msgs {
		result[i] = fetchedMessageToJS(vm, q.ic, m)
	}
	return result
}

// buildMessageObject builds a lazy Message handle.
func buildMessageObject(vm *goja.Runtime, ic *IMAPClient, uid imap.UID, fetched *FetchedMessage) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("uid", uint32(uid))
	_ = obj.Set("mailbox", ic.SelectedMailbox())

	if fetched != nil {
		populateMessageFields(vm, obj, ic, fetched)
	}

	// msg.addFlags(flags)
	_ = obj.Set("addFlags", func(call goja.FunctionCall) goja.Value {
		flags := jsToFlags(vm, call.Argument(0))
		if err := ic.StoreFlags([]imap.UID{uid}, imap.StoreFlagsAdd, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.removeFlags(flags)
	_ = obj.Set("removeFlags", func(call goja.FunctionCall) goja.Value {
		flags := jsToFlags(vm, call.Argument(0))
		if err := ic.StoreFlags([]imap.UID{uid}, imap.StoreFlagsDel, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.setFlags(flags)
	_ = obj.Set("setFlags", func(call goja.FunctionCall) goja.Value {
		flags := jsToFlags(vm, call.Argument(0))
		if err := ic.StoreFlags([]imap.UID{uid}, imap.StoreFlagsSet, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.markSeen()
	_ = obj.Set("markSeen", func(call goja.FunctionCall) goja.Value {
		if err := ic.StoreFlags([]imap.UID{uid}, imap.StoreFlagsAdd, []imap.Flag{imap.FlagSeen}, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.markUnseen()
	_ = obj.Set("markUnseen", func(call goja.FunctionCall) goja.Value {
		if err := ic.StoreFlags([]imap.UID{uid}, imap.StoreFlagsDel, []imap.Flag{imap.FlagSeen}, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.move(destMailbox)
	_ = obj.Set("move", func(call goja.FunctionCall) goja.Value {
		dest := call.Argument(0).String()
		if err := ic.MoveUIDs([]imap.UID{uid}, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.copy(destMailbox)
	_ = obj.Set("copy", func(call goja.FunctionCall) goja.Value {
		dest := call.Argument(0).String()
		if err := ic.CopyUIDs([]imap.UID{uid}, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.delete()
	_ = obj.Set("delete", func(call goja.FunctionCall) goja.Value {
		if err := ic.DeleteUIDs([]imap.UID{uid}, true); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})
	// msg.getBody(options?) → string
	_ = obj.Set("getBody", func(call goja.FunctionCall) goja.Value {
		prefer := "text"
		maxBytes := 0
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) {
			optsObj := call.Arguments[0].ToObject(vm)
			if v := optsObj.Get("prefer"); v != nil {
				prefer = v.String()
			}
			if v := optsObj.Get("maxBytes"); v != nil {
				maxBytes = int(v.ToInteger())
			}
		}
		fields := []FetchField{FetchBodyText, FetchBodyHTML}
		msgs, err := ic.Fetch([]imap.UID{uid}, fields)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		var body string
		if len(msgs) > 0 {
			m := msgs[0]
			if prefer == "html" {
				if m.BodyHTML != "" {
					body = m.BodyHTML
				} else {
					body = m.BodyText
				}
			} else {
				if m.BodyText != "" {
					body = m.BodyText
				} else {
					body = m.BodyHTML
				}
			}
			if maxBytes > 0 && len(body) > maxBytes {
				body = body[:maxBytes]
			}
		}
		_ = err
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(body)
	})
	// msg.getPart(partId, options?) → bytes as string
	_ = obj.Set("getPart", func(call goja.FunctionCall) goja.Value {
		partStr := call.Argument(0).String()
		part := parsePartID(partStr)
		data, err := ic.FetchBodyPart(uid, part)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(string(data))
	})
	// msg.saveAttachments(dir, options?) → not implemented (host decides FS)
	_ = obj.Set("saveAttachments", func(call goja.FunctionCall) goja.Value {
		panic(vm.NewGoError(fmt.Errorf("saveAttachments: not implemented in this host")))
	})

	return obj
}

// populateMessageFields fills in the fetched fields on a message object.
func populateMessageFields(vm *goja.Runtime, obj *goja.Object, ic *IMAPClient, m *FetchedMessage) {
	_ = obj.Set("uid", m.UID)
	_ = obj.Set("mailbox", m.mailboxName)
	if len(m.Flags) > 0 {
		_ = obj.Set("flags", m.Flags)
	}
	if m.Size > 0 {
		_ = obj.Set("size", m.Size)
	}
	if m.InternalDate != "" {
		_ = obj.Set("internalDate", m.InternalDate)
	}
	if m.Envelope != nil {
		_ = obj.Set("envelope", map[string]interface{}{
			"date":      m.Envelope.Date,
			"subject":   m.Envelope.Subject,
			"from":      m.Envelope.From,
			"to":        m.Envelope.To,
			"cc":        m.Envelope.CC,
			"bcc":       m.Envelope.BCC,
			"replyTo":   m.Envelope.ReplyTo,
			"messageId": m.Envelope.MessageID,
			"inReplyTo": m.Envelope.InReplyTo,
		})
	}
	if m.Headers != nil {
		_ = obj.Set("headers", m.Headers)
	}
	if m.BodyText != "" {
		_ = obj.Set("bodyText", m.BodyText)
	}
	if m.BodyHTML != "" {
		_ = obj.Set("bodyHTML", m.BodyHTML)
	}
	if len(m.Attachments) > 0 {
		atts := make([]interface{}, len(m.Attachments))
		for i, a := range m.Attachments {
			atts[i] = map[string]interface{}{
				"part":        a.Part,
				"filename":    a.Filename,
				"contentType": a.ContentType,
				"size":        a.Size,
				"cid":         a.CID,
			}
		}
		_ = obj.Set("attachments", atts)
	}
}

// fetchedMessageToJS converts a FetchedMessage to a JS object with all actions.
func fetchedMessageToJS(vm *goja.Runtime, ic *IMAPClient, m *FetchedMessage) *goja.Object {
	uid := imap.UID(m.UID)
	obj := buildMessageObject(vm, ic, uid, m)
	return obj
}

// buildBatchObject builds the JS batch object.
func buildBatchObject(vm *goja.Runtime, ic *IMAPClient) *goja.Object {
	obj := vm.NewObject()

	_ = obj.Set("fetch", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		fields := jsToFetchFields(vm, call.Argument(2))

		_, err := ic.SelectMailbox(mailboxName, true)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		msgs, err := ic.Fetch(uids, fields)
		if err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		result := make([]interface{}, len(msgs))
		for i, m := range msgs {
			result[i] = fetchedMessageToJS(vm, ic, m)
		}
		return vm.ToValue(result)
	})

	_ = obj.Set("move", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		dest := call.Argument(2).String()
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.MoveUIDs(uids, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"moved": len(uids)})
	})

	_ = obj.Set("copy", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		dest := call.Argument(2).String()
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.CopyUIDs(uids, dest); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"copied": len(uids)})
	})

	_ = obj.Set("addFlags", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		flags := jsToFlags(vm, call.Argument(2))
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.StoreFlags(uids, imap.StoreFlagsAdd, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"flagged": len(uids)})
	})

	_ = obj.Set("removeFlags", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		flags := jsToFlags(vm, call.Argument(2))
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.StoreFlags(uids, imap.StoreFlagsDel, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"unflagged": len(uids)})
	})

	_ = obj.Set("setFlags", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		flags := jsToFlags(vm, call.Argument(2))
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.StoreFlags(uids, imap.StoreFlagsSet, flags, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"flagged": len(uids)})
	})

	_ = obj.Set("delete", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		uids := jsToUIDs(vm, call.Argument(1))
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.DeleteUIDs(uids, true); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return vm.ToValue(map[string]interface{}{"deleted": len(uids)})
	})

	_ = obj.Set("expunge", func(call goja.FunctionCall) goja.Value {
		mailboxName := call.Argument(0).String()
		var uids []imap.UID
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			uids = jsToUIDs(vm, call.Arguments[1])
		}
		if _, err := ic.SelectMailbox(mailboxName, false); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		if err := ic.Expunge(uids); err != nil {
			panic(vm.NewGoError(wrapMailError(err)))
		}
		return goja.Undefined()
	})

	return obj
}

// ---- criteria parsing ----

// parseCriteria converts a JS criteria object or raw string to SearchCriteria.
func parseCriteria(vm *goja.Runtime, val goja.Value) *SearchCriteria {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return &SearchCriteria{All: true}
	}
	// Raw IMAP string escape hatch
	if val.ExportType().Kind().String() == "string" {
		return &SearchCriteria{Raw: val.String()}
	}

	obj := val.ToObject(vm)
	c := &SearchCriteria{}

	getBool := func(key string) (bool, bool) {
		v := obj.Get(key)
		if v == nil || goja.IsUndefined(v) {
			return false, false
		}
		return v.ToBoolean(), true
	}
	getString := func(key string) string {
		v := obj.Get(key)
		if v == nil || goja.IsUndefined(v) {
			return ""
		}
		return v.String()
	}

	if v, ok := getBool("all"); ok && v {
		c.All = true
	}
	if v, ok := getBool("seen"); ok && v {
		c.Seen = true
	}
	if v, ok := getBool("unseen"); ok && v {
		c.Unseen = true
	}
	if v, ok := getBool("flagged"); ok && v {
		c.Flagged = true
	}
	if v, ok := getBool("answered"); ok && v {
		c.Answered = true
	}
	if v, ok := getBool("deleted"); ok && v {
		c.Deleted = true
	}
	if v, ok := getBool("draft"); ok && v {
		c.Draft = true
	}

	// String/regex fields
	if v := obj.Get("from"); v != nil && !goja.IsUndefined(v) {
		c.From = jsStringOrRegex(v)
	}
	if v := obj.Get("to"); v != nil && !goja.IsUndefined(v) {
		c.To = jsStringOrRegex(v)
	}
	if v := obj.Get("cc"); v != nil && !goja.IsUndefined(v) {
		c.CC = jsStringOrRegex(v)
	}
	if v := obj.Get("bcc"); v != nil && !goja.IsUndefined(v) {
		c.BCC = jsStringOrRegex(v)
	}
	if v := obj.Get("subject"); v != nil && !goja.IsUndefined(v) {
		c.Subject = jsStringOrRegex(v)
	}
	if v := getString("text"); v != "" {
		c.Text = v
	}
	if v := getString("body"); v != "" {
		c.Body = v
	}

	// Date fields: Date | "YYYY-MM-DD" | "30d" relative
	if v := obj.Get("since"); v != nil && !goja.IsUndefined(v) {
		t := parseJSDate(v)
		c.Since = &t
	}
	if v := obj.Get("before"); v != nil && !goja.IsUndefined(v) {
		t := parseJSDate(v)
		c.Before = &t
	}
	if v := obj.Get("on"); v != nil && !goja.IsUndefined(v) {
		t := parseJSDate(v)
		c.On = &t
	}

	// Size fields
	if v := obj.Get("larger"); v != nil && !goja.IsUndefined(v) {
		c.Larger = v.ToInteger()
	}
	if v := obj.Get("smaller"); v != nil && !goja.IsUndefined(v) {
		c.Smaller = v.ToInteger()
	}

	// header: { "X-Spam": "YES" }
	if v := obj.Get("header"); v != nil && !goja.IsUndefined(v) {
		hObj := v.ToObject(vm)
		c.Header = make(map[string]string)
		for _, key := range hObj.Keys() {
			c.Header[key] = hObj.Get(key).String()
		}
	}

	// uid: "1:*" or [1,2,3]
	if v := obj.Get("uid"); v != nil && !goja.IsUndefined(v) {
		uidSet := parseUIDSet(v)
		c.UID = &uidSet
	}

	// not: criteria
	if v := obj.Get("not"); v != nil && !goja.IsUndefined(v) {
		c.Not = parseCriteria(vm, v)
	}

	// or: [c1, c2]
	if v := obj.Get("or"); v != nil && !goja.IsUndefined(v) {
		arr := v.ToObject(vm)
		keys := arr.Keys()
		for _, k := range keys {
			c.Or = append(c.Or, parseCriteria(vm, arr.Get(k)))
		}
	}

	// and: [c1, c2]
	if v := obj.Get("and"); v != nil && !goja.IsUndefined(v) {
		arr := v.ToObject(vm)
		keys := arr.Keys()
		for _, k := range keys {
			c.And = append(c.And, parseCriteria(vm, arr.Get(k)))
		}
	}

	return c
}

// jsStringOrRegex extracts the string value from a JS string or regex.
// For regex, it extracts the source pattern (IMAP doesn't support regex,
// so we use the pattern as a substring match).
func jsStringOrRegex(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	// Check if it's a RegExp object
	if obj, ok := v.(*goja.Object); ok {
		if src := obj.Get("source"); src != nil && !goja.IsUndefined(src) {
			// Extract a simple substring from the regex source
			return regexToSubstring(src.String())
		}
	}
	return v.String()
}

// regexToSubstring extracts a simple literal substring from a regex pattern.
// This is a best-effort conversion for IMAP search (which doesn't support regex).
func regexToSubstring(pattern string) string {
	// Remove anchors and flags
	pattern = strings.TrimPrefix(pattern, "^")
	pattern = strings.TrimSuffix(pattern, "$")
	// Remove alternation and use first branch
	if idx := strings.Index(pattern, "|"); idx >= 0 {
		pattern = pattern[:idx]
	}
	// Remove special regex chars
	re := regexp.MustCompile(`[\\^$.*+?()[\]{}|]`)
	return re.ReplaceAllString(pattern, "")
}

// parseJSDate parses a JS Date, "YYYY-MM-DD" string, or "30d" relative string.
func parseJSDate(v goja.Value) time.Time {
	if obj, ok := v.(*goja.Object); ok {
		// JS Date object: call .getTime() to get ms since epoch
		if getTime := obj.Get("getTime"); getTime != nil {
			if fn, ok := goja.AssertFunction(getTime); ok {
				ret, err := fn(v)
				if err == nil {
					ms := ret.ToInteger()
					return time.Unix(ms/1000, (ms%1000)*1e6).UTC()
				}
			}
		}
	}
	s := v.String()
	// Relative: "30d", "7d", etc.
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		}
	}
	// Absolute: "YYYY-MM-DD"
	t, err := time.Parse("2006-01-02", s)
	if err == nil {
		return t.UTC()
	}
	// Try RFC3339
	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return t.UTC()
	}
	return time.Time{}
}

// parseUIDSet parses a JS value to imap.UIDSet.
// Accepts: "1:*", "1,2,3", [1,2,3], or a single number.
func parseUIDSet(v goja.Value) imap.UIDSet {
	if arr, ok := v.(*goja.Object); ok {
		// Array of UIDs
		if arr.Get("length") != nil {
			var uids []imap.UID
			length := int(arr.Get("length").ToInteger())
			for i := 0; i < length; i++ {
				el := arr.Get(strconv.Itoa(i))
				if el != nil {
					uids = append(uids, imap.UID(el.ToInteger()))
				}
			}
			return imap.UIDSetNum(uids...)
		}
	}
	s := v.String()
	// Parse "1:*" or "1,2,3" format
	var set imap.UIDSet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			bounds := strings.SplitN(part, ":", 2)
			var lo, hi imap.UID
			if bounds[0] == "*" {
				lo = 0
			} else {
				n, _ := strconv.ParseUint(bounds[0], 10, 32)
				lo = imap.UID(n)
			}
			if bounds[1] == "*" {
				hi = 0
			} else {
				n, _ := strconv.ParseUint(bounds[1], 10, 32)
				hi = imap.UID(n)
			}
			set.AddRange(lo, hi)
		} else {
			n, err := strconv.ParseUint(part, 10, 32)
			if err == nil {
				set.AddNum(imap.UID(n))
			}
		}
	}
	return set
}

// ---- conversion helpers ----

func jsToUIDs(vm *goja.Runtime, val goja.Value) []imap.UID {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}
	exported := val.Export()
	switch v := exported.(type) {
	case []interface{}:
		uids := make([]imap.UID, 0, len(v))
		for _, el := range v {
			switch n := el.(type) {
			case int64:
				uids = append(uids, imap.UID(n))
			case float64:
				uids = append(uids, imap.UID(n))
			case int:
				uids = append(uids, imap.UID(n))
			case uint32:
				uids = append(uids, imap.UID(n))
			}
		}
		return uids
	case int64:
		return []imap.UID{imap.UID(v)}
	case float64:
		return []imap.UID{imap.UID(v)}
	default:
		// Try as object/array
		if obj, ok := val.(*goja.Object); ok {
			if lenV := obj.Get("length"); lenV != nil {
				length := int(lenV.ToInteger())
				uids := make([]imap.UID, 0, length)
				for i := 0; i < length; i++ {
					el := obj.Get(strconv.Itoa(i))
					if el != nil {
						uids = append(uids, imap.UID(el.ToInteger()))
					}
				}
				return uids
			}
		}
		return []imap.UID{imap.UID(val.ToInteger())}
	}
}

func jsToFetchFields(vm *goja.Runtime, val goja.Value) []FetchField {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return []FetchField{FetchUID, FetchFlags, FetchEnvelope, FetchInternalDate, FetchSize}
	}
	exported := val.Export()
	var strs []string
	switch v := exported.(type) {
	case []interface{}:
		for _, el := range v {
			strs = append(strs, fmt.Sprintf("%v", el))
		}
	case string:
		strs = []string{v}
	default:
		if obj, ok := val.(*goja.Object); ok {
			if lenV := obj.Get("length"); lenV != nil {
				length := int(lenV.ToInteger())
				for i := 0; i < length; i++ {
					el := obj.Get(strconv.Itoa(i))
					if el != nil {
						strs = append(strs, el.String())
					}
				}
			}
		}
	}
	fields := make([]FetchField, 0, len(strs))
	for _, s := range strs {
		fields = append(fields, FetchField(s))
	}
	return fields
}

func jsToFlags(vm *goja.Runtime, val goja.Value) []imap.Flag {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}
	exported := val.Export()
	var strs []string
	switch v := exported.(type) {
	case []interface{}:
		for _, el := range v {
			strs = append(strs, fmt.Sprintf("%v", el))
		}
	case string:
		strs = []string{v}
	default:
		if obj, ok := val.(*goja.Object); ok {
			if lenV := obj.Get("length"); lenV != nil {
				length := int(lenV.ToInteger())
				for i := 0; i < length; i++ {
					el := obj.Get(strconv.Itoa(i))
					if el != nil {
						strs = append(strs, el.String())
					}
				}
			}
		}
	}
	flags := make([]imap.Flag, len(strs))
	for i, s := range strs {
		flags[i] = imap.Flag(s)
	}
	return flags
}

func jsToTime(vm *goja.Runtime, val goja.Value) time.Time {
	return parseJSDate(val)
}

// parsePartID parses "1.2.3" into []int{1, 2, 3}.
func parsePartID(s string) []int {
	parts := strings.Split(s, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err == nil {
			result = append(result, n)
		}
	}
	return result
}

// wrapMailError wraps a Go error into a MailError if it isn't one already.
func wrapMailError(err error) error {
	if err == nil {
		return nil
	}
	var me *MailError
	if errors.As(err, &me) {
		return me
	}
	return &MailError{
		Name:    "ProtocolError",
		Message: err.Error(),
		Source:  "imap",
	}
}
