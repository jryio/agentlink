package link

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func TestEngineEndToEnd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "CLAUDE.md"), "# Claude instructions\n\nKeep peers aligned.\n")
	writeTestFile(t, filepath.Join(dir, "codex", "AGENTS.md"), "# Codex instructions\n\nKeep peers aligned.\n")
	writeTestFile(t, filepath.Join(dir, "claude", "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: Test.\nallowed-tools:\n  - Bash\n---\n\n# Alpha\n\nDo it.\n")
	writeTestFile(t, filepath.Join(dir, "codex", "skills", "alpha", "SKILL.md"), "---\ndescription: Test.\nname: alpha\nallowed-tools: [Bash]\n---\n\n# Alpha\n\nDo it.\n")

	engine, closeEngine := newTestEngine(t, dir)
	t.Cleanup(closeEngine)

	report := engine.Check(t.Context(), nil)
	if !report.Clean() {
		t.Fatalf("initial Check() findings = %v, want clean", report.Pairs)
	}

	writeTestFile(t, filepath.Join(dir, "claude", "skills", "alpha", "SKILL.md"), "---\nname: alpha\ndescription: Test.\n---\n\n# Alpha\n\nDo it better.\n")
	report = engine.Check(t.Context(), nil)
	if report.FindingCount() != 1 || report.Pairs[1].Findings[0].State != StateDifferent {
		t.Fatalf("drifting Check() = %+v, want one different finding", report)
	}

	violations, err := engine.Guard(t.Context(), []string{"claude/skills/alpha/SKILL.md"})
	if err != nil {
		t.Fatalf("Guard(): %v", err)
	}
	if len(violations) != 1 || violations[0].Pair != "skills" {
		t.Fatalf("Guard() = %+v, want skills violation", violations)
	}

	plan, err := engine.PlanSync(t.Context(), Side("claude"), false, nil)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if got, want := len(plan.Operations), 1; got != want {
		t.Fatalf("len(PlanSync().Operations) = %d, want %d", got, want)
	}
	if err := engine.Apply(t.Context(), plan); err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	if report = engine.Check(t.Context(), nil); !report.Clean() {
		t.Fatalf("post-Apply Check() = %+v, want clean", report)
	}

	writeTestFile(t, filepath.Join(dir, "codex", "skills", "codex-only", "SKILL.md"), "# Codex only\n")
	plan, err = engine.PlanSync(t.Context(), Side("claude"), false, nil)
	if err != nil {
		t.Fatalf("PlanSync(no prune): %v", err)
	}
	if len(plan.Operations) != 0 || len(plan.Unresolved) != 1 {
		t.Fatalf("PlanSync(no prune) = %+v, want one unresolved", plan)
	}
	plan, err = engine.PlanSync(t.Context(), Side("claude"), true, nil)
	if err != nil {
		t.Fatalf("PlanSync(prune): %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Kind != OperationDelete {
		t.Fatalf("PlanSync(prune) = %+v, want delete", plan)
	}
	if err := engine.Apply(t.Context(), plan); err != nil {
		t.Fatalf("Apply(prune): %v", err)
	}
	if report = engine.Check(t.Context(), nil); !report.Clean() {
		t.Fatalf("post-prune Check() = %+v, want clean", report)
	}
}

func TestEngineExceptionDocumentsDivergence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "skills", "private", "SKILL.md"), "# Private\n")
	doc := testDocument(dir)
	doc.Config.Pairs[1].Optional = true
	doc.Config.Exceptions = []config.Exception{{Pair: "skills", Paths: []string{"private/**"}, Reason: "Claude-only runtime"}}
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
	if report := engine.Check(t.Context(), map[string]bool{"skills": true}); !report.Clean() {
		t.Fatalf("Check() = %+v, want documented divergence ignored", report)
	}
}

func TestMCPWiringIgnoresSecretValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{
  "mcpServers": {
    "tasks": {
      "command": "/usr/bin/env",
      "args": ["tasks-mcp"],
      "env": {"TASKS_TOKEN": "claude-secret"}
    }
  }
}`)
	writeTestFile(t, filepath.Join(dir, ".codex", "config.toml"), `[mcp_servers.tasks]
