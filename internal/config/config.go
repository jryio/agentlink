// Package config defines and validates agentlink's YAML configuration.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	glob "github.com/jryio/agentlink/internal/pattern"
)

const (
	// CurrentVersion is the configuration format understood by this build.
	CurrentVersion  = 1
	defaultMaxSize  = 4 << 20
	defaultMaxFiles = 25_000
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Config is the on-disk configuration format.
type Config struct {
	Version     int               `yaml:"version" json:"version"`
	Sources     map[string]Source `yaml:"sources" json:"sources"`
	Pairs       []Pair            `yaml:"pairs" json:"pairs"`
	MCPServers  []MCPServer       `yaml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
	Activations []Activation      `yaml:"activations,omitempty" json:"activations,omitempty"`
	Exceptions  []Exception       `yaml:"exceptions,omitempty" json:"exceptions,omitempty"`
	Ignore      []string          `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Limits      Limits            `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// Source names a filesystem root. It can be a normal directory, a symlink into
// a sync service, or a directory managed by any version-control system.
type Source struct {
	Root       string `yaml:"root" json:"root"`
	RelativeTo string `yaml:"relative_to,omitempty" json:"relative_to,omitempty"`
	Optional   bool   `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// Pair maps Claude and Codex peer artifacts.
type Pair struct {
	ID         string   `yaml:"id" json:"id"`
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Kind       string   `yaml:"kind" json:"kind"`
	Claude     Endpoint `yaml:"claude" json:"claude"`
	Codex      Endpoint `yaml:"codex" json:"codex"`
	Normalizer string   `yaml:"normalizer,omitempty" json:"normalizer,omitempty"`
	Sync       string   `yaml:"sync,omitempty" json:"sync,omitempty"`
	Ignore     []string `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Optional   bool     `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// Endpoint selects a path beneath a named source root.
type Endpoint struct {
	Source string `yaml:"source" json:"source"`
	Path   string `yaml:"path" json:"path"`
}

// Exception documents an intentional divergence. Matching relative paths are
// omitted from drift checks and guards, but remain visible in `agentlink list`.
type Exception struct {
	Pair   string   `yaml:"pair" json:"pair"`
	Paths  []string `yaml:"paths" json:"paths"`
	Reason string   `yaml:"reason" json:"reason"`
}

// Limits bounds work performed on untrusted or accidentally huge trees.
type Limits struct {
	MaxFileSize int64 `yaml:"max_file_size,omitempty" json:"max_file_size,omitempty"`
	MaxFiles    int   `yaml:"max_files,omitempty" json:"max_files,omitempty"`
}

// MCPServer verifies secret-safe wiring parity for one MCP service.
type MCPServer struct {
	ID            string   `yaml:"id" json:"id"`
	Name          string   `yaml:"name,omitempty" json:"name,omitempty"`
	Claude        MCPPeer  `yaml:"claude" json:"claude"`
	Codex         MCPPeer  `yaml:"codex" json:"codex"`
	ComparePublic *bool    `yaml:"compare_public,omitempty" json:"compare_public,omitempty"`
	RequiredEnv   []string `yaml:"required_env,omitempty" json:"required_env,omitempty"`
	Optional      bool     `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// MCPPeer locates a tool's configuration and its server table name.
type MCPPeer struct {
	Config Endpoint `yaml:"config" json:"config"`
	Server string   `yaml:"server" json:"server"`
}

// Activation verifies that a live tool path is a symlink to a durable artifact.
type Activation struct {
	ID       string   `yaml:"id" json:"id"`
	Name     string   `yaml:"name,omitempty" json:"name,omitempty"`
	Expected Endpoint `yaml:"expected" json:"expected"`
	Live     Endpoint `yaml:"live" json:"live"`
	Optional bool     `yaml:"optional,omitempty" json:"optional,omitempty"`
}

// PairByID returns the configured pair with id.
func (c *Config) PairByID(id string) (Pair, bool) {
	for _, pair := range c.Pairs {
		if pair.ID == id {
			return pair, true
		}
	}
	return Pair{}, false
}

// PairIDs returns pair identifiers in deterministic order.
func (c *Config) PairIDs() []string {
	ids := make([]string, 0, len(c.Pairs)+len(c.MCPServers)+len(c.Activations))
	for _, pair := range c.Pairs {
		ids = append(ids, pair.ID)
	}
	for _, server := range c.MCPServers {
		ids = append(ids, server.ID)
	}
	for _, activation := range c.Activations {
		ids = append(ids, activation.ID)
	}
	slices.Sort(ids)
	return ids
}

// ArtifactPairIDs returns only file/tree pair identifiers.
func (c *Config) ArtifactPairIDs() []string {
	ids := make([]string, 0, len(c.Pairs))
	for _, pair := range c.Pairs {
		ids = append(ids, pair.ID)
	}
	slices.Sort(ids)
	return ids
}

// MaxFileSize returns the configured file-size limit or its safe default.
func (c *Config) MaxFileSize() int64 {
	if c.Limits.MaxFileSize > 0 {
		return c.Limits.MaxFileSize
	}
	return defaultMaxSize
}

// MaxFiles returns the per-tree inventory limit or its safe default.
func (c *Config) MaxFiles() int {
	if c.Limits.MaxFiles > 0 {
		return c.Limits.MaxFiles
	}
	return defaultMaxFiles
}

// Validate checks the complete configuration contract.
func (c *Config) Validate() error {
	var errs []error
	if c.Version != CurrentVersion {
		errs = append(errs, fmt.Errorf("version: got %d, want %d", c.Version, CurrentVersion))
	}
	if len(c.Sources) == 0 {
		errs = append(errs, errors.New("sources: define at least one source"))
	}
	if len(c.Pairs) == 0 {
		errs = append(errs, errors.New("pairs: define at least one pair"))
	}
	if c.Limits.MaxFileSize < 0 {
		errs = append(errs, errors.New("limits.max_file_size: must not be negative"))
	}
	if c.Limits.MaxFiles < 0 {
		errs = append(errs, errors.New("limits.max_files: must not be negative"))
	}

	for _, name := range slices.Sorted(maps.Keys(c.Sources)) {
		source := c.Sources[name]
		if !idPattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("sources.%s: name must match %s", name, idPattern))
		}
		if strings.TrimSpace(source.Root) == "" {
			errs = append(errs, fmt.Errorf("sources.%s.root: must not be empty", name))
		}
		if source.RelativeTo != "" && source.RelativeTo != "config" && source.RelativeTo != "cwd" {
			errs = append(errs, fmt.Errorf("sources.%s.relative_to: must be config or cwd", name))
		}
	}

	seen := make(map[string]struct{}, len(c.Pairs))
	artifactSeen := make(map[string]struct{}, len(c.Pairs))
	for i, pair := range c.Pairs {
		prefix := fmt.Sprintf("pairs[%d]", i)
		if !idPattern.MatchString(pair.ID) {
			errs = append(errs, fmt.Errorf("%s.id: must match %s", prefix, idPattern))
		}
		if _, ok := seen[pair.ID]; ok {
			errs = append(errs, fmt.Errorf("%s.id: duplicate %q", prefix, pair.ID))
		}
		seen[pair.ID] = struct{}{}
		artifactSeen[pair.ID] = struct{}{}
		if pair.Kind != "file" && pair.Kind != "tree" && pair.Kind != "siblings" {
			errs = append(errs, fmt.Errorf("%s.kind: must be file, tree, or siblings", prefix))
		}
		if !validNormalizer(pair.Normalizer) {
			errs = append(errs, fmt.Errorf("%s.normalizer: must be exact, text, instructions, skill, hook, or presence", prefix))
		}
		if pair.Sync != "" && pair.Sync != "manual" && pair.Sync != "copy" {
			errs = append(errs, fmt.Errorf("%s.sync: must be manual or copy", prefix))
		}
		errs = append(errs, validateEndpoint(prefix+".claude", pair.Claude, c.Sources)...)
		errs = append(errs, validateEndpoint(prefix+".codex", pair.Codex, c.Sources)...)
		if pair.Claude == pair.Codex {
			errs = append(errs, fmt.Errorf("%s: Claude and Codex endpoints must be different", prefix))
		}
		if pair.Kind == "siblings" {
			if path.Base(pair.Claude.Path) != pair.Claude.Path || pair.Claude.Path == "." {
				errs = append(errs, fmt.Errorf("%s.claude.path: siblings path must be a file name", prefix))
			}
			if path.Base(pair.Codex.Path) != pair.Codex.Path || pair.Codex.Path == "." {
				errs = append(errs, fmt.Errorf("%s.codex.path: siblings path must be a file name", prefix))
			}
		}
		if pair.Kind == "file" && (pair.Claude.Path == "." || pair.Codex.Path == ".") {
			errs = append(errs, fmt.Errorf("%s: file endpoints must name files, not source roots", prefix))
		}
		for _, pattern := range append(slices.Clone(c.Ignore), pair.Ignore...) {
			if err := validatePattern(pattern); err != nil {
				errs = append(errs, fmt.Errorf("%s.ignore %q: %w", prefix, pattern, err))
			}
		}
	}
	for i, server := range c.MCPServers {
		prefix := fmt.Sprintf("mcp_servers[%d]", i)
		if !idPattern.MatchString(server.ID) {
			errs = append(errs, fmt.Errorf("%s.id: must match %s", prefix, idPattern))
		}
		if _, ok := seen[server.ID]; ok {
			errs = append(errs, fmt.Errorf("%s.id: duplicate %q", prefix, server.ID))
		}
		seen[server.ID] = struct{}{}
		errs = append(errs, validateEndpoint(prefix+".claude.config", server.Claude.Config, c.Sources)...)
		errs = append(errs, validateEndpoint(prefix+".codex.config", server.Codex.Config, c.Sources)...)
		if server.Claude == server.Codex {
			errs = append(errs, fmt.Errorf("%s: Claude and Codex MCP peers must be different", prefix))
		}
		if strings.TrimSpace(server.Claude.Server) == "" {
			errs = append(errs, fmt.Errorf("%s.claude.server: must not be empty", prefix))
		}
		if strings.TrimSpace(server.Codex.Server) == "" {
			errs = append(errs, fmt.Errorf("%s.codex.server: must not be empty", prefix))
		}
		if server.Claude.Config.Path == "." || server.Codex.Config.Path == "." {
			errs = append(errs, fmt.Errorf("%s: MCP config endpoints must name files, not source roots", prefix))
		}
		envSeen := make(map[string]struct{}, len(server.RequiredEnv))
		for _, name := range server.RequiredEnv {
			if !validEnvName(name) {
				errs = append(errs, fmt.Errorf("%s.required_env: invalid environment name %q", prefix, name))
			}
			if _, ok := envSeen[name]; ok {
				errs = append(errs, fmt.Errorf("%s.required_env: duplicate environment name %q", prefix, name))
			}
			envSeen[name] = struct{}{}
		}
	}
	for i, activation := range c.Activations {
		prefix := fmt.Sprintf("activations[%d]", i)
		if !idPattern.MatchString(activation.ID) {
			errs = append(errs, fmt.Errorf("%s.id: must match %s", prefix, idPattern))
		}
		if _, ok := seen[activation.ID]; ok {
			errs = append(errs, fmt.Errorf("%s.id: duplicate %q", prefix, activation.ID))
		}
		seen[activation.ID] = struct{}{}
		errs = append(errs, validateEndpoint(prefix+".expected", activation.Expected, c.Sources)...)
		errs = append(errs, validateEndpoint(prefix+".live", activation.Live, c.Sources)...)
		if activation.Expected == activation.Live {
			errs = append(errs, fmt.Errorf("%s: expected and live endpoints must be different", prefix))
		}
	}

	for i, exception := range c.Exceptions {
		prefix := fmt.Sprintf("exceptions[%d]", i)
		if _, ok := artifactSeen[exception.Pair]; !ok {
			errs = append(errs, fmt.Errorf("%s.pair: unknown pair %q", prefix, exception.Pair))
		}
		if len(exception.Paths) == 0 {
			errs = append(errs, fmt.Errorf("%s.paths: must not be empty", prefix))
		}
		if strings.TrimSpace(exception.Reason) == "" {
			errs = append(errs, fmt.Errorf("%s.reason: must document the divergence", prefix))
		}
		for _, pattern := range exception.Paths {
			if err := validatePattern(pattern); err != nil {
				errs = append(errs, fmt.Errorf("%s.paths %q: %w", prefix, pattern, err))
			}
		}
	}

	return errors.Join(errs...)
}

func validEnvName(name string) bool {
	if name == "" || (name[0] != '_' && (name[0] < 'A' || name[0] > 'Z') && (name[0] < 'a' || name[0] > 'z')) {
		return false
	}
	for i := 1; i < len(name); i++ {
		char := name[i]
		if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func validateEndpoint(prefix string, endpoint Endpoint, sources map[string]Source) []error {
	var errs []error
	if _, ok := sources[endpoint.Source]; !ok {
		errs = append(errs, fmt.Errorf("%s.source: unknown source %q", prefix, endpoint.Source))
	}
	if !fs.ValidPath(endpoint.Path) || strings.Contains(endpoint.Path, `\`) {
		errs = append(errs, fmt.Errorf("%s.path: %q must be a slash-separated path beneath its source", prefix, endpoint.Path))
	}
	return errs
}

func validNormalizer(name string) bool {
	switch name {
	case "", "exact", "text", "instructions", "skill", "hook", "presence":
		return true
	default:
		return false
	}
}

func validatePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return errors.New("pattern must not be empty")
	}
	if path.IsAbs(pattern) || strings.HasPrefix(path.Clean(pattern), "../") {
		return errors.New("pattern must stay beneath the pair")
	}
	_, err := glob.Compile([]string{pattern})
	return err
}
