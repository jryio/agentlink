package format

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jryio/agentlink/internal/agent"
)

// canonicalHeading is the instruction heading both peers converge on during
// comparison; translation rewrites it to the target's display name.
const canonicalHeading = "# Agent instructions"

// instructionsFormatter rewrites the canonical instruction heading to the
// target's display name. Any other heading passes through unchanged — free
// prose is never rewritten.
type instructionsFormatter struct{}

func (instructionsFormatter) Kind() string { return "instructions" }

func (instructionsFormatter) Format(canonical, _ []byte, target agent.Spec) ([]byte, []string, error) {
	text := NormalizeText(canonical)
	line, rest, found := strings.Cut(string(text), "\n")
	if found && line == canonicalHeading {
		text = []byte("# " + target.DisplayNames[0] + " instructions\n" + rest)
	}
	return text, nil, nil
}

// Canonicalize converges a native instructions document to the canonical
// form: the title heading and the source agent's own display names become
// the "Agent" token.
func (instructionsFormatter) Canonicalize(self agent.Spec, data []byte) ([]byte, error) {
	return ConvergeDisplayNames(NormalizeText(data), self), nil
}

// ConvergeDisplayNames rewrites the given specs' display names — in the
// title heading and in prose — to the canonical "Agent" token, longest names
// first so "Claude Code" replaces before "Claude". Display names only match
// complete tokens, so a short name such as "Pi" does not corrupt "Pipeline".
func ConvergeDisplayNames(data []byte, specs ...agent.Spec) []byte {
	names := displayNames(specs...)
	if len(names) == 0 {
		return data
	}
	quoted := make([]string, 0, len(names))
	for _, display := range names {
		quoted = append(quoted, regexp.QuoteMeta(display))
	}
	alternatives := strings.Join(quoted, "|")
	namePattern := regexp.MustCompile(alternatives)
	titlePattern := regexp.MustCompile("(?i:" + alternatives + ")")

	text := string(data)
	// Only the title line gets the wholesale heading rewrite; a multiline
	// match would destroy later H1s (e.g. "# Claude Code configuration")
	// that Format cannot reverse.
	title, rest, cut := strings.Cut(text, "\n")
	if strings.HasPrefix(title, "# ") && hasBoundedMatch(titlePattern, title) {
		title = canonicalHeading
	}
	if cut {
		text = title + "\n" + rest
	} else {
		text = title
	}
	return []byte(replaceBoundedMatches(namePattern, text, "Agent"))
}

func hasBoundedMatch(pattern *regexp.Regexp, text string) bool {
	for _, match := range pattern.FindAllStringIndex(text, -1) {
		if boundedMatch(text, match[0], match[1]) {
			return true
		}
	}
	return false
}

func replaceBoundedMatches(pattern *regexp.Regexp, text, replacement string) string {
	matches := pattern.FindAllStringIndex(text, -1)
	var out strings.Builder
	last := 0
	replaced := false
	for _, match := range matches {
		if !boundedMatch(text, match[0], match[1]) {
			continue
		}
		if !replaced {
			out.Grow(len(text))
		}
		out.WriteString(text[last:match[0]])
		out.WriteString(replacement)
		last = match[1]
		replaced = true
	}
	if !replaced {
		return text
	}
	out.WriteString(text[last:])
	return out.String()
}

func boundedMatch(text string, start, end int) bool {
	if start > 0 {
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		if unicode.IsLetter(before) || unicode.IsNumber(before) || before == '_' {
			return false
		}
	}
	if end < len(text) {
		after, _ := utf8.DecodeRuneInString(text[end:])
		if unicode.IsLetter(after) || unicode.IsNumber(after) || after == '_' {
			return false
		}
	}
	return true
}

// displayNames returns the specs' display names deduplicated, longest first.
func displayNames(specs ...agent.Spec) []string {
	seen := make(map[string]bool)
	var names []string
	for _, spec := range specs {
		for _, name := range spec.DisplayNames {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	slices.SortStableFunc(names, func(a, b string) int { return len(b) - len(a) })
	return names
}

func init() {
	Register(instructionsFormatter{})
}
