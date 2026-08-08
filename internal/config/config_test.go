package config

import (
	"strings"
	"testing"
)

func TestConfigValidateFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"unsupported version", func(c *Config) { c.Version = 99 }, "version"},
		{"no sources", func(c *Config) { c.Sources = nil }, "define at least one source"},
		{"no pairs", func(c *Config) { c.Pairs = nil }, "define at least one pair"},
		{"negative file limit", func(c *Config) { c.Limits.MaxFileSize = -1 }, "max_file_size"},
		{"negative inventory limit", func(c *Config) { c.Limits.MaxFiles = -1 }, "max_files"},
		{"invalid source name", func(c *Config) { c.Sources["Bad Name"] = Source{Root: "."} }, "name must match"},
		{"empty source root", func(c *Config) { c.Sources["root"] = Source{} }, "must not be empty"},
		{"invalid relative base", func(c *Config) { c.Sources["root"] = Source{Root: ".", RelativeTo: "repo"} }, "must be config or cwd"},
		{"invalid pair id", func(c *Config) { c.Pairs[0].ID = "Bad" }, "id: must match"},
		{"duplicate pair id", func(c *Config) { c.Pairs = append(c.Pairs, c.Pairs[0]) }, "duplicate"},
		{"invalid kind", func(c *Config) { c.Pairs[0].Kind = "directory" }, "must be file, tree, or siblings"},
		{"invalid normalizer", func(c *Config) { c.Pairs[0].Normalizer = "magic" }, "normalizer"},
		{"invalid sync policy", func(c *Config) { c.Pairs[0].Sync = "merge" }, "sync"},
		{"identical endpoints", func(c *Config) { c.Pairs[0].Codex = c.Pairs[0].Claude }, "endpoints must be different"},
		{"file source root", func(c *Config) { c.Pairs[0].Claude.Path = "." }, "file endpoints must name files"},
		{"unknown source", func(c *Config) { c.Pairs[0].Claude.Source = "missing" }, "unknown source"},
		{"escaping endpoint", func(c *Config) { c.Pairs[0].Claude.Path = "../escape" }, "beneath its source"},
		{"backslash endpoint", func(c *Config) { c.Pairs[0].Claude.Path = `..\escape` }, "slash-separated"},
		{"nested sibling name", func(c *Config) { c.Pairs[0].Kind = "siblings"; c.Pairs[0].Claude.Path = "docs/CLAUDE.md" }, "siblings path must be a file name"},
		{"invalid double star", func(c *Config) { c.Ignore = []string{"foo**bar"} }, "complete path component"},
		{"unknown exception pair", func(c *Config) { c.Exceptions = []Exception{{Pair: "missing", Paths: []string{"x"}, Reason: "test"}} }, "unknown pair"},
		{"empty exception paths", func(c *Config) { c.Exceptions = []Exception{{Pair: "instructions", Reason: "test"}} }, "paths: must not be empty"},
		{"empty exception reason", func(c *Config) { c.Exceptions = []Exception{{Pair: "instructions", Paths: []string{"x"}}} }, "must document"},
		{"invalid MCP env", func(c *Config) { c.MCPServers = []MCPServer{validMCP("mcp", "BAD-NAME")} }, "invalid environment name"},
		{"identical MCP peers", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.Codex = server.Claude
			c.MCPServers = []MCPServer{server}
		}, "MCP peers must be different"},
		{"MCP source root", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.Claude.Config.Path = "."
			c.MCPServers = []MCPServer{server}
		}, "MCP config endpoints must name files"},
		{"duplicate MCP env", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.RequiredEnv = append(server.RequiredEnv, "TOKEN")
			c.MCPServers = []MCPServer{server}
		}, "duplicate environment name"},
		{"duplicate MCP id", func(c *Config) { c.MCPServers = []MCPServer{validMCP("instructions", "TOKEN")} }, "duplicate"},
		{"duplicate activation id", func(c *Config) {
			c.Activations = []Activation{{ID: "instructions", Expected: c.Pairs[0].Claude, Live: c.Pairs[0].Codex}}
		}, "duplicate"},
		{"identical activation endpoints", func(c *Config) {
			c.Activations = []Activation{{ID: "live", Expected: c.Pairs[0].Claude, Live: c.Pairs[0].Claude}}
		}, "expected and live endpoints must be different"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig()
			test.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Config.Validate() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestConfigAccessors(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid Config.Validate(): %v", err)
	}
	if _, ok := cfg.PairByID("instructions"); !ok {
		t.Fatal("PairByID(instructions) was not found")
	}
	if _, ok := cfg.PairByID("missing"); ok {
		t.Fatal("PairByID(missing) was found")
	}
	if got := cfg.MaxFileSize(); got != defaultMaxSize {
		t.Errorf("MaxFileSize() = %d, want %d", got, defaultMaxSize)
	}
	if got := cfg.MaxFiles(); got != defaultMaxFiles {
		t.Errorf("MaxFiles() = %d, want %d", got, defaultMaxFiles)
	}
	cfg.Limits = Limits{MaxFileSize: 10, MaxFiles: 20}
	if got := cfg.MaxFileSize(); got != 10 {
		t.Errorf("custom MaxFileSize() = %d, want 10", got)
	}
	if got := cfg.MaxFiles(); got != 20 {
		t.Errorf("custom MaxFiles() = %d, want 20", got)
	}
}

func validConfig() Config {
	return Config{
		Version: CurrentVersion,
		Sources: map[string]Source{"root": {Root: "."}},
		Pairs: []Pair{{
			ID: "instructions", Kind: "file", Normalizer: "instructions",
			Claude: Endpoint{Source: "root", Path: "CLAUDE.md"},
			Codex:  Endpoint{Source: "root", Path: "AGENTS.md"},
		}},
	}
}

func validMCP(id, env string) MCPServer {
	return MCPServer{
		ID:          id,
		Claude:      MCPPeer{Config: Endpoint{Source: "root", Path: ".mcp.json"}, Server: "tasks"},
		Codex:       MCPPeer{Config: Endpoint{Source: "root", Path: ".codex/config.toml"}, Server: "tasks"},
		RequiredEnv: []string{env},
	}
}
