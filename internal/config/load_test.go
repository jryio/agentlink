package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSample(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	writeFile(t, path, Sample())
	doc, err := Load(path, dir)
	if err != nil {
		t.Fatalf("Load(%q): %v", path, err)
	}
	if got, want := doc.Roots["project"], dir; got != want {
		t.Errorf("project root = %q, want %q", got, want)
	}
	if got, want := len(doc.Config.Pairs), 2; got != want {
		t.Errorf("pair count = %d, want %d", got, want)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	data := strings.Replace(string(Sample()), "version: 2", "version: 2\nunknown: true", 1)
	writeFile(t, path, []byte(data))
	if _, err := Load(path, dir); err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("Load() error = %v, want unknown field error", err)
	}
}

func TestLoadResolvesCWDSource(t *testing.T) {
	t.Parallel()

	configDir := t.TempDir()
	workingDir := t.TempDir()
	path := filepath.Join(configDir, "agentlink.yaml")
	data := strings.Replace(string(Sample()), "relative_to: config", "relative_to: cwd", 1)
	writeFile(t, path, []byte(data))
	doc, err := Load(path, workingDir)
	if err != nil {
		t.Fatalf("Load(%q): %v", path, err)
	}
	if got := doc.Roots["project"]; got != workingDir {
		t.Errorf("project root = %q, want cwd %q", got, workingDir)
	}
}

func TestFindRejectsAmbiguousConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "agentlink.yaml"), Sample())
	writeFile(t, filepath.Join(dir, "agentlink.yml"), Sample())
	if _, err := Find("", dir); err == nil || !strings.Contains(err.Error(), "multiple configurations") {
		t.Fatalf("Find() error = %v, want ambiguity", err)
	}
}

func TestSchemaCopyIsCurrent(t *testing.T) {
	t.Parallel()

	embedded := Schema()
	if !json.Valid(embedded) {
		t.Fatal("embedded Schema() is not valid JSON")
	}
	checkedIn, err := os.ReadFile(filepath.Join("..", "..", "agentlink.schema.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(agentlink.schema.json): %v", err)
	}
	if !bytes.Equal(embedded, checkedIn) {
		t.Fatal("agentlink.schema.json differs from internal/config/schema.json")
	}
}

func TestFindWalksToParent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "agentlink.yaml"), Sample())
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("os.MkdirAll(%q): %v", nested, err)
	}
	got, err := Find("", nested)
	if err != nil {
		t.Fatalf("Find(%q): %v", nested, err)
	}
	if want := filepath.Join(dir, "agentlink.yaml"); got != want {
		t.Errorf("Find() = %q, want %q", got, want)
	}
}

func TestLoadFollowsFinalConfigSymlink(t *testing.T) {
	t.Parallel()

	realDir := t.TempDir()
	linkDir := t.TempDir()
	realPath := filepath.Join(realDir, "config.yaml")
	writeFile(t, realPath, Sample())
	linkPath := filepath.Join(linkDir, "agentlink.yaml")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("os.Symlink() unavailable: %v", err)
	}
	doc, err := Load(linkPath, linkDir)
	if err != nil {
		t.Fatalf("Load(symlink): %v", err)
	}
	if doc.Path != realPath {
		t.Errorf("Document.Path = %q, want resolved %q", doc.Path, realPath)
	}
}

func TestLoadExpandsEnvironmentSource(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AGENTLINK_TEST_ROOT", dir)
	path := filepath.Join(dir, "agentlink.yaml")
	data := strings.Replace(string(Sample()), "root: .", "root: ${AGENTLINK_TEST_ROOT}", 1)
	writeFile(t, path, []byte(data))
	doc, err := Load(path, dir)
	if err != nil {
		t.Fatalf("Load(environment source): %v", err)
	}
	if got := doc.Roots["project"]; got != dir {
		t.Errorf("expanded project root = %q, want %q", got, dir)
	}
}

func TestLoadRejectsMissingEnvironmentSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	data := strings.Replace(string(Sample()), "root: .", "root: ${AGENTLINK_VARIABLE_THAT_DOES_NOT_EXIST}", 1)
	writeFile(t, path, []byte(data))
	if _, err := Load(path, dir); err == nil || !strings.Contains(err.Error(), "is not set") {
		t.Fatalf("Load(missing environment source) error = %v", err)
	}
}

func TestLoadRejectsMultipleDocuments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	writeFile(t, path, append(Sample(), []byte("---\nversion: 2\n")...))
	if _, err := Load(path, dir); err == nil || !strings.Contains(err.Error(), "multiple YAML documents") {
		t.Fatalf("Load(multiple documents) error = %v", err)
	}
}

func TestLoadRejectsOversizedConfiguration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	writeFile(t, path, bytes.Repeat([]byte{'x'}, maxConfigFileSize+1))
	if _, err := Load(path, dir); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("Load(oversized) error = %v, want size limit", err)
	}
}

func TestLoadRejectsAliasedAndOverlappingEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    string
		claude  string
		codex   string
		wantErr string
	}{
		{"aliased file", "file", "same.md", "same.md", "same path"},
		{"overlapping trees", "tree", "skills", "skills/codex", "overlap"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "agentlink.yaml")
			data := []byte("version: 2\nsources:\n  a: {root: .}\n  b: {root: .}\npairs:\n  - id: peers\n    kind: " + test.kind + "\n    peers:\n      claude: {source: a, path: " + test.claude + "}\n      codex: {source: b, path: " + test.codex + "}\n")
			writeFile(t, path, data)
			if _, err := Load(path, dir); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Load() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestLoadRejectsOverlappingEndpointsThroughSourceSymlink(t *testing.T) {
	t.Parallel()

	realRoot := t.TempDir()
	aliasDir := t.TempDir()
	aliasRoot := filepath.Join(aliasDir, "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("os.Symlink() unavailable: %v", err)
	}
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "agentlink.yaml")
	data := []byte("version: 2\nsources:\n  real: {root: " + realRoot + "}\n  alias: {root: " + aliasRoot + "}\npairs:\n  - id: skills\n    kind: tree\n    peers:\n      claude: {source: real, path: skills}\n      codex: {source: alias, path: skills/codex}\n")
	writeFile(t, configPath, data)

	if _, err := Load(configPath, configDir); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("Load(symlink aliases) error = %v, want overlap", err)
	}
}

func writeFile(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}
}

func TestLoadRejectsLegacyEndpointKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agentlink.yaml")
	legacy := "version: 1\nsources:\n  project: {root: .}\npairs:\n  - id: skills\n    kind: tree\n    claude: {source: project, path: .claude/skills}\n    codex: {source: project, path: .codex/skills}\n"
	writeFile(t, path, []byte(legacy))
	_, err := Load(path, dir)
	if err == nil {
		t.Fatal("Load() succeeded with version-1 claude:/codex: keys")
	}
	if got := err.Error(); !strings.Contains(got, "peers:") || !strings.Contains(got, "version: 2") {
		t.Fatalf("Load() error = %q, want migration guidance naming peers: and version: 2", got)
	}
}
