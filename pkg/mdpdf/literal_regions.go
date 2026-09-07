package mdpdf

import "strings"

// literalLines marks fenced code and line-start display math, including their
// delimiters. It is a guard for our line-based list rewrites, not a full Markdown
// parser. Unclosed regions remain protected through EOF.
func literalLines(lines []string) []bool {
	protected := make([]bool, len(lines))
	var fence byte
	var fenceLength int
	var mathEnd string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != 0 {
			protected[i] = true
			ch, n := fencePrefix(trimmed)
			if ch == fence && n >= fenceLength && strings.TrimSpace(trimmed[n:]) == "" {
				fence, fenceLength = 0, 0
			}
			continue
		}
		if mathEnd != "" {
			protected[i] = true
			if hasUnescapedDelimiter(line, mathEnd) {
				mathEnd = ""
			}
			continue
		}

		if ch, n := fencePrefix(trimmed); n >= 3 {
			// Backtick fence info strings cannot themselves contain backticks.
			if ch != '`' || !strings.Contains(trimmed[n:], "`") {
				protected[i] = true
				fence, fenceLength = ch, n
				continue
			}
		}
		var start, end string
		switch {
		case strings.HasPrefix(trimmed, "$$"):
			start, end = "$$", "$$"
		case strings.HasPrefix(trimmed, `\[`):
			start, end = `\[`, `\]`
		default:
			continue
		}
		protected[i] = true
		if !hasUnescapedDelimiter(trimmed[len(start):], end) {
			mathEnd = end
		}
	}
	return protected
}

func fencePrefix(line string) (byte, int) {
	if len(line) == 0 || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	i := 1
	for i < len(line) && line[i] == line[0] {
		i++
	}
	return line[0], i
}

func hasUnescapedDelimiter(line, delimiter string) bool {
	for offset := 0; offset < len(line); {
		relative := strings.Index(line[offset:], delimiter)
		if relative < 0 {
			return false
		}
		at := offset + relative
		backslashes := 0
		for j := at - 1; j >= 0 && line[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return true
		}
		offset = at + len(delimiter)
	}
	return false
}
