package format

import (
	"strings"

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

func init() {
	Register(instructionsFormatter{})
}
