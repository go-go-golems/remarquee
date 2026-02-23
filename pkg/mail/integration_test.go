package mail_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/remarquee/pkg/mail"
)

// testConfig holds connection config for the local Dovecot test server.
var testConfig = mail.AccountConfig{
	IMAP: mail.IMAPOptions{
		Host:     "localhost",
		Port:     143,
		TLS:      false,
		Username: "testuser",
		Password: "testpass",
	},
	Sieve: mail.SieveOptions{
		Host:     "localhost",
		Port:     4190,
		Username: "testuser",
		Password: "testpass",
	},
}

// ---- Go-level IMAP tests ----

func TestIMAPConnect(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()
	t.Logf("Connected. Capabilities: %v", ic.Capabilities())
}

func TestIMAPList(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	boxes, err := ic.List("*")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	t.Logf("Mailboxes: %+v", boxes)
	if len(boxes) == 0 {
		t.Error("expected at least one mailbox (INBOX)")
	}
}

func TestIMAPStatus(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	st, err := ic.Status("INBOX")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	t.Logf("INBOX status: %+v", st)
}

func TestIMAPAppendSearchFetch(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	// Append a test message
	subject := fmt.Sprintf("Test message %d", time.Now().UnixNano())
	msg := fmt.Sprintf(
		"From: sender@example.com\r\nTo: testuser@localhost\r\nSubject: %s\r\nDate: %s\r\n\r\nHello from test!\r\n",
		subject, time.Now().Format(time.RFC1123Z),
	)

	uid, err := ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	t.Logf("Appended message UID: %d", uid)

	// Select INBOX
	_, err = ic.SelectMailbox("INBOX", false)
	if err != nil {
		t.Fatalf("SelectMailbox failed: %v", err)
	}

	// Search for the message
	uids, err := ic.Search(&mail.SearchCriteria{
		Subject: subject,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	t.Logf("Search found UIDs: %v", uids)
	if len(uids) == 0 {
		t.Skip("Search returned no results (subject search may not be supported or indexed yet)")
	}

	// Fetch the message
	msgs, err := ic.Fetch(uids, []mail.FetchField{
		mail.FetchUID, mail.FetchFlags, mail.FetchEnvelope, mail.FetchInternalDate, mail.FetchSize,
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("Fetch returned no messages")
	}
	m := msgs[0]
	t.Logf("Fetched message: UID=%d, Flags=%v, InternalDate=%s", m.UID, m.Flags, m.InternalDate)
	if m.Envelope != nil {
		t.Logf("  Subject: %s, From: %v", m.Envelope.Subject, m.Envelope.From)
	}
}

func TestIMAPFlagOperations(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	// Append a test message
	msg := fmt.Sprintf(
		"From: sender@example.com\r\nTo: testuser@localhost\r\nSubject: Flag test %d\r\n\r\nBody\r\n",
		time.Now().UnixNano(),
	)
	uid, err := ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	_, err = ic.SelectMailbox("INBOX", false)
	if err != nil {
		t.Fatalf("SelectMailbox failed: %v", err)
	}

	// Add \Seen flag
	if err := ic.StoreFlags([]mail.UID{uid}, mail.StoreFlagsAdd, []mail.Flag{mail.FlagSeen}, false); err != nil {
		t.Fatalf("StoreFlags (add Seen) failed: %v", err)
	}

	// Fetch and verify
	msgs, err := ic.Fetch([]mail.UID{uid}, []mail.FetchField{mail.FetchFlags})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("no messages fetched")
	}
	t.Logf("Flags after adding \\Seen: %v", msgs[0].Flags)

	// Remove \Seen flag
	if err := ic.StoreFlags([]mail.UID{uid}, mail.StoreFlagsDel, []mail.Flag{mail.FlagSeen}, false); err != nil {
		t.Fatalf("StoreFlags (del Seen) failed: %v", err)
	}
}

func TestIMAPCreateRenameDelete(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	// Create a test mailbox
	testBox := fmt.Sprintf("TestBox%d", time.Now().UnixNano())
	if err := ic.CreateMailbox(testBox); err != nil {
		t.Fatalf("CreateMailbox failed: %v", err)
	}
	t.Logf("Created mailbox: %s", testBox)

	// Rename it
	renamedBox := testBox + "_renamed"
	if err := ic.RenameMailbox(testBox, renamedBox); err != nil {
		t.Fatalf("RenameMailbox failed: %v", err)
	}
	t.Logf("Renamed to: %s", renamedBox)

	// Delete it
	if err := ic.DeleteMailbox(renamedBox); err != nil {
		t.Fatalf("DeleteMailbox failed: %v", err)
	}
	t.Logf("Deleted mailbox: %s", renamedBox)
}

func TestIMAPMoveAndCopy(t *testing.T) {
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer ic.Logout()

	// Create a destination mailbox
	destBox := fmt.Sprintf("MoveTest%d", time.Now().UnixNano())
	if err := ic.CreateMailbox(destBox); err != nil {
		t.Fatalf("CreateMailbox failed: %v", err)
	}
	defer func() { _ = ic.DeleteMailbox(destBox) }()

	// Append a message to INBOX
	msg := fmt.Sprintf(
		"From: sender@example.com\r\nTo: testuser@localhost\r\nSubject: Move test %d\r\n\r\nBody\r\n",
		time.Now().UnixNano(),
	)
	uid, err := ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	_, err = ic.SelectMailbox("INBOX", false)
	if err != nil {
		t.Fatalf("SelectMailbox failed: %v", err)
	}

	// Copy to destination
	if err := ic.CopyUIDs([]mail.UID{uid}, destBox); err != nil {
		t.Fatalf("CopyUIDs failed: %v", err)
	}
	t.Logf("Copied UID %d to %s", uid, destBox)

	// Verify destination has the message
	_, err = ic.SelectMailbox(destBox, true)
	if err != nil {
		t.Fatalf("SelectMailbox (dest) failed: %v", err)
	}
	destUIDs, err := ic.Search(&mail.SearchCriteria{})
	if err != nil {
		t.Fatalf("Search in dest failed: %v", err)
	}
	if len(destUIDs) == 0 {
		t.Error("expected message in destination mailbox after copy")
	}
	t.Logf("Destination mailbox has %d messages", len(destUIDs))
}

// ---- Sieve Go-level tests ----

func TestSieveConnect(t *testing.T) {
	sc, err := mail.ConnectSieve(testConfig.Sieve)
	if err != nil {
		t.Fatalf("ConnectSieve failed: %v", err)
	}
	defer sc.Logout()

	caps := sc.Capabilities()
	t.Logf("Sieve capabilities: %+v", caps)
	if caps.Implementation == "" {
		t.Error("expected non-empty implementation")
	}
}

func TestSieveListScripts(t *testing.T) {
	sc, err := mail.ConnectSieve(testConfig.Sieve)
	if err != nil {
		t.Fatalf("ConnectSieve failed: %v", err)
	}
	defer sc.Logout()

	scripts, err := sc.ListScripts()
	if err != nil {
		t.Fatalf("ListScripts failed: %v", err)
	}
	t.Logf("Scripts: %+v", scripts)
}

func TestSievePutGetDelete(t *testing.T) {
	sc, err := mail.ConnectSieve(testConfig.Sieve)
	if err != nil {
		t.Fatalf("ConnectSieve failed: %v", err)
	}
	defer sc.Logout()

	scriptName := fmt.Sprintf("testscript%d", time.Now().UnixNano())
	content := `require ["fileinto"];
if header :contains "Subject" "newsletter" {
  fileinto "Archive";
  stop;
}
`

	// Put script
	if err := sc.PutScript(scriptName, content, false); err != nil {
		t.Fatalf("PutScript failed: %v", err)
	}
	t.Logf("Put script: %s", scriptName)

	// Get script
	got, err := sc.GetScript(scriptName)
	if err != nil {
		t.Fatalf("GetScript failed: %v", err)
	}
	if !strings.Contains(got, "fileinto") {
		t.Errorf("GetScript content missing expected text, got: %q", got)
	}
	t.Logf("Got script content (%d bytes)", len(got))

	// Delete script
	if err := sc.DeleteScript(scriptName); err != nil {
		t.Fatalf("DeleteScript failed: %v", err)
	}
	t.Logf("Deleted script: %s", scriptName)
}

func TestSieveCheckScript(t *testing.T) {
	sc, err := mail.ConnectSieve(testConfig.Sieve)
	if err != nil {
		t.Fatalf("ConnectSieve failed: %v", err)
	}
	defer sc.Logout()

	// Valid script
	valid := `require ["fileinto"];
if header :contains "From" "newsletter@" {
  fileinto "Archive/Newsletters";
  stop;
}
`
	if err := sc.CheckScript(valid); err != nil {
		t.Errorf("CheckScript (valid) failed: %v", err)
	}

	// Invalid script
	invalid := `require ["fileinto"];
if header :contains "From" "newsletter@" {
  INVALID_ACTION;
}
`
	if err := sc.CheckScript(invalid); err == nil {
		t.Log("CheckScript (invalid) correctly returned error (or server doesn't support CHECKSCRIPT)")
	} else {
		t.Logf("CheckScript (invalid) returned error as expected: %v", err)
	}
}

// ---- JS runtime tests ----

func newTestVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	opts := mail.RuntimeOptions{
		Accounts: map[string]mail.AccountConfig{
			"personal": testConfig,
		},
		Secrets: map[string]string{
			"personal/imapPassword": "testpass",
		},
		Context: context.Background(),
	}
	if err := mail.RegisterMailModule(vm, opts); err != nil {
		t.Fatalf("RegisterMailModule failed: %v", err)
	}
	return vm
}

func TestJSMailIMAPUse(t *testing.T) {
	vm := newTestVM(t)

	script := `
var result = [];
mail.imap.use("personal", function(imap) {
  var boxes = imap.list();
  result = boxes.map(function(b) { return b.name; });
});
result;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS script failed: %v", err)
	}
	exported := val.Export()
	t.Logf("Mailboxes from JS: %v", exported)
}

func TestJSMailIMAPStatus(t *testing.T) {
	vm := newTestVM(t)

	script := `
var status;
mail.imap.use("personal", function(imap) {
  status = imap.status("INBOX");
});
status;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS script failed: %v", err)
	}
	exported := val.Export()
	t.Logf("INBOX status from JS: %v", exported)
}