command = "/usr/bin/env"
args = ["tasks-mcp"]

[mcp_servers.tasks.env]
TASKS_TOKEN = "codex-secret"
`)
	doc := testDocument(dir)
	doc.Config.MCPServers = []config.MCPServer{{
		ID: "mcp-tasks",
		Peers: map[string]config.MCPPeer{
			"claude": {Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
		},
		RequiredEnv: []string{"TASKS_TOKEN"},
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
	report := engine.Check(t.Context(), map[string]bool{"mcp-tasks": true})
	if !report.Clean() {
		t.Fatalf("Check() = %+v, want different secret values ignored", report)
	}

	writeTestFile(t, filepath.Join(dir, ".codex", "config.toml"), `[mcp_servers.tasks]
command = "/usr/bin/other"
args = ["tasks-mcp"]

[mcp_servers.tasks.env]
TASKS_TOKEN = "codex-secret"
`)
	report = engine.Check(t.Context(), map[string]bool{"mcp-tasks": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateDifferent {
		t.Fatalf("Check(public drift) = %+v, want different", report)
	}
	if strings.Contains(report.Pairs[0].Findings[0].Detail, "secret") && strings.Contains(report.Pairs[0].Findings[0].Detail, "codex-secret") {
		t.Fatal("MCP finding leaked a secret value")
	}
	violations, err := engine.Guard(t.Context(), []string{".codex/config.toml"})
	if err != nil {
		t.Fatalf("Guard(MCP config): %v", err)
	}
	if len(violations) != 1 || violations[0].Pair != "mcp-tasks" {
		t.Fatalf("Guard(MCP config) = %+v, want MCP violation", violations)
	}
}

func TestSyncCreatesEmptyOptionalCounterpartTree(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "claude", "skills"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(claude skills): %v", err)
	}
	doc := testDocument(dir)
	doc.Config.Pairs[1].Optional = true
	engine, closeEngine := newEngine(t, doc)
	t.Cleanup(closeEngine)
	selected := map[string]bool{"skills": true}
	plan, err := engine.PlanSync(t.Context(), Side("claude"), false, selected)
	if err != nil {
		t.Fatalf("PlanSync(): %v", err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Kind != OperationMkdir {
		t.Fatalf("PlanSync() = %+v, want mkdir", plan)
	}
	if err := engine.Apply(t.Context(), plan); err != nil {
		t.Fatalf("Apply(mkdir): %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "codex", "skills"))
	if err != nil || !info.IsDir() {
		t.Fatalf("Codex skills directory = %v, %v", info, err)
	}
	if _, err := engine.PlanSync(t.Context(), Side("invalid"), false, selected); err == nil {
		t.Fatal("PlanSync(invalid side) succeeded")
	}
}

func TestCanceledCheckReportsEveryCheck(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	doc := testDocument(dir)
	doc.Config.MCPServers = []config.MCPServer{{
		ID: "mcp", Name: "MCP",
		Peers: map[string]config.MCPPeer{
			"claude": {Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
		},
	}}
	doc.Config.Activations = []config.Activation{{
		ID: "live", Name: "Live",
		Expected: config.Endpoint{Source: "workspace", Path: "tracked"},
		Live:     config.Endpoint{Source: "workspace", Path: "live"},
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
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report := engine.Check(ctx, nil)
	if got, want := report.FindingCount(), 4; got != want {
		t.Fatalf("canceled FindingCount() = %d, want %d: %+v", got, want, report)
	}
}

func TestRunTasksBoundsConcurrency(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{}, maxConcurrentChecks+1)
		release := make(chan struct{})
		tasks := make([]func() int, maxConcurrentChecks+1)
		for i := range tasks {
			tasks[i] = func() int {
				started <- struct{}{}
				<-release
				return i
			}
		}
		var wait sync.WaitGroup
		wait.Go(func() { runTasks(tasks) })
		for range maxConcurrentChecks {
			<-started
		}
		synctest.Wait()
		select {
		case <-started:
			t.Fatalf("runTasks() started more than %d tasks", maxConcurrentChecks)
		default:
		}
		close(release)
		wait.Wait()
	})
}

func TestPairAcrossIndependentSourceRoots(t *testing.T) {
	t.Parallel()

	claudeDir := t.TempDir()
	codexDir := t.TempDir()
	writeTestFile(t, filepath.Join(claudeDir, "CLAUDE.md"), "# Instructions\n\nFrom Claude.\n")
	writeTestFile(t, filepath.Join(codexDir, "AGENTS.md"), "# Instructions\n\nOld.\n")
	doc := &config.Document{
		CWD: t.TempDir(),
		Roots: map[string]string{
			"synced-claude": claudeDir,
			"local-codex":   codexDir,
		},
		Config: config.Config{
			Version: config.CurrentVersion,
			Sources: map[string]config.Source{
				"synced-claude": {Root: claudeDir},
				"local-codex":   {Root: codexDir},
			},
			Pairs: []config.Pair{{
				ID: "cross-root", Kind: "file", Normalizer: "text",
				Peers: map[string]config.Endpoint{
					"claude": {Source: "synced-claude", Path: "CLAUDE.md"},
					"codex":  {Source: "local-codex", Path: "AGENTS.md"},
				},
			}},
		},
	}
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
	if err := engine.Apply(t.Context(), plan); err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	codexRoot, err := roots.Root("local-codex")
	if err != nil {
		t.Fatalf("Root(local-codex): %v", err)
	}
	data, _, err := codexRoot.ReadFile("AGENTS.md", 1024)
	if err != nil {
		t.Fatalf("ReadFile(AGENTS.md): %v", err)
	}
	if !strings.Contains(string(data), "From Claude") {
		t.Fatalf("cross-root sync result = %q", data)
	}
}

func TestSemanticSyncIsManualByDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "claude", "skills", "alpha", "SKILL.md"), "# Claude version\n")
	writeTestFile(t, filepath.Join(dir, "codex", "skills", "alpha", "SKILL.md"), "# Codex version\n")
	doc := testDocument(dir)
	doc.Config.Pairs[1].Sync = ""
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
	if len(plan.Operations) != 0 || len(plan.Unresolved) != 1 || !strings.Contains(plan.Unresolved[0].Detail, "raw copy disabled") {
		t.Fatalf("PlanSync(semantic default) = %+v, want manual finding", plan)
	}
}

func newTestEngine(t testing.TB, dir string) (*Engine, func()) {
	t.Helper()
	return newEngine(t, testDocument(dir))
}

func newEngine(t testing.TB, doc *config.Document) (*Engine, func()) {
	t.Helper()
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	engine, err := New(doc, roots)
	if err != nil {
		_ = roots.Close()
		t.Fatalf("New(): %v", err)
	}
	return engine, func() {
		if err := roots.Close(); err != nil {
			t.Errorf("roots.Close(): %v", err)
		}
	}
}

func testDocument(dir string) *config.Document {
	return &config.Document{
		Path: filepath.Join(dir, "agentlink.yaml"),
		Dir:  dir,
		CWD:  dir,
		Roots: map[string]string{
			"workspace": dir,
		},
		Config: config.Config{
			Version: config.CurrentVersion,
			Sources: map[string]config.Source{"workspace": {Root: dir}},
			Pairs: []config.Pair{
				{
					ID: "instructions", Kind: "file", Normalizer: "instructions",
					Peers: map[string]config.Endpoint{
						"claude": {Source: "workspace", Path: "claude/CLAUDE.md"},
						"codex":  {Source: "workspace", Path: "codex/AGENTS.md"},
					},
				},
				{
					ID: "skills", Kind: "tree", Normalizer: "skill", Sync: "copy",
					Peers: map[string]config.Endpoint{
						"claude": {Source: "workspace", Path: "claude/skills"},
						"codex":  {Source: "workspace", Path: "codex/skills"},
					},
				},
			},
		},
	}
}

func writeTestFile(t testing.TB, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}
}
