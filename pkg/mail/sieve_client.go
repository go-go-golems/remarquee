package mail

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// SieveOptions holds connection parameters for a ManageSieve server.
type SieveOptions struct {
	Host     string
	Port     int
	Username string
	Password string
}

// ScriptInfo describes a Sieve script on the server.
type ScriptInfo struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// SieveCapabilities holds server capabilities.
type SieveCapabilities struct {
	Implementation string   `json:"implementation"`
	Sieve          []string `json:"sieve"`
	Notify         []string `json:"notify"`
	SASL           []string `json:"sasl"`
	StartTLS       bool     `json:"starttls"`
	Version        string   `json:"version"`
}

// SieveClient is a minimal ManageSieve protocol client.
type SieveClient struct {
	conn net.Conn
	r    *bufio.Reader
	caps SieveCapabilities
}

// ConnectSieve opens a ManageSieve connection and authenticates.
func ConnectSieve(opts SieveOptions) (*SieveClient, error) {
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	log.Debug().Str("addr", addr).Msg("connecting to ManageSieve")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "dial ManageSieve")
	}

	sc := &SieveClient{
		conn: conn,
		r:    bufio.NewReader(conn),
	}

	// Read greeting / capabilities
	if err := sc.readCapabilities(); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "reading greeting")
	}

	// Authenticate with PLAIN
	authStr := base64.StdEncoding.EncodeToString(
		[]byte("\x00" + opts.Username + "\x00" + opts.Password),
	)
	if err := sc.sendLine(fmt.Sprintf("AUTHENTICATE \"PLAIN\" %q", authStr)); err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "send AUTHENTICATE")
	}
	if err := sc.expectOK(); err != nil {
		_ = conn.Close()
		return nil, &MailError{Name: "AuthError", Message: err.Error(), Source: "sieve"}
	}

	return sc, nil
}

// Capabilities returns the server capabilities.
func (sc *SieveClient) Capabilities() SieveCapabilities {
	return sc.caps
}

// ListScripts returns all scripts on the server.
func (sc *SieveClient) ListScripts() ([]ScriptInfo, error) {
	if err := sc.sendLine("LISTSCRIPTS"); err != nil {
		return nil, errors.Wrap(err, "LISTSCRIPTS")
	}
	var scripts []ScriptInfo
	for {
		line, err := sc.readLine()
		if err != nil {
			return nil, errors.Wrap(err, "reading LISTSCRIPTS response")
		}
		if isOK(line) {
			break
		}
		if isNO(line) {
			return nil, parseSieveError(line)
		}
		// Parse: "scriptname" [ACTIVE]
		name, active := parseScriptLine(line)
		if name != "" {
			scripts = append(scripts, ScriptInfo{Name: name, Active: active})
		}
	}
	return scripts, nil
}

// GetScript retrieves a script's content.
func (sc *SieveClient) GetScript(name string) (string, error) {
	if err := sc.sendLine(fmt.Sprintf("GETSCRIPT %q", name)); err != nil {
		return "", errors.Wrap(err, "GETSCRIPT")
	}
	// Response: {size}\r\n<content>\r\nOK
	content, err := sc.readLiteral()
	if err != nil {
		return "", errors.Wrap(err, "reading script literal")
	}
	if err := sc.expectOK(); err != nil {
		return "", errors.Wrap(err, "GETSCRIPT OK")
	}
	return content, nil
}

// PutScript uploads a script. If activate is true, activates it.
func (sc *SieveClient) PutScript(name, content string, activate bool) error {
	// RFC 5804: literal must be followed by CRLF after the content bytes
	cmd := fmt.Sprintf("PUTSCRIPT %q {%d+}\r\n%s\r\n", name, len(content), content)
	if err := sc.sendRaw(cmd); err != nil {
		return errors.Wrap(err, "PUTSCRIPT")
	}
	if err := sc.expectOK(); err != nil {
		return errors.Wrap(err, "PUTSCRIPT response")
	}
	if activate {
		return sc.Activate(name)
	}
	return nil
}

// Activate sets the active script.
func (sc *SieveClient) Activate(name string) error {
	if err := sc.sendLine(fmt.Sprintf("SETACTIVE %q", name)); err != nil {
		return errors.Wrap(err, "SETACTIVE")
	}
	return sc.expectOK()
}

// Deactivate deactivates all scripts.
func (sc *SieveClient) Deactivate() error {
	if err := sc.sendLine(`SETACTIVE ""`); err != nil {
		return errors.Wrap(err, "SETACTIVE (deactivate)")
	}
	return sc.expectOK()
}

// DeleteScript deletes a script.
func (sc *SieveClient) DeleteScript(name string) error {
	if err := sc.sendLine(fmt.Sprintf("DELETESCRIPT %q", name)); err != nil {
		return errors.Wrap(err, "DELETESCRIPT")
	}
	return sc.expectOK()
}

