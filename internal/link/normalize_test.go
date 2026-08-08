package link

import (
	"bytes"
	"strings"
	"testing"
)

func TestNormalizePresets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		preset string
		left   string
		right  string
	}{
		{"exact", "exact", "same\r\n", "same\r\n"},
		{"text line endings", "text", "same  \r\n", "same\n"},
		{"instruction titles", "instructions", "# Claude Code rules\nUse Claude Code and Codex.\n", "# Codex rules\nUse Claude and Codex.\n"},
		{"skill tool frontmatter", "skill", "---\nname: x\nallowed-tools:\n  - Bash\n---\nBody\n", "---\nname: x\n---\nBody\n"},
		{"hook agent command", "hook", "agentlink remind-claude --agent claude\n", "agentlink remind-codex --agent codex\n"},
		{"presence", "presence", "anything", "different"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			left, err := normalize(test.preset, []byte(test.left))
			if err != nil {
				t.Fatalf("normalize(%q, left): %v", test.preset, err)
			}
			right, err := normalize(test.preset, []byte(test.right))
			if err != nil {
				t.Fatalf("normalize(%q, right): %v", test.preset, err)
			}
			if !bytes.Equal(left, right) {
				t.Errorf("normalized values differ:\nleft  %q\nright %q", left, right)
			}
		})
	}
}

func TestNormalizeFailures(t *testing.T) {
	t.Parallel()

	if _, err := normalize("text", []byte{'a', 0, 'b'}); err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("normalize(binary text) error = %v", err)
	}
	if _, err := normalize("unknown", []byte("text")); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("normalize(unknown) error = %v", err)
	}
	if _, err := normalize("skill", []byte("---\ninvalid: [\n---\nbody")); err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("normalize(invalid skill) error = %v", err)
	}
}
