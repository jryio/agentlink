// Package format translates canonical .agents artifacts into each registered
// agent's native on-disk shape. Variation is data-driven from
// internal/agent.Spec; this package holds only the mechanics per artifact
// kind (skill frontmatter, instruction headings, hook documents).
package format

import (
	"fmt"

	"github.com/jryio/agentlink/internal/agent"
)

// Formatter translates a canonical .agents artifact into one target agent's
// native shape. existing carries the current target file bytes (nil when the
// file does not exist) so formats embedded in larger documents can merge
// instead of replace. Warnings report dropped events or keys a target cannot
// express; translation never fabricates values.
type Formatter interface {
	Kind() string
	Format(canonical, existing []byte, target agent.Spec) (out []byte, warnings []string, err error)
}

var formatters = map[string]Formatter{}

// Register adds a formatter; it panics on a duplicate kind. Called from
// init() in the per-kind files of this package.
func Register(f Formatter) {
	if _, dup := formatters[f.Kind()]; dup {
		panic(fmt.Sprintf("format: duplicate formatter for kind %q", f.Kind()))
	}
	formatters[f.Kind()] = f
}

// For returns the formatter registered for kind.
func For(kind string) (Formatter, bool) {
	f, ok := formatters[kind]
	return f, ok
}
