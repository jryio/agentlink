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
		Peers: map[string]config.MCPPeer{
			"claude": {Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
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

	plan, err := engine.PlanSync(t.Context(), Side("claude"), false, nil)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Pair != "instructions" {
		t.Fatalf("PlanSync() = %+v, want only the artifact operation", plan)
	}
	if len(plan.Unresolved) != 0 {
		t.Fatalf("PlanSync().Unresolved = %+v, want none from MCP checks", plan.Unresolved)
	}

	empty, err := engine.PlanSync(t.Context(), Side("claude"), false, map[string]bool{"mcp": true})
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

	plan, err := engine.PlanSync(t.Context(), Side("claude"), false, nil)
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

	plan, err := engine.PlanSync(t.Context(), Side("claude"), false, map[string]bool{"skills": true})
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 0 || len(plan.Unresolved) != 1 || !strings.Contains(plan.Unresolved[0].Detail, "non-regular source") {
		t.Fatalf("PlanSync() = %+v, want symlink copy blocked in preview", plan)
	}
}

func TestTranslatePreservesExistingTargetMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".agents", "hooks.json"), `{"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "x"}]}]}
`)
	settings := filepath.Join(dir, ".claude", "settings.json")
	writeTestFile(t, settings, `{
  "model": "opus",
  "permissions": {"allow": ["Bash(ls:*)"]}
}
`)
	if err := os.Chmod(settings, 0o600); err != nil {
		t.Fatalf("os.Chmod(settings): %v", err)
	}
	doc := testDocument(dir)
	doc.Config.Pairs = []config.Pair{{
		ID: "hooks", Kind: "file", Normalizer: "hook", Sync: "translate",
		Peers: map[string]config.Endpoint{
			"agents": {Source: "workspace", Path: ".agents/hooks.json"},
			"claude": {Source: "workspace", Path: ".claude/settings.json"},
		},
	}}
	engine, closeEngine := newEngine(t, doc)
	t.Cleanup(closeEngine)

	plan, err := engine.PlanSync(t.Context(), Side("agents"), false, nil)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Transform != "hook" {
		t.Fatalf("PlanSync() = %+v, want one translate operation", plan)
	}
	if err := engine.Apply(t.Context(), plan); err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() { _ = roots.Close() })
	workspace, err := roots.Root("workspace")
	if err != nil {
		t.Fatalf("Root(workspace): %v", err)
	}
	data, _, err := workspace.ReadFile(".claude/settings.json", 1<<20)
	if err != nil {
		t.Fatalf("ReadFile(settings): %v", err)
	}
	if !strings.Contains(string(data), `"model": "opus"`) || !strings.Contains(string(data), `"PreToolUse"`) {
		t.Fatalf("merged settings = %s, want model preserved and hooks merged", data)
	}
	info, err := os.Stat(settings)
	if err != nil {
		t.Fatalf("os.Stat(settings): %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("settings mode = %o, want 0600 preserved from the existing file", got)
	}
	if report := engine.Check(t.Context(), nil); !report.Clean() {
		t.Fatalf("post-translate Check() = %+v, want clean", report)
	}
}

func TestTranslateBlocksOnUnreadableExistingTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".agents", "hooks.json"), "{}\n")
	writeTestFile(t, filepath.Join(dir, ".claude", "settings.json"), `{"model": "opus", "note": "long enough to exceed the limit"}
`)
	doc := testDocument(dir)
	doc.Config.Limits.MaxFileSize = 16 // source fits, existing target does not
	doc.Config.Pairs = []config.Pair{{
		ID: "hooks", Kind: "file", Normalizer: "hook", Sync: "translate",
		Peers: map[string]config.Endpoint{
			"agents": {Source: "workspace", Path: ".agents/hooks.json"},
			"claude": {Source: "workspace", Path: ".claude/settings.json"},
		},
	}}
	engine, closeEngine := newEngine(t, doc)
	t.Cleanup(closeEngine)

	// Drive operationFor directly: Check would report the oversize target as a
	// read error first; this exercises the plan-time translate read path.
	pair := doc.Config.Pairs[0]
	finding := Finding{Pair: pair.ID, Relative: ".", State: StateDifferent}
	_, actionable, reason := engine.operationFor(pair, finding, Side("agents"), false)
	if actionable || !strings.Contains(reason, "read translate target") {
		t.Fatalf("operationFor() = %v, %q, want blocked with read translate target", actionable, reason)
	}
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() { _ = roots.Close() })
	workspace, err := roots.Root("workspace")
	if err != nil {
		t.Fatalf("Root(workspace): %v", err)
	}
	data, _, err := workspace.ReadFile(".claude/settings.json", 1<<20)
	if err != nil {
		t.Fatalf("ReadFile(settings): %v", err)
	}
	if !strings.Contains(string(data), `"model": "opus"`) {
		t.Fatalf("settings was clobbered: %s", data)
	}
}

func TestPlanSyncRejectsDuplicateTargets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "one.md"), "one\n")
	writeTestFile(t, filepath.Join(dir, "claude", "two.md"), "two\n")
	doc := testDocument(dir)
	doc.Config.Pairs = []config.Pair{
		{
			ID: "one", Kind: "file", Normalizer: "exact", Sync: "copy",
			Peers: map[string]config.Endpoint{
				"claude": {Source: "workspace", Path: "claude/one.md"},
				"codex":  {Source: "workspace", Path: "codex/shared.md"},
			},
		},
		{
			ID: "two", Kind: "file", Normalizer: "exact", Sync: "copy",
			Peers: map[string]config.Endpoint{
				"claude": {Source: "workspace", Path: "claude/two.md"},
				"codex":  {Source: "workspace", Path: "codex/shared.md"},
			},
		},
	}
	engine, closeEngine := newEngine(t, doc)
	t.Cleanup(closeEngine)

	_, err := engine.PlanSync(t.Context(), Side("claude"), false, nil)
	if err == nil || !strings.Contains(err.Error(), "target the same path") {
		t.Fatalf("PlanSync() error = %v, want duplicate target rejection", err)
	}
}
