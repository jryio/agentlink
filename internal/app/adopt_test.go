package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdoptCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, ".claude", "skills", "go-testing", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(source): %v", err)
	}
	writeAppFile(t, source, "# Testing\n")

	out, _ := runCLI(t, dir, nil, "adopt", "--from", ".claude/skills")
	if !strings.Contains(out, ".claude/skills → .agents/skills") || !strings.Contains(out, "Plan only") {
		t.Fatalf("adopt preview output = %q, want plan", out)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("managed destination exists after preview: %v", err)
	}

	out, _ = runCLI(t, dir, nil, "adopt", "--from", ".claude/skills", "--apply")
	if !strings.Contains(out, "adopted .claude/skills") {
		t.Fatalf("adopt output = %q, want success", out)
	}
	info, err := os.Lstat(filepath.Join(dir, ".claude", "skills"))
	if err != nil {
		t.Fatalf("os.Lstat(linked source): %v", err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("source mode = %v, want symbolic link", info.Mode())
	}
	contents := readAppFile(t, dir, filepath.Join(".claude", "skills", "go-testing", "SKILL.md"))
	if string(contents) != "# Testing\n" {
		t.Fatalf("linked skill = %q, want source content", contents)
	}
}

func TestAdoptCommandWarnsBeforeOverwriting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for filePath, contents := range map[string]string{
		filepath.Join(dir, ".codex", "skills", "review", "SKILL.md"):  "# New\n",
		filepath.Join(dir, ".agents", "skills", "review", "SKILL.md"): "# Old\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			t.Fatalf("os.MkdirAll(%q): %v", filepath.Dir(filePath), err)
		}
		writeAppFile(t, filePath, contents)
	}

	out, _, err := runCLIErr(t, dir, nil, "adopt", "--from", ".codex/skills", "--apply")
	assertExitCode(t, err, exitUsage)
	if !strings.Contains(out, "warning:") || !strings.Contains(out, "--force --apply") {
		t.Fatalf("adopt overwrite output = %q, want warning", out)
	}

	runCLI(t, dir, nil, "adopt", "--from", ".codex/skills", "--apply", "--force")
	contents := readAppFile(t, dir, filepath.Join(".agents", "skills", "review", "SKILL.md"))
	if string(contents) != "# New\n" {
		t.Fatalf("replaced skill = %q, want selected source", contents)
	}
}

func TestAdoptCommandUsage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, args := range [][]string{
		{"adopt"},
		{"adopt", "--from", ".claude/skills", "extra"},
		{"adopt", "--unknown"},
	} {
		_, _, err := runCLIErr(t, dir, nil, args...)
		assertExitCode(t, err, exitUsage)
	}
}

func readAppFile(t testing.TB, rootPath, name string) []byte {
	t.Helper()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatalf("os.OpenRoot(%q): %v", rootPath, err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("root.Close(): %v", err)
		}
	})
	contents, err := root.ReadFile(name)
	if err != nil {
		t.Fatalf("root.ReadFile(%q): %v", name, err)
	}
	return contents
}