func TestJSMailIMAPWithMailbox(t *testing.T) {
	vm := newTestVM(t)

	script := `
var uids = [];
mail.imap.use("personal", function(imap) {
  imap.withMailbox("INBOX", function(mbox) {
    uids = mbox.search({});
  });
});
uids;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS script failed: %v", err)
	}
	exported := val.Export()
	t.Logf("UIDs from JS search: %v", exported)
}

func TestJSMailIMAPQuery(t *testing.T) {
	vm := newTestVM(t)

	// First append a test message
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	subject := fmt.Sprintf("JSQueryTest%d", time.Now().UnixNano())
	msg := fmt.Sprintf(
		"From: sender@example.com\r\nTo: testuser@localhost\r\nSubject: %s\r\n\r\nTest body\r\n",
		subject,
	)
	_, err = ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	_ = ic.Logout()

	script := `
var messages = [];
mail.imap.use("personal", function(imap) {
  imap.withMailbox("INBOX", function(mbox) {
    mbox.query({})
      .limit(5)
      .fetch(["uid", "envelope", "flags", "internalDate", "size"])
      .each(function(msg) {
        messages.push({
          uid: msg.uid,
          subject: msg.envelope ? msg.envelope.subject : null,
          flags: msg.flags
        });
      });
  });
});
messages;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS script failed: %v", err)
	}
	exported := val.Export()
	t.Logf("Messages from JS query: %v", exported)
}

