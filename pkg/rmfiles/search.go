package rmfiles

import (
	"sort"
	"strings"
)

// SearchResult pairs an entry with its relevance score.
type SearchResult struct {
	Entry Entry `json:"entry"`
	Score int   `json:"score"`
}

// Search ranks entries against a query. An empty query returns every entry
// (capped by limit, sorted by path). Scoring prefers name matches over path
// matches and rewards prefix, word-start, and consecutive-character matches.
func Search(entries []Entry, query string, limit int) []SearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 100
	}
	if q == "" {
		out := make([]SearchResult, 0, len(entries))
		for _, e := range entries {
			out = append(out, SearchResult{Entry: e})
		}
		if len(out) > limit {
			out = out[:limit]
		}
		return out
	}

	results := make([]SearchResult, 0, 32)
	for _, e := range entries {
		score, ok := scoreEntry(q, e)
		if !ok {
			continue
		}
		results = append(results, SearchResult{Entry: e, Score: score})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Entry.Name != results[j].Entry.Name {
			return strings.ToLower(results[i].Entry.Name) < strings.ToLower(results[j].Entry.Name)
		}
		return results[i].Entry.Path < results[j].Entry.Path
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func scoreEntry(q string, e Entry) (int, bool) {
	name := strings.ToLower(e.Name)
	path := strings.ToLower(e.Path)
	dirBonus := 0
	if e.IsDir {
		dirBonus = 5
	}
	if name == q {
		return 1000 + dirBonus, true
	}
	if strings.HasPrefix(name, q) {
		return 800 + len(q)*4 + dirBonus, true
	}
	if s, ok := fuzzySubsequence(q, name, pathWordStart(name)); ok {
		return 400 + s + dirBonus, true
	}
	if s, ok := fuzzySubsequence(q, path, nil); ok {
		return 150 + s, true
	}
	return 0, false
}

// prefixFn reports whether the byte at index i begins a word in s.
type prefixFn func(i int) bool

func pathWordStart(s string) prefixFn {
	return func(i int) bool {
		if i == 0 {
			return true
		}
		c := s[i-1]
		return c == '/' || c == ' ' || c == '-' || c == '_' || c == '.'
	}
}

// fuzzySubsequence greedily matches q as a subsequence of s. Consecutive
// matches and word-start matches score higher. Returns the score and whether
// q matched.
func fuzzySubsequence(q, s string, isWordStart prefixFn) (int, bool) {
	if q == "" {
		return 0, true
	}
	score := 0
	streak := 0
	pos := 0
	for i := 0; i < len(s) && pos < len(q); i++ {
		if s[i] != q[pos] {
			streak = 0
			continue
		}
		score += 10
		if streak > 0 {
			score += streak * 5
		}
		if isWordStart != nil && isWordStart(i) {
			score += 15
		}
		streak++
		pos++
	}
	if pos != len(q) {
		return 0, false
	}
	// Shorter haystacks are better matches.
	score -= len(s) / 20
	if score < 0 {
		score = 0
	}
	return score, true
}
