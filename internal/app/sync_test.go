package app

import (
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/config"
)

func TestSyncPostCheckExcludesMCPWiring(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	configPath := filepath.Join(dir, "agentlink.yaml")
	data := append(config.Sample(), []byte(`
mcp_servers:
  - id: missing-mcp
    claude:
      config: {source: project, path: .mcp.json}
      server: tasks
    codex:
      config: {source: project, path: .codex/config.toml}
      server: tasks
`)...)
	writeAppFile(t, configPath, string(data))
	writeAppFile(t, filepath.Join(dir, "CLAUDE.md"), "# Instructions\n\nAligned by sync.\n")

	runCLI(t, dir, nil, "sync", "--from", "claude", "--apply")
	if _, _, err := runCLIErr(t, dir, nil, "check"); err == nil {
		t.Fatal("check succeeded despite missing MCP configuration")
	} else {
		var exitErr *ExitError
		if !errors.As(err, &exitErr) || exitErr.Code != exitDrift {
			t.Fatalf("check error = %v, want drift", err)
		}
	}
}

func TestSyncApplyEmitsOneJSONDocument(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	writeAppFile(t, filepath.Join(dir, "CLAUDE.md"), "# Instructions\n\nAligned by sync.\n")

	output, _ := runCLI(t, dir, nil, "--json", "sync", "--from", "claude", "--apply")
	decoder := json.NewDecoder(strings.NewReader(output))
	var result struct {
		Applied      bool         `json:"applied"`
		Verification reportOutput `json:"verification"`
	}
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("Decode(sync result): %v\noutput: %s", err, output)
	}
	if !result.Applied || !result.Verification.Clean {
		t.Fatalf("sync result = %+v, want applied and clean", result)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatalf("second Decode(sync result) error = %v, want io.EOF\noutput: %s", err, output)
	}
}
