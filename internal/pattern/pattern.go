// Package pattern matches portable slash-separated configuration globs.
package pattern

import (
	"fmt"
	"path"
	"strings"
)

// Set is a compiled collection of glob patterns. A double-star is supported
// as a complete path component and matches zero or more components.
type Set struct {
	patterns [][]string
}

// Compile validates patterns and returns a matcher.
func Compile(patterns []string) (Set, error) {
	set := Set{patterns: make([][]string, 0, len(patterns))}
	for _, raw := range patterns {
		clean := strings.TrimPrefix(path.Clean(raw), "./")
		parts := strings.Split(clean, "/")
		for _, part := range parts {
			if strings.Contains(part, "**") && part != "**" {
				return Set{}, fmt.Errorf("%q: ** must be a complete path component", raw)
			}
			if part != "**" {
				if _, err := path.Match(part, "probe"); err != nil {
					return Set{}, fmt.Errorf("%q: %w", raw, err)
				}
			}
		}
		set.patterns = append(set.patterns, parts)
	}
	return set, nil
}

// Match reports whether name matches any pattern in the set.
func (s Set) Match(name string) bool {
	name = strings.TrimPrefix(path.Clean(name), "./")
	parts := strings.Split(name, "/")
	for _, candidate := range s.patterns {
		if match(candidate, parts) {
			return true
		}
	}
	return false
}

// match reports whether pattern matches name. Each ** component forks into
// "advance the pattern" and "consume a path component", so overlapping
// subproblems recur. Memoizing on (pattern index, name index) collapses that
// branching into polynomial work bounded by len(pattern) * len(name), instead
// of re-solving shared states for every walked entry (CWE-400).
func match(pattern, name []string) bool {
	// memo[i][j] is the cached result of match(pattern[i:], name[j:]):
	// -1 unknown, 0 false, 1 true.
	memo := make([][]int8, len(pattern)+1)
	for i := range memo {
		memo[i] = make([]int8, len(name)+1)
		for j := range memo[i] {
			memo[i][j] = -1
		}
	}
	var rec func(i, j int) bool
	rec = func(i, j int) bool {
		if cached := memo[i][j]; cached != -1 {
			return cached == 1
		}
		var result bool
		switch {
		case i == len(pattern):
			result = j == len(name)
		case pattern[i] == "**":
			result = rec(i+1, j) || (j < len(name) && rec(i, j+1))
		case j == len(name):
			result = false
		default:
			ok, err := path.Match(pattern[i], name[j])
			result = err == nil && ok && rec(i+1, j+1)
		}
		if result {
			memo[i][j] = 1
		} else {
			memo[i][j] = 0
		}
		return result
	}
	return rec(0, 0)
}
