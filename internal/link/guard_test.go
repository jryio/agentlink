package link

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardDifferentPeerNamesBothCounterparts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "# Instructions\n\nShared body.\n")
	writeTestFile(t, filepath.Join(dir, "codex", "AGENTS.md"), "# Instructions\n\nRewritten body.\n")
	engine, closeEngine := newTestEngine(t, dir)
	t.Cleanup(closeEngine)

	violations, err := engine.Guard(t.Context(), []string{"codex/AGENTS.md"})
	if err != nil {
		t.Fatalf("Guard(): %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("Guard() = %+v, want one violation", violations)
	}
	counterpart := violations[0].Counterpart
	if !strings.Contains(counterpart, "CLAUDE.md") || !strings.Contains(counterpart, "AGENTS.md") {
		t.Fatalf("Guard().Counterpart = %q, want both peer paths", counterpart)
	}
}

func TestEndpointRelativeIncludesTreeRoot(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "skills")
	if relative, ok := endpointRelative(base, base, "tree"); !ok || relative != "." {
		t.Fatalf("endpointRelative(root) = %q, %v, want root match", relative, ok)
	}
}
