package link

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/agent"
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
		format := map[string]agent.Dialect{"json": agent.DialectJSON, "toml": agent.DialectTOML, "yaml": agent.DialectYAML}[extension]
		values, err := readStructured(root, "config."+extension, 1024, format)
		if err != nil {
			t.Errorf("readStructured(%s): %v", extension, err)
			continue
		}
		if values["value"] != extension {
			t.Errorf("readStructured(%s)[value] = %v", extension, values["value"])
		}
	}
	writeTestFile(t, filepath.Join(dir, "config.jsonc"), "// comment\n{\"value\":\"jsonc\",}\n")
	values, err := readStructured(root, "config.jsonc", 1024, agent.DialectJSONC)
	if err != nil {
		t.Fatalf("readStructured(jsonc): %v", err)
	}
	if values["value"] != "jsonc" {
		t.Errorf("readStructured(jsonc)[value] = %v", values["value"])
	}
	if _, err := readStructured(root, "config.txt", 1024, agent.DialectNone); err == nil {
		t.Fatal("readStructured(unsupported format) succeeded, want error")
	}
	for _, name := range []string{"multiple.json", "multiple.yaml"} {
		format := agent.DialectJSON
		if strings.HasSuffix(name, ".yaml") {
			format = agent.DialectYAML
		}
		if _, err := readStructured(root, name, 1024, format); err == nil {
			t.Errorf("readStructured(%s) succeeded, want multiple-value error", name)
		}
	}
	writeTestFile(t, filepath.Join(dir, "secret.json"), `{"token":"do-not-print"`)
	if _, err := readStructured(root, "secret.json", 1024, agent.DialectJSON); err == nil || strings.Contains(err.Error(), "do-not-print") {
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
		ID: "mcp",
		Peers: map[string]config.MCPPeer{
			"claude": {Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
		},
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
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateMissing || report.Pairs[0].Findings[0].Peer != "codex" {
		t.Fatalf("Check(one entry) = %+v, want missing codex", report)
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
	if report.Pairs[0].Findings[0].State != StateMissing || report.Pairs[0].Findings[0].Peer != "claude" {
		t.Fatalf("Check(missing Claude config) = %+v", report)
	}
}

func TestMCPCrossAgentParity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentID   string
		file      string
		contents  string
		wantClean bool
	}{
		{
			"kilo jsonc mcp table with environment",
			"kilo", "kilo.jsonc",
			"// kilo config\n{\n  \"mcp\": {\n    \"tasks\": {\n      \"type\": \"local\",\n      \"command\": [\"/usr/bin/env\", \"tasks-mcp\"],\n      \"environment\": {\"TASKS_TOKEN\": \"kilo-secret\"},\n    },\n  },\n}\n",
			true,
		},
		{
			"opencode array command",
			"opencode", "opencode.json",
			`{"mcp": {"tasks": {"type": "local", "command": ["/usr/bin/env", "tasks-mcp"], "environment": {"TASKS_TOKEN": "opencode-secret"}}}}` + "\n",
			true,
		},
		{
			"opencode command drift detected",
			"opencode", "opencode.json",
			`{"mcp": {"tasks": {"type": "local", "command": ["/usr/bin/other", "tasks-mcp"]}}}` + "\n",
			false,
		},
		{
			"goose stdio extension counts as server",
			"goose", "goose-config.yaml",
			"extensions:\n  tasks:\n    type: stdio\n    command: /usr/bin/env\n    args: [tasks-mcp]\n    env:\n      TASKS_TOKEN: goose-secret\n",
			true,
		},
		{
			"goose builtin extension is not a server",
			"goose", "goose-config.yaml",
			"extensions:\n  tasks:\n    type: builtin\n    name: developer\n",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, filepath.Join(dir, ".agents", "mcp.json"), `{
  "mcpServers": {
    "tasks": {
      "command": "/usr/bin/env",
      "args": ["tasks-mcp"],
      "env": {"TASKS_TOKEN": "canonical-secret"}
    }
  }
}`)
			writeTestFile(t, filepath.Join(dir, test.file), test.contents)
			doc := testDocument(dir)
			doc.Config.MCPServers = []config.MCPServer{{
				ID: "mcp",
				Peers: map[string]config.MCPPeer{
					"agents":     {Config: config.Endpoint{Source: "workspace", Path: ".agents/mcp.json"}, Server: "tasks"},
					test.agentID: {Config: config.Endpoint{Source: "workspace", Path: test.file}, Server: "tasks"},
				},
				RequiredEnv: []string{"TASKS_TOKEN"},
			}}
			engine, closeEngine := newEngine(t, doc)
			t.Cleanup(closeEngine)
			report := engine.Check(t.Context(), map[string]bool{"mcp": true})
			if test.wantClean && !report.Clean() {
				t.Fatalf("Check() = %+v, want clean", report.Pairs[0].Findings)
			}
			if !test.wantClean && report.Clean() {
				t.Fatalf("Check() = clean, want drift")
			}
		})
	}
}

func TestMCPReadErrorTakesPriorityOverMissingPeer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{"token":"do-not-print"`)
	doc := testDocument(dir)
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
	report := engine.Check(t.Context(), map[string]bool{"mcp": true})
	if report.FindingCount() != 1 || report.Pairs[0].Findings[0].State != StateError {
		t.Fatalf("Check() = %+v, want Claude read error", report)
	}
	if strings.Contains(report.Pairs[0].Findings[0].Detail, "do-not-print") {
		t.Fatal("MCP read finding leaked malformed secret content")
	}
}

func TestMCPCodexEnvVarsObjectForm(t *testing.T) {
	t.Parallel()

	// Codex's env_vars accepts {name, source} objects as well as plain
	// strings; both count as forwarding the key.
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"tasks":{"command":"x","env":{"TOKEN":"s3cret"}}}}`)
	writeTestFile(t, filepath.Join(dir, ".codex", "config.toml"), "[mcp_servers.tasks]\ncommand = \"x\"\nenv_vars = [{name = \"TOKEN\", source = \"local\"}]\n")
	doc := testDocument(dir)
	doc.Config.MCPServers = []config.MCPServer{{
		ID: "mcp",
		Peers: map[string]config.MCPPeer{
			"claude": {Config: config.Endpoint{Source: "workspace", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: config.Endpoint{Source: "workspace", Path: ".codex/config.toml"}, Server: "tasks"},
		},
		RequiredEnv: []string{"TOKEN"},
	}}
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() { _ = roots.Close() })
	engine, err := New(doc, roots)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	report := engine.Check(t.Context(), map[string]bool{"mcp": true})
	if !report.Clean() {
		t.Fatalf("Check() = %+v, want clean: object-form env_vars forwards TOKEN", report)
	}
}
