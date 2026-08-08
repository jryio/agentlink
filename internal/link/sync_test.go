package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func TestPlanSyncChecksOnlyArtifactPairs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "# Instructions\n\nNew.\n")
	writeTestFile(t, filepath.Join(dir, "codex", "AGENTS.md"), "# Instructions\n\nOld.\n")
	doc := testDocument(dir)
	doc.Config.Pairs[0].Sync = "copy"
	doc.Config.Pairs[1].Optional = true
	doc.Config.MCPServers = []config.MCPServer{{
		ID: "mcp",
		Claude: config.MCPPeer{
			Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"},
			Server: "tasks",
		},
		Codex: config.MCPPeer{
			Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"},
			Server: "tasks",
		},
	}}
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

	plan, err := engine.PlanSync(t.Context(), SideClaude, false, nil)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Pair != "instructions" {
		t.Fatalf("PlanSync() = %+v, want only the artifact operation", plan)
	}
	if len(plan.Unresolved) != 0 {
		t.Fatalf("PlanSync().Unresolved = %+v, want none from MCP checks", plan.Unresolved)
	}

	empty, err := engine.PlanSync(t.Context(), SideClaude, false, map[string]bool{"mcp": true})
	if err != nil {
		t.Fatalf("PlanSync(MCP selection): %v", err)
	}
	if len(empty.Operations) != 0 || len(empty.Unresolved) != 0 {
		t.Fatalf("PlanSync(MCP selection) = %+v, want empty plan", empty)
	}
}

func TestSingleFileReportCountsMissingPeer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "# Instructions\n")
	engine, closeEngine := newTestEngine(t, dir)
	t.Cleanup(closeEngine)

	report := engine.Check(t.Context(), map[string]bool{"instructions": true})
	if len(report.Pairs) != 1 || report.Pairs[0].Files != 1 {
		t.Fatalf("Check().Pairs = %+v, want one compared file", report.Pairs)
	}
}

func TestSyncDoesNotPlanUnreadableSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "too large")
	doc := testDocument(dir)
	doc.Config.Limits.MaxFileSize = 2
	doc.Config.Pairs[0].Sync = "copy"
	doc.Config.Pairs[1].Optional = true
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

	plan, err := engine.PlanSync(t.Context(), SideClaude, false, nil)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 0 || len(plan.Unresolved) != 1 || plan.Unresolved[0].State != StateError {
		t.Fatalf("PlanSync() = %+v, want unreadable source left unresolved", plan)
	}
}

func TestSyncPreviewBlocksSymlinkCopies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "shared.txt"), "shared\n")
	if err := os.MkdirAll(filepath.Join(dir, "claude", "skills"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(skills): %v", err)
	}
	if err := os.Symlink("../shared.txt", filepath.Join(dir, "claude", "skills", "linked.txt")); err != nil {
		t.Skipf("os.Symlink() unavailable: %v", err)
	}
	doc := testDocument(dir)
	doc.Config.Pairs[0].Optional = true
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

	plan, err := engine.PlanSync(t.Context(), SideClaude, false, map[string]bool{"skills": true})
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 0 || len(plan.Unresolved) != 1 || !strings.Contains(plan.Unresolved[0].Detail, "non-regular source") {
		t.Fatalf("PlanSync() = %+v, want symlink copy blocked in preview", plan)
	}
}
