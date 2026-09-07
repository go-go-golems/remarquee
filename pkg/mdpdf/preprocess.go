package mdpdf

import "strings"

// StripYAMLFrontmatter removes a leading YAML frontmatter block delimited by:
//
//	---\n
//	...yaml...
//	---\n
//
// If no such block exists, it returns the input unchanged.
//
// This is intentionally simple and matches docmgr's strict delimiter logic.
// It also avoids pandoc failing on invalid YAML frontmatter and prevents
// frontmatter metadata from appearing in rendered PDFs.
func StripYAMLFrontmatter(mdText string) string {
	if !strings.HasPrefix(mdText, "---") {
		return mdText
	}

	lines := strings.Split(mdText, "\n")
	if len(lines) == 0 {
		return mdText
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return mdText
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		return mdText
	}

	body := strings.Join(lines[endIdx+1:], "\n")
	return strings.TrimLeft(body, "\n")
}

func isListItemLine(line string) bool {
	stripped := strings.TrimLeft(line, " \t")
	if stripped == "" {
		return false
	}

	// Unordered list: starts with -, *, or + followed by space.
	if len(stripped) >= 2 {
		switch stripped[0] {
		case '-', '*', '+':
			return stripped[1] == ' '
		}
	}

	// Ordered list: starts with digit(s) followed by ". " (e.g. "1. " or "12. ").
	if stripped[0] < '0' || stripped[0] > '9' {
		return false
	}
	j := 1
	for j < len(stripped) && stripped[j] >= '0' && stripped[j] <= '9' {
		j++
	}
	if j < len(stripped) && stripped[j] == '.' {
		if j+1 < len(stripped) && stripped[j+1] == ' ' {
			return true
		}
	}

	return false
}

// NormalizeListSpacing ensures proper spacing before bullet/numbered lists so pandoc
// recognizes them. It inserts a blank line before list items that are not preceded by
// a blank line (and where the previous line is not itself a list item).
// Fenced code and line-start display math are preserved verbatim.
func NormalizeListSpacing(mdText string) string {
	lines := strings.Split(mdText, "\n")
	if len(lines) == 0 {
		return mdText
	}

	protected := literalLines(lines)
	out := make([]string, 0, len(lines)+16)
	for i, line := range lines {
		isList := !protected[i] && isListItemLine(line)
		if isList && i > 0 {
			prevLine := strings.TrimSpace(lines[i-1])
			prevIsList := false
			if prevLine != "" && !protected[i-1] {
				prevIsList = isListItemLine(lines[i-1])
			}
			if prevLine != "" && !prevIsList {
				out = append(out, "")
			}
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// FlattenDeepLists caps markdown list nesting at maxDepth levels.
// LaTeX only supports 4 levels of list nesting by default, so deeply nested
// lists (e.g. from accessibility tree dumps) cause "Too deeply nested" errors.
// Items beyond maxDepth are re-indented to the maximum depth, merging them
// into the deepest valid list level. Fenced code and line-start display math
// are preserved verbatim.
func FlattenDeepLists(mdText string, maxDepth int) string {
	if maxDepth < 1 {
		maxDepth = 1
	}

	lines := strings.Split(mdText, "\n")
	protected := literalLines(lines)
	out := make([]string, 0, len(lines))

	for i, line := range lines {
		if protected[i] {
			out = append(out, line)
			continue
		}
		indent := listIndentLevel(line)
		if indent > maxDepth {
			// Re-indent this line to maxDepth
			stripped := strings.TrimLeft(line, " \t")
			newIndent := strings.Repeat("  ", maxDepth-1)
			out = append(out, newIndent+stripped)
		} else {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}

// listIndentLevel returns the nesting depth of a list item line (1-based).
// Returns 0 for non-list lines.
func listIndentLevel(line string) int {
	if !isListItemLine(line) {
		return 0
	}
	// Count leading spaces/tabs to determine nesting.
	// Each 2 spaces or 1 tab = 1 level of nesting.
	spaces := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			spaces++
		case '\t':
			spaces += 2
		default:
			return (spaces / 2) + 1
		}
	}
	return (spaces / 2) + 1
}