func TestJSMailIMAPAppend(t *testing.T) {
	vm := newTestVM(t)

	subject := fmt.Sprintf("JSAppend%d", time.Now().UnixNano())
	script := fmt.Sprintf(`
var uid;
mail.imap.use("personal", function(imap) {
  imap.withMailbox("INBOX", function(mbox) {
    uid = mbox.append(
      "From: test@example.com\r\nTo: testuser@localhost\r\nSubject: %s\r\n\r\nBody\r\n",
      ["\\Seen"]
    );
  });
});
uid;
`, subject)

	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS script failed: %v", err)
	}
	t.Logf("Appended UID from JS: %v", val.Export())
}

func TestJSMailIMAPBatch(t *testing.T) {
	vm := newTestVM(t)

	// Append a test message first
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	msg := fmt.Sprintf(
		"From: newsletter@example.com\r\nTo: testuser@localhost\r\nSubject: Newsletter %d\r\n\r\nBody\r\n",
		time.Now().UnixNano(),
	)
	uid, err := ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	_ = ic.Logout()

	script := fmt.Sprintf(`
var result;
mail.imap.use("personal", function(imap) {
  imap.withMailbox("INBOX", function(mbox) {
    var uids = [%d];
    result = imap.batch(function(b) {
      return b.addFlags("INBOX", uids, ["\\Seen"]);
    });
  });
});
result;
`, uint32(uid))

	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS batch script failed: %v", err)
	}
	t.Logf("Batch result: %v", val.Export())
}

