package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func TestReadStructuredFormats(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "config.json"), `{"value":"json"}`)
	writeTestFile(t, filepath.Join(dir, "config.toml"), `value = "toml"`)
	writeTestFile(t, filepath.Join(dir, "config.yaml"), "value: yaml\n")
	writeTestFile(t, filepath.Join(dir, "multiple.json"), "{} {}")
	writeTestFile(t, filepath.Join(dir, "multiple.yaml"), "{}\n---\n{}\n")
	writeTestFile(t, filepath.Join(dir, "config.txt"), "value=text\n")
	doc := testDocument(dir)
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() {
		if err := roots.Close(); err != nil {
			t.Errorf("roots.Close(): %v", err)
		}
	})
	root, err := roots.Root("workspace")
	if err != nil {
		t.Fatalf("Root(workspace): %v", err)
	}
	for _, extension := range []string{"json", "toml", "yaml"} {
		values, err := readStructured(root, "config."+extension, 1024)
		if err != nil {
			t.Errorf("readStructured(%s): %v", extension, err)
			continue
		}
		if values["value"] != extension {
			t.Errorf("readStructured(%s)[value] = %v", extension, values["value"])
		}
	}
	if _, err := readStructured(root, "config.txt", 1024); err == nil {
		t.Fatal("readStructured(txt) succeeded, want unsupported extension error")
	}
	for _, name := range []string{"multiple.json", "multiple.yaml"} {
		if _, err := readStructured(root, name, 1024); err == nil {
			t.Errorf("readStructured(%s) succeeded, want multiple-value error", name)
		}
	}
	writeTestFile(t, filepath.Join(dir, "secret.json"), `{"token":"do-not-print"`)
	if _, err := readStructured(root, "secret.json", 1024); err == nil || strings.Contains(err.Error(), "do-not-print") {
		t.Fatalf("readStructured(secret) error = %v, want redacted decode error", err)
	}
}

func TestMCPMissingEntriesAndEnvironment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{}`)
	writeTestFile(t, filepath.Join(dir, ".codex", "config.toml"), "")
	doc := testDocument(dir)
	doc.Config.MCPServers = []config.MCPServer{{
		ID:          "mcp",
		Claude:      config.MCPPeer{Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
		Codex:       config.MCPPeer{Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
		RequiredEnv: []string{"TOKEN"},
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
	report := engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateMissingBoth {
		t.Fatalf("Check(no entries) = %+v, want missing both", report)
	}

	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"tasks":{"command":"x"}}}`)
	report = engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateMissingCodex {
		t.Fatalf("Check(one entry) = %+v, want missing Codex", report)
	}

	writeTestFile(t, filepath.Join(dir, ".codex", "config.toml"), "[mcp_servers.tasks]\ncommand = \"x\"\n")
	report = engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateMissingBoth {
		t.Fatalf("Check(missing env) = %+v, want missing env on both", report)
	}

	if err := os.Remove(filepath.Join(dir, ".mcp.json")); err != nil {
		t.Fatalf("os.Remove(.mcp.json): %v", err)
	}
	report = engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.Pairs[0].Findings[0].State != StateMissingClaude {
		t.Fatalf("Check(missing Claude config) = %+v", report)
	}
}

func TestMCPReadErrorTakesPriorityOverMissingPeer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{"token":"do-not-print"`)
	doc := testDocument(dir)
	doc.Config.MCPServers = []config.MCPServer{{
		ID:     "mcp",
		Claude: config.MCPPeer{Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
		Codex:  config.MCPPeer{Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
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
	report := engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateError {
		t.Fatalf("Check() = %+v, want Claude read error", report)
	}
	if strings.Contains(report.Pairs[0].Findings[0].Detail, "do-not-print") {
		t.Fatal("MCP read finding leaked malformed secret content")
	}
}
