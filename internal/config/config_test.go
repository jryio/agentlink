package config

import (
	"strconv"
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
		{"file limit above ceiling", func(c *Config) { c.Limits.MaxFileSize = hardMaxSize + 1 }, "max_file_size"},
		{"inventory limit above ceiling", func(c *Config) { c.Limits.MaxFiles = hardMaxFiles + 1 }, "max_files"},
		{"too many pairs", func(c *Config) { c.Pairs = manyPairs() }, "pairs: must not exceed"},
		{"too many mcp servers", func(c *Config) { c.MCPServers = manyMCP() }, "mcp_servers: must not exceed"},
		{"too many activations", func(c *Config) { c.Activations = manyActivations() }, "activations: must not exceed"},
		{"invalid source name", func(c *Config) { c.Sources["Bad Name"] = Source{Root: "."} }, "name must match"},
		{"empty source root", func(c *Config) { c.Sources["root"] = Source{} }, "must not be empty"},
		{"invalid relative base", func(c *Config) { c.Sources["root"] = Source{Root: ".", RelativeTo: "repo"} }, "must be config or cwd"},
		{"invalid pair id", func(c *Config) { c.Pairs[0].ID = "Bad" }, "id: must match"},
		{"duplicate pair id", func(c *Config) { c.Pairs = append(c.Pairs, c.Pairs[0]) }, "duplicate"},
		{"invalid kind", func(c *Config) { c.Pairs[0].Kind = "directory" }, "must be file, tree, or siblings"},
		{"invalid normalizer", func(c *Config) { c.Pairs[0].Normalizer = "magic" }, "normalizer"},
		{"invalid sync policy", func(c *Config) { c.Pairs[0].Sync = "merge" }, "sync"},
		{"single peer", func(c *Config) {
			c.Pairs[0].Peers = map[string]Endpoint{"claude": {Source: "root", Path: "CLAUDE.md"}}
		}, "exactly two agents"},
		{"unknown peer agent", func(c *Config) {
			c.Pairs[0].Peers["bogus"] = c.Pairs[0].Peers["codex"]
			delete(c.Pairs[0].Peers, "codex")
		}, "unknown agent"},
		{"identical endpoints", func(c *Config) {
			c.Pairs[0].Peers["codex"] = c.Pairs[0].Peers["claude"]
		}, "endpoints must be different"},
		{"translate with plain text normalizer", func(c *Config) {
			c.Pairs[0].Normalizer = "text"
			c.Pairs[0].Sync = "translate"
		}, "translate requires normalizer"},
		{"translate hooks for code-only agent", func(c *Config) {
			c.Pairs[0].Normalizer = "hook"
			c.Pairs[0].Sync = "translate"
			c.Pairs[0].Peers["pi"] = c.Pairs[0].Peers["codex"]
			delete(c.Pairs[0].Peers, "codex")
		}, "no declarative hook file"},
		{"file source root", func(c *Config) { withClaudeEndpoint(c, Endpoint{Source: "root", Path: "."}) }, "file endpoints must name files"},
		{"unknown source", func(c *Config) { withClaudeEndpoint(c, Endpoint{Source: "missing", Path: "CLAUDE.md"}) }, "unknown source"},
		{"escaping endpoint", func(c *Config) { withClaudeEndpoint(c, Endpoint{Source: "root", Path: "../escape"}) }, "beneath its source"},
		{"backslash endpoint", func(c *Config) { withClaudeEndpoint(c, Endpoint{Source: "root", Path: `..\escape`}) }, "slash-separated"},
		{"nested sibling name", func(c *Config) {
			c.Pairs[0].Kind = "siblings"
			withClaudeEndpoint(c, Endpoint{Source: "root", Path: "docs/CLAUDE.md"})
		}, "siblings path must be a file name"},
		{"invalid double star", func(c *Config) { c.Ignore = []string{"foo**bar"} }, "complete path component"},
		{"unknown exception pair", func(c *Config) { c.Exceptions = []Exception{{Pair: "missing", Paths: []string{"x"}, Reason: "test"}} }, "unknown pair"},
		{"empty exception paths", func(c *Config) { c.Exceptions = []Exception{{Pair: "instructions", Reason: "test"}} }, "paths: must not be empty"},
		{"empty exception reason", func(c *Config) { c.Exceptions = []Exception{{Pair: "instructions", Paths: []string{"x"}}} }, "must document"},
		{"invalid MCP env", func(c *Config) { c.MCPServers = []MCPServer{validMCP("mcp", "BAD-NAME")} }, "invalid environment name"},
		{"MCP peer without MCP support", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.Peers["pi"] = server.Peers["codex"]
			delete(server.Peers, "codex")
			c.MCPServers = []MCPServer{server}
		}, "has no MCP configuration"},
		{"unknown MCP peer agent", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.Peers["bogus"] = server.Peers["codex"]
			delete(server.Peers, "codex")
			c.MCPServers = []MCPServer{server}
		}, "unknown agent"},
		{"identical MCP peers", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.Peers["codex"] = server.Peers["claude"]
			c.MCPServers = []MCPServer{server}
		}, "MCP endpoints must be different"},
		{"MCP source root", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			peer := server.Peers["claude"]
			peer.Config.Path = "."
			server.Peers["claude"] = peer
			c.MCPServers = []MCPServer{server}
		}, "MCP config endpoints must name files"},
		{"duplicate MCP env", func(c *Config) {
			server := validMCP("mcp", "TOKEN")
			server.RequiredEnv = append(server.RequiredEnv, "TOKEN")
			c.MCPServers = []MCPServer{server}
		}, "duplicate environment name"},
		{"duplicate MCP id", func(c *Config) { c.MCPServers = []MCPServer{validMCP("instructions", "TOKEN")} }, "duplicate"},
		{"duplicate activation id", func(c *Config) {
			c.Activations = []Activation{{ID: "instructions", Expected: c.Pairs[0].Peers["claude"], Live: c.Pairs[0].Peers["codex"]}}
		}, "duplicate"},
		{"identical activation endpoints", func(c *Config) {
			c.Activations = []Activation{{ID: "live", Expected: c.Pairs[0].Peers["claude"], Live: c.Pairs[0].Peers["claude"]}}
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
	if got := cfg.Pairs[0].PeerIDs(); got != [2]string{"claude", "codex"} {
		t.Errorf("PeerIDs() = %v, want [claude codex]", got)
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
	// Values above the hard ceilings are clamped so a repository-controlled
	// limit can never remove the application's resource budget (CWE-400).
	cfg.Limits = Limits{MaxFileSize: hardMaxSize + 1, MaxFiles: hardMaxFiles + 1}
	if got := cfg.MaxFileSize(); got != hardMaxSize {
		t.Errorf("ceiling MaxFileSize() = %d, want %d", got, hardMaxSize)
	}
	if got := cfg.MaxFiles(); got != hardMaxFiles {
		t.Errorf("ceiling MaxFiles() = %d, want %d", got, hardMaxFiles)
	}
}

func withClaudeEndpoint(c *Config, endpoint Endpoint) {
	c.Pairs[0].Peers["claude"] = endpoint
}

func validConfig() Config {
	return Config{
		Version: CurrentVersion,
		Sources: map[string]Source{"root": {Root: "."}},
		Pairs: []Pair{{
			ID: "instructions", Kind: "file", Normalizer: "instructions",
			Peers: map[string]Endpoint{
				"claude": {Source: "root", Path: "CLAUDE.md"},
				"codex":  {Source: "root", Path: "AGENTS.md"},
			},
		}},
	}
}

func manyPairs() []Pair {
	pairs := make([]Pair, maxPairs+1)
	for i := range pairs {
		pairs[i] = Pair{
			ID: "p" + strconv.Itoa(i), Kind: "file",
			Peers: map[string]Endpoint{
				"claude": {Source: "root", Path: "CLAUDE.md"},
				"codex":  {Source: "root", Path: "AGENTS.md"},
			},
		}
	}
	return pairs
}

func manyMCP() []MCPServer {
	servers := make([]MCPServer, maxMCPServers+1)
	for i := range servers {
		servers[i] = MCPServer{
			ID: "s" + strconv.Itoa(i),
			Peers: map[string]MCPPeer{
				"claude": {Config: Endpoint{Source: "root", Path: "a.json"}, Server: "svc"},
				"codex":  {Config: Endpoint{Source: "root", Path: "b.toml"}, Server: "svc"},
			},
		}
	}
	return servers
}

func manyActivations() []Activation {
	activations := make([]Activation, maxActivations+1)
	for i := range activations {
		activations[i] = Activation{
			ID:       "a" + strconv.Itoa(i),
			Expected: Endpoint{Source: "root", Path: "expected"},
			Live:     Endpoint{Source: "root", Path: "live"},
		}
	}
	return activations
}

func validMCP(id, env string) MCPServer {
	return MCPServer{
		ID: id,
		Peers: map[string]MCPPeer{
			"claude": {Config: Endpoint{Source: "root", Path: ".mcp.json"}, Server: "tasks"},
			"codex":  {Config: Endpoint{Source: "root", Path: ".codex/config.toml"}, Server: "tasks"},
		},
		RequiredEnv: []string{env},
	}
}
