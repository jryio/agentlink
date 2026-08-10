package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

var configNames = [...]string{"agentlink.yaml", "agentlink.yml"}

const maxConfigFileSize = 4 << 20

// Document is a validated configuration and the context used to resolve it.
type Document struct {
	Config Config
	Path   string
	Dir    string
	CWD    string
	Roots  map[string]string
}

// Find locates a configuration without depending on a particular project or
// version-control layout.
func Find(explicit, cwd string) (string, error) {
	if explicit != "" {
		return absolute(explicit, cwd)
	}
	if env := os.Getenv("AGENTLINK_CONFIG"); env != "" {
		return absolute(env, cwd)
	}

	dir, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		found := existingConfigs(dir)
		switch len(found) {
		case 0:
		case 1:
			return found[0], nil
		default:
			return "", fmt.Errorf("multiple configurations in %s: %s", dir, strings.Join(found, ", "))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		configDir := filepath.Join(userConfigDir, "agentlink")
		found := existingConfigs(configDir)
		switch len(found) {
		case 0:
		case 1:
			return found[0], nil
		default:
			return "", fmt.Errorf("multiple configurations in %s: %s", configDir, strings.Join(found, ", "))
		}
	}
	return "", errors.New("no agentlink.yaml found; run `agentlink init` or pass --config")
}

// rejectLegacyKeys detects version-1 configurations before strict decoding
// rejects them with a bare unknown-field error. Version 1 keyed pair and MCP
// endpoints by fixed claude:/codex: fields; version 2 uses peers: maps keyed
// by registered agent ID.
func rejectLegacyKeys(path string, data []byte) error {
	var probe struct {
		Pairs      []map[string]any `yaml:"pairs"`
		MCPServers []map[string]any `yaml:"mcp_servers"`
	}
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil // not decodable at all; the strict decoder reports it better
	}
	usesLegacyKeys := func(entry map[string]any) bool {
		if _, ok := entry["claude"]; ok {
			return true
		}
		_, ok := entry["codex"]
		return ok
	}
	for _, entry := range probe.Pairs {
		if usesLegacyKeys(entry) {
			return fmt.Errorf("decode config %s: version-1 claude:/codex: endpoint keys are no longer supported; rewrite endpoints as a peers: map keyed by agent ID and set version: %d (see CHANGELOG.md)", path, CurrentVersion)
		}
	}
	for _, entry := range probe.MCPServers {
		if usesLegacyKeys(entry) {
			return fmt.Errorf("decode config %s: version-1 claude:/codex: endpoint keys are no longer supported; rewrite endpoints as a peers: map keyed by agent ID and set version: %d (see CHANGELOG.md)", path, CurrentVersion)
		}
	}
	return nil
}

// Load reads, strictly decodes, validates, and resolves a configuration.
func Load(file, cwd string) (*Document, error) {
	path, err := absolute(file, cwd)
	if err != nil {
		return nil, err
	}
	path, err = resolveFinalSymlink(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config %s: %w", path, err)
	}
	data, err := readConfined(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := rejectLegacyKeys(path, data); err != nil {
		return nil, err
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("decode config %s: multiple YAML documents are not allowed", path)
		}
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}
	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}
	doc := &Document{
		Config: cfg,
		Path:   path,
		Dir:    filepath.Dir(path),
		CWD:    absCWD,
		Roots:  make(map[string]string, len(cfg.Sources)),
	}
	for name, source := range cfg.Sources {
		root, resolveErr := resolveSource(source, doc.Dir, doc.CWD)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve source %q: %w", name, resolveErr)
		}
		doc.Roots[name] = root
	}
	if err := validateResolvedEndpoints(doc); err != nil {
		return nil, fmt.Errorf("validate resolved endpoints: %w", err)
	}
	return doc, nil
}

func validateResolvedEndpoints(doc *Document) error {
	rootIdentities := make(map[string]rootIdentity, len(doc.Roots))
	for name, root := range doc.Roots {
		identity := rootIdentity{path: root}
		confined, err := os.OpenRoot(root)
		if err == nil {
			identity.info, err = confined.Stat(".")
			closeErr := confined.Close()
			if err != nil || closeErr != nil {
				identity.info = nil
			}
		}
		rootIdentities[name] = identity
	}
	for _, pair := range doc.Config.Pairs {
		ids := pair.PeerIDs()
		left, right := pair.Peers[ids[0]], pair.Peers[ids[1]]
		leftPath := resolvedEndpoint(rootIdentities, left)
		rightPath := resolvedEndpoint(rootIdentities, right)
		if endpointsEqual(rootIdentities, left, right) {
			return fmt.Errorf("pair %q endpoints resolve to the same path %s", pair.ID, leftPath)
		}
		if pair.Kind == KindTree && endpointsOverlap(rootIdentities, left, right) {
			return fmt.Errorf("pair %q tree endpoints overlap: %s and %s", pair.ID, leftPath, rightPath)
		}
	}
	for _, server := range doc.Config.MCPServers {
		ids := server.PeerIDs()
		left, right := server.Peers[ids[0]], server.Peers[ids[1]]
		if endpointsEqual(rootIdentities, left.Config, right.Config) && left.Server == right.Server {
			return fmt.Errorf("MCP check %q peers resolve to the same entry", server.ID)
		}
	}
	for _, activation := range doc.Config.Activations {
		if endpointsEqual(rootIdentities, activation.Expected, activation.Live) {
			return fmt.Errorf("activation %q endpoints resolve to the same path", activation.ID)
		}
	}
	return nil
}

