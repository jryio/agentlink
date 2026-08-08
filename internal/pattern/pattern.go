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

func match(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		if match(pattern[1:], name) {
			return true
		}
		return len(name) > 0 && match(pattern, name[1:])
	}
	if len(name) == 0 {
		return false
	}
	ok, err := path.Match(pattern[0], name[0])
	return err == nil && ok && match(pattern[1:], name[1:])
}
