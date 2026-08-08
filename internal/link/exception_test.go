package link

import (
	"path/filepath"
	"testing"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func TestFilePairRootExceptionSuppressesDrift(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "Claude body.\n")
	writeTestFile(t, filepath.Join(dir, "codex", "AGENTS.md"), "Codex body.\n")
	doc := testDocument(dir)
	doc.Config.Exceptions = []config.Exception{{Pair: "instructions", Paths: []string{"."}, Reason: "intentionally different"}}
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() {
		if err := roots.Close(); err != nil {
			t.Errorf("roots.Close(): %v", err)
		}
	})
	engine, err := New(doc, roots)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	report := engine.Check(t.Context(), map[string]bool{"instructions": true})
	if !report.Clean() || len(report.Pairs) != 1 || !report.Pairs[0].Skipped {
		t.Fatalf("Check() = %+v, want intentionally excepted file pair skipped", report)
	}
}
