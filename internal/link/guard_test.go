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

func TestGuardAllowsUnmanagedBasePairPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	review := "---\nname: review\ndescription: Shared review\n---\nBody.\n"
	writeTestFile(t, filepath.Join(dir, "shared", "skills", "review", "SKILL.md"), review)
	writeTestFile(t, filepath.Join(dir, "repo", "skills", "review", "SKILL.md"), review)
	writeTestFile(t, filepath.Join(dir, "repo", "skills", "local", "SKILL.md"), "---\nname: local\ndescription: Local review\n---\nBody.\n")
	engine, closeEngine := newEngine(t, basePairDocument(dir))
	t.Cleanup(closeEngine)

	violations, err := engine.Guard(t.Context(), []string{"repo/skills/local/SKILL.md"})
	if err != nil {
		t.Fatalf("Guard(unmanaged local skill): %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("Guard(unmanaged local skill) = %+v, want no violations", violations)
	}

	writeTestFile(t, filepath.Join(dir, "repo", "skills", "review", "SKILL.md"), review+"Changed.\n")
	violations, err = engine.Guard(t.Context(), []string{"repo/skills/review/SKILL.md"})
	if err != nil {
		t.Fatalf("Guard(modified base skill): %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("Guard(modified base skill) = %+v, want one violation", violations)
	}
}

func TestEndpointRelativeIncludesTreeRoot(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "skills")
	if relative, ok := endpointRelative(base, base, "tree"); !ok || relative != "." {
		t.Fatalf("endpointRelative(root) = %q, %v, want root match", relative, ok)
	}
}
