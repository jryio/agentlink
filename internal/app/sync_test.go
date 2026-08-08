package app

import (
	"errors"
	"path/filepath"
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