// RenameScript renames a script.
func (sc *SieveClient) RenameScript(oldName, newName string) error {
	if err := sc.sendLine(fmt.Sprintf("RENAMESCRIPT %q %q", oldName, newName)); err != nil {
		return errors.Wrap(err, "RENAMESCRIPT")
	}
	return sc.expectOK()
}

// CheckScript validates a script without storing it.
func (sc *SieveClient) CheckScript(content string) error {
	// RFC 5804: literal must be followed by CRLF after the content bytes
	cmd := fmt.Sprintf("CHECKSCRIPT {%d+}\r\n%s\r\n", len(content), content)
	if err := sc.sendRaw(cmd); err != nil {
		return errors.Wrap(err, "CHECKSCRIPT")
	}
	line, err := sc.readLine()
	if err != nil {
		return errors.Wrap(err, "CHECKSCRIPT response")
	}
	if isOK(line) {
		return nil
	}
	return parseSieveError(line)
}

// HaveSpace checks if the server has space for a script.
func (sc *SieveClient) HaveSpace(name string, sizeBytes int) (bool, error) {
	if err := sc.sendLine(fmt.Sprintf("HAVESPACE %q %d", name, sizeBytes)); err != nil {
		return false, errors.Wrap(err, "HAVESPACE")
	}
	line, err := sc.readLine()
	if err != nil {
		return false, errors.Wrap(err, "HAVESPACE response")
	}
	return isOK(line), nil
}

// Logout closes the connection.
func (sc *SieveClient) Logout() error {
	_ = sc.sendLine("LOGOUT")
	return sc.conn.Close()
}

// ---- low-level helpers ----

func (sc *SieveClient) sendLine(s string) error {
	return sc.sendRaw(s + "\r\n")
}

func (sc *SieveClient) sendRaw(s string) error {
	log.Trace().Str("cmd", s).Msg("sieve send")
	_, err := fmt.Fprint(sc.conn, s)
	return err
}

func (sc *SieveClient) readLine() (string, error) {
	line, err := sc.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	log.Trace().Str("line", line).Msg("sieve recv")
	return line, nil
}

func (sc *SieveClient) expectOK() error {
	line, err := sc.readLine()
	if err != nil {
		return err
	}
	if isOK(line) {
		return nil
	}
	return parseSieveError(line)
}

func isOK(line string) bool {
	return strings.HasPrefix(strings.ToUpper(line), "OK")
}

func isNO(line string) bool {
	return strings.HasPrefix(strings.ToUpper(line), "NO") ||
		strings.HasPrefix(strings.ToUpper(line), "BYE")
}

func parseSieveError(line string) error {
	return &MailError{
		Name:    "SieveError",
		Message: line,
		Source:  "sieve",
	}
}

func (sc *SieveClient) readCapabilities() error {
	for {
		line, err := sc.readLine()
		if err != nil {
			return err
		}
		if isOK(line) {
			break
		}
		if isNO(line) {
			return fmt.Errorf("server error: %s", line)
		}
		sc.parseCapLine(line)
	}
	return nil
}

func (sc *SieveClient) parseCapLine(line string) {
	// Format: "KEY" "VALUE"  or  "KEY"
	parts := splitQuoted(line)
	if len(parts) == 0 {
		return
	}
	key := strings.ToUpper(parts[0])
	val := ""
	if len(parts) > 1 {
		val = parts[1]
	}
	switch key {
	case "IMPLEMENTATION":
		sc.caps.Implementation = val
	case "SIEVE":
		sc.caps.Sieve = strings.Fields(val)
	case "NOTIFY":
		sc.caps.Notify = strings.Fields(val)
	case "SASL":
		sc.caps.SASL = strings.Fields(val)
	case "STARTTLS":
		sc.caps.StartTLS = true
	case "VERSION":
		sc.caps.Version = val
	}
}

func (sc *SieveClient) readLiteral() (string, error) {
	// Expect: {size}\r\n<content>
	line, err := sc.readLine()
	if err != nil {
		return "", err
	}
	if isNO(line) {
		return "", parseSieveError(line)
	}
	// line should be like {123}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		return "", fmt.Errorf("expected literal size, got: %s", line)
	}
	sizeStr := line[1 : len(line)-1]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return "", fmt.Errorf("invalid literal size: %s", sizeStr)
	}
	buf := make([]byte, size)
	if _, err := sc.r.Read(buf); err != nil {
		return "", errors.Wrap(err, "reading literal content")
	}
	// consume trailing \r\n
	_, _ = sc.readLine()
	return string(buf), nil
}

func parseScriptLine(line string) (name string, active bool) {
	// Format: "scriptname" [ACTIVE]
	parts := splitQuoted(line)
	if len(parts) == 0 {
		return "", false
	}
	name = parts[0]
	for _, p := range parts[1:] {
		if strings.EqualFold(p, "ACTIVE") {
			active = true
		}
	}
	return name, active
}

// splitQuoted splits a line by spaces, respecting double-quoted strings.
func splitQuoted(s string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == ' ' && !inQuote {
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}