type rootIdentity struct {
	path string
	info os.FileInfo
}

func resolvedEndpoint(roots map[string]rootIdentity, endpoint Endpoint) string {
	return filepath.Clean(filepath.Join(roots[endpoint.Source].path, filepath.FromSlash(endpoint.Path)))
}

func endpointsEqual(roots map[string]rootIdentity, first, second Endpoint) bool {
	if rootsShareIdentity(roots[first.Source], roots[second.Source]) {
		return endpointRelativePath(first) == endpointRelativePath(second)
	}
	return resolvedEndpoint(roots, first) == resolvedEndpoint(roots, second)
}

func endpointsOverlap(roots map[string]rootIdentity, first, second Endpoint) bool {
	if rootsShareIdentity(roots[first.Source], roots[second.Source]) {
		return pathsOverlap(endpointRelativePath(first), endpointRelativePath(second))
	}
	return pathsOverlap(resolvedEndpoint(roots, first), resolvedEndpoint(roots, second))
}

func rootsShareIdentity(first, second rootIdentity) bool {
	return first.info != nil && second.info != nil && os.SameFile(first.info, second.info)
}

func endpointRelativePath(endpoint Endpoint) string {
	return filepath.Clean(filepath.FromSlash(endpoint.Path))
}

func pathsOverlap(first, second string) bool {
	return pathContains(first, second) || pathContains(second, first)
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func existingConfigs(dir string) []string {
	var found []string
	root, err := os.OpenRoot(dir)
	if err != nil {
		return found
	}
	defer func() { _ = root.Close() }()
	for _, name := range configNames {
		if info, statErr := root.Stat(name); statErr == nil && !info.IsDir() {
			found = append(found, filepath.Join(dir, name))
		}
	}
	return found
}

func readConfined(filePath string) (data []byte, err error) {
	root, err := os.OpenRoot(filepath.Dir(filePath))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	info, err := root.Stat(filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("configuration is not a regular file")
	}
	if info.Size() > maxConfigFileSize {
		return nil, fmt.Errorf("configuration is %d bytes; limit is %d", info.Size(), maxConfigFileSize)
	}
	file, err := root.Open(filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	data, err = io.ReadAll(io.LimitReader(file, maxConfigFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxConfigFileSize {
		return nil, fmt.Errorf("configuration grew beyond the %d-byte limit while being read", maxConfigFileSize)
	}
	return data, nil
}

func resolveFinalSymlink(filePath string) (string, error) {
	filePath = filepath.Clean(filePath)
	for range 32 {
		root, err := os.OpenRoot(filepath.Dir(filePath))
		if err != nil {
			return "", err
		}
		info, err := root.Lstat(filepath.Base(filePath))
		if err != nil {
			_ = root.Close()
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if err := root.Close(); err != nil {
				return "", err
			}
			return filePath, nil
		}
		target, err := root.Readlink(filepath.Base(filePath))
		closeErr := root.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(filePath), target)
		}
		filePath = filepath.Clean(target)
	}
	return "", errors.New("config symlink depth exceeds 32")
}

func resolveSource(source Source, configDir, cwd string) (string, error) {
	expanded, err := expand(source.Root)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	base := configDir
	if source.RelativeTo == "cwd" {
		base = cwd
	}
	return filepath.Abs(filepath.Join(base, expanded))
}

func absolute(raw, cwd string) (string, error) {
	expanded, err := expand(raw)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(cwd, expanded)
	}
	return filepath.Abs(expanded)
}

func expand(raw string) (string, error) {
	var missing string
	expanded := os.Expand(raw, func(key string) string {
		value, ok := os.LookupEnv(key)
		if !ok && missing == "" {
			missing = key
		}
		return value
	})
	if missing != "" {
		return "", fmt.Errorf("environment variable %s is not set", missing)
	}
	if expanded == "~" || strings.HasPrefix(expanded, "~/") || strings.HasPrefix(expanded, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if expanded == "~" {
			return home, nil
		}
		return filepath.Join(home, expanded[2:]), nil
	}
	if strings.HasPrefix(expanded, "~") {
		return "", errors.New("~user paths are not supported; use an absolute path")
	}
	return expanded, nil
}
