package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAndSyncRepos(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	review := "---\nname: review\ndescription: Shared review\n---\nBody.\n"
	local := "---\nname: local\ndescription: Repo local\n---\nBody.\n"
	writeAppFile(t, filepath.Join(root, "skills", "review", "SKILL.md"), review)
	for _, repo := range []string{"repo-a", "repo-b"} {
		if err := os.MkdirAll(filepath.Join(root, repo, ".git"), 0o750); err != nil {
			t.Fatalf("os.MkdirAll(%s .git): %v", repo, err)
		}
	}
	writeAppFile(t, filepath.Join(root, "repo-a", ".agents", "skills", "local", "SKILL.md"), local)
	writeAppFile(t, filepath.Join(root, "agentlink.yaml"), `version: 2
sources:
  shared: {root: skills}
  project: {root: ., relative_to: cwd}
pairs:
  - id: cloudx-skills
    kind: tree
    base: agents
    peers:
      agents: {source: shared, path: .}
      codex: {source: project, path: .agents/skills}
    normalizer: skill
    sync: copy
`)

	output, _, err := runCLIErr(t, root, nil, "check", "--repos", ".")
	assertExitCode(t, err, exitDrift)
	if !strings.Contains(output, "=== repo-a") || !strings.Contains(output, "=== repo-b") {
		t.Fatalf("check --repos output = %q, want both repository headers", output)
	}

	runCLI(t, root, nil, "sync", "--repos", ".", "--from", "agents", "--apply")
	for _, path := range []string{
		filepath.Join(root, "repo-a", ".agents", "skills", "review", "SKILL.md"),
		filepath.Join(root, "repo-b", ".agents", "skills", "review", "SKILL.md"),
		filepath.Join(root, "repo-a", ".agents", "skills", "local", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("os.Stat(%q): %v", path, err)
		}
	}

	runCLI(t, root, nil, "check", "--repos", ".")
	output, _ = runCLI(t, root, nil, "--json", "check", "--repos", ".")
	var result struct {
		Repos []repoCheckOutput `json:"repos"`
		Clean bool              `json:"clean"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode JSON check result: %v\noutput: %s", err, output)
	}
	if !result.Clean || len(result.Repos) != 2 {
		t.Fatalf("JSON check result = %+v, want two clean repositories", result)
	}

	empty := t.TempDir()
	_, _, err = runCLIErr(t, empty, nil, "check", "--repos", ".")
	var exitErr *ExitError
	if err == nil || errors.As(err, &exitErr) || !strings.Contains(err.Error(), "no repositories") {
		t.Fatalf("check empty --repos error = %v, want plain no repositories error", err)
	}
}