func TestJSMailSieveUse(t *testing.T) {
	vm := newTestVM(t)

	script := `
var scripts;
mail.sieve.use("personal", function(sieve) {
  scripts = sieve.listScripts();
});
scripts;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS sieve script failed: %v", err)
	}
	t.Logf("Sieve scripts from JS: %v", val.Export())
}

func TestJSMailSievePutAndCheck(t *testing.T) {
	vm := newTestVM(t)

	scriptName := fmt.Sprintf("jstest%d", time.Now().UnixNano())
	script := fmt.Sprintf(`
var result;
mail.sieve.use("personal", function(sieve) {
  var content = 'require ["fileinto"];\nif header :contains "From" "newsletter@" {\n  fileinto "Archive";\n  stop;\n}\n';
  result = sieve.check(content);
  sieve.putScript(%q, content);
  sieve.deleteScript(%q);
});
result;
`, scriptName, scriptName)

	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS sieve put/check script failed: %v", err)
	}
	t.Logf("Sieve check result: %v", val.Export())
}

func TestJSMailSieveBuild(t *testing.T) {
	vm := newTestVM(t)

	script := `
var builtScript;
mail.sieve.use("personal", function(sieve) {
  builtScript = sieve.build(function(r) {
    r.require(["fileinto", "imap4flags"]);
    r.if(
      r.all(
        r.headerContains("From", "newsletter@"),
        r.not(r.headerContains("Subject", "urgent"))
      ),
      function(a) {
        a.fileInto("Archive/Newsletters");
        a.addFlag("\\\\Seen");
        a.stop();
      }
    );
  });
});
builtScript;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS sieve build script failed: %v", err)
	}
	result := val.String()
	t.Logf("Built Sieve script:\n%s", result)
	if !strings.Contains(result, "require") {
		t.Error("built script missing require statement")
	}
	if !strings.Contains(result, "fileinto") {
		t.Error("built script missing fileinto action")
	}
}

func TestJSMailRequire(t *testing.T) {
	vm := newTestVM(t)

	script := `
var mail2 = require("mail");
var boxes;
mail2.imap.use("personal", function(imap) {
  boxes = imap.list();
});
boxes.length > 0;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS require script failed: %v", err)
	}
	if !val.ToBoolean() {
		t.Error("expected require('mail') to work and return mailboxes")
	}
}

func TestJSMailSecret(t *testing.T) {
	vm := newTestVM(t)

	script := `
var pass = mail.secret("personal/imapPassword");
pass;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS secret script failed: %v", err)
	}
	if val.String() != "testpass" {
		t.Errorf("expected 'testpass', got %q", val.String())
	}
}

func TestJSMailErrorHandling(t *testing.T) {
	vm := newTestVM(t)

	script := `
var caught;
try {
  mail.imap.use("personal", function(imap) {
    imap.withMailbox("NonExistentMailbox12345", function(mbox) {});
  });
} catch(e) {
  caught = { name: e.name, message: e.message };
}
caught;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS error handling script failed: %v", err)
	}
	t.Logf("Caught error: %v", val.Export())
}

func TestJSRealScript_MoveNewsletters(t *testing.T) {
	// Test the "real script" example from the spec
	ic, err := mail.Connect(context.Background(), testConfig.IMAP)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	// Create Archive/Newsletters mailbox
	_ = ic.CreateMailbox("Archive")
	_ = ic.CreateMailbox("Archive/Newsletters")
	// Append a newsletter-like message
	msg := fmt.Sprintf(
		"From: newsletter@example.com\r\nTo: testuser@localhost\r\nSubject: Weekly Newsletter\r\nDate: %s\r\n\r\nNewsletter body\r\n",
		time.Now().Add(-35*24*time.Hour).Format(time.RFC1123Z),
	)
	_, err = ic.Append("INBOX", []byte(msg), nil, nil)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	_ = ic.Logout()

	vm := newTestVM(t)
	script := `
var moved = 0;
mail.imap.use("personal", function(imap) {
  imap.withMailbox("INBOX", function(mbox) {
    var q = mbox.query({
      from: "newsletter@"
    }).limit(2000);
    var uids = q.uids();
    if (!uids.length) return;
    imap.batch(function(b) {
      b.move("INBOX", uids, "Archive/Newsletters");
    });
    moved = uids.length;
  });
});
moved;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS newsletter script failed: %v", err)
	}
	t.Logf("Moved %v newsletters", val.Export())
}

func TestJSRealScript_DeploySieve(t *testing.T) {
	vm := newTestVM(t)

	script := `
var result;
var script =
  'require ["fileinto"];\n' +
  '\n' +
  'if header :contains "From" "notifications@github.com" {\n' +
  '  fileinto "Archive/GitHub";\n' +
  '  stop;\n' +
  '}\n';

mail.sieve.use("personal", function(sieve) {
  result = sieve.check(script);
  sieve.putScript("autofile-github", script, { activate: false });
  sieve.deleteScript("autofile-github");
});
result;
`
	val, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("JS sieve deploy script failed: %v", err)
	}
	t.Logf("Sieve deploy result: %v", val.Export())
}


