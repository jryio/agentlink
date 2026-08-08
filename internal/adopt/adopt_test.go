package adopt

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAdoptsSkillDirectory(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "go-testing", "SKILL.md"), "# Testing\n")

	destination, err := DestinationFor(".claude/skills", "")
	if err != nil {
		t.Fatalf("DestinationFor(): %v", err)
	}
	if destination != ".agents/skills" {
		t.Fatalf("DestinationFor() = %q, want .agents/skills", destination)
	}
	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if plan.Managed || plan.Overwrite || !plan.Directory {
		t.Fatalf("NewPlan() = %#v, want unmanaged directory without overwrite", plan)
	}
	if err := Apply(plan, false); err != nil {
		t.Fatalf("Apply(): %v", err)
	}

	assertSymlink(t, filepath.Join(project, ".claude", "skills"), filepath.Join("..", ".agents", "skills"))
	got := readProjectFile(t, project, filepath.Join(".agents", "skills", "go-testing", "SKILL.md"))
	if string(got) != "# Testing\n" {
		t.Fatalf("managed skill = %q, want copied content", got)
	}
	got = readProjectFile(t, project, filepath.Join(".claude", "skills", "go-testing", "SKILL.md"))
	if string(got) != "# Testing\n" {
		t.Fatalf("linked skill = %q, want copied content", got)
	}
}

func TestApplyAdoptsAgentSpecificFile(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "settings.local.json"), "{\"permissions\":{}}\n")

	plan, err := NewPlan(project, ".claude/settings.local.json", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if plan.Destination != ".agents/claude/settings.local.json" {
		t.Fatalf("plan destination = %q, want agent-specific target", plan.Destination)
	}
	if err := Apply(plan, false); err != nil {
		t.Fatalf("Apply(): %v", err)
	}

	assertSymlink(t, filepath.Join(project, ".claude", "settings.local.json"), filepath.Join("..", ".agents", "claude", "settings.local.json"))
}

func TestApplyRequiresForceToReplaceManagedConfiguration(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".codex", "skills", "review", "SKILL.md"), "# New\n")
	writeFile(t, filepath.Join(project, ".agents", "skills", "review", "SKILL.md"), "# Old\n")

	plan, err := NewPlan(project, ".codex/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if !plan.Overwrite {
		t.Fatalf("NewPlan() = %#v, want overwrite warning", plan)
	}
	if err := Apply(plan, false); err == nil || !strings.Contains(err.Error(), "--force --apply") {
		t.Fatalf("Apply() without force = %v, want overwrite refusal", err)
	}
	current := readProjectFile(t, project, filepath.Join(".agents", "skills", "review", "SKILL.md"))
	if string(current) != "# Old\n" {
		t.Fatalf("managed skill after refused Apply = %q, want original content", current)
	}
	if err := Apply(plan, true); err != nil {
		t.Fatalf("Apply() with force: %v", err)
	}
	current = readProjectFile(t, project, filepath.Join(".agents", "skills", "review", "SKILL.md"))
	if string(current) != "# New\n" {
		t.Fatalf("replaced managed skill = %q, want source content", current)
	}
	assertSymlink(t, filepath.Join(project, ".codex", "skills"), filepath.Join("..", ".agents", "skills"))
}

func TestNewPlanRecognizesManagedLink(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".agents", "skills", "go", "SKILL.md"), "# Go\n")
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.claude): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".agents", "skills"), filepath.Join(project, ".claude", "skills")); err != nil {
		t.Fatalf("os.Symlink(.claude/skills): %v", err)
	}

	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if !plan.Managed {
		t.Fatalf("NewPlan() = %#v, want already managed", plan)
	}
	if err := Apply(plan, false); err != nil {
		t.Fatalf("Apply(managed): %v", err)
	}
}

func TestNewPlanRecognizesCloudxStyleSkillLinks(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".agents", "skills", "go-testing", "SKILL.md"), "# Testing\n")
	if err := os.MkdirAll(filepath.Join(project, ".claude", "skills"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.claude/skills): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "..", ".agents", "skills", "go-testing"), filepath.Join(project, ".claude", "skills", "go-testing")); err != nil {
		t.Fatalf("os.Symlink(.claude/skills/go-testing): %v", err)
	}

	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if !plan.Managed {
		t.Fatalf("NewPlan() = %#v, want Cloudx-style skill links recognized as managed", plan)
	}
}

func TestNewPlanRejectsUnsafeSourcePaths(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "SKILL.md"), "# Skill\n")
	for _, source := range []string{"", "..", "../outside", "/tmp/skills"} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			if _, err := NewPlan(project, source, ""); err == nil {
				t.Fatalf("NewPlan(%q) succeeded, want invalid path error", source)
			}
		})
	}
}

func TestApplyCopiesInternalSourceTreeSymlink(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "real", "SKILL.md"), "# Skill\n")
	if err := os.Symlink("real", filepath.Join(project, ".claude", "skills", "linked")); err != nil {
		t.Fatalf("os.Symlink(): %v", err)
	}

	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if err := Apply(plan, false); err != nil {
		t.Fatalf("Apply(): %v", err)
	}
	contents := readProjectFile(t, project, filepath.Join(".agents", "skills", "linked", "SKILL.md"))
	if string(contents) != "# Skill\n" {
		t.Fatalf("copied linked skill = %q, want source content", contents)
	}
}

func TestApplyRejectsExternalSourceTreeSymlink(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	external := t.TempDir()
	writeFile(t, filepath.Join(external, "skill", "SKILL.md"), "# Skill\n")
	if err := os.MkdirAll(filepath.Join(project, ".claude", "skills"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.claude/skills): %v", err)
	}
	if err := os.Symlink(filepath.Join(external, "skill"), filepath.Join(project, ".claude", "skills", "linked")); err != nil {
		t.Fatalf("os.Symlink(): %v", err)
	}

	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	err = Apply(plan, false)
	if err == nil || !strings.Contains(err.Error(), "external symlink") {
		t.Fatalf("Apply() = %v, want external symlink rejection", err)
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".claude", "skills")); statErr != nil {
		t.Fatalf("source removed after rejected Apply: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(project, ".agents", "skills")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("destination exists after rejected Apply: %v", statErr)
	}
}

func readProjectFile(t testing.TB, project, name string) []byte {
	t.Helper()
	root, err := os.OpenRoot(project)
	if err != nil {
		t.Fatalf("os.OpenRoot(%q): %v", project, err)
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

func assertSymlink(t testing.TB, filePath, want string) {
	t.Helper()
	info, err := os.Lstat(filePath)
	if err != nil {
		t.Fatalf("os.Lstat(%q): %v", filePath, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s mode = %v, want symbolic link", filePath, info.Mode())
	}
	got, err := os.Readlink(filePath)
	if err != nil {
		t.Fatalf("os.Readlink(%q): %v", filePath, err)
	}
	if got != want {
		t.Fatalf("os.Readlink(%q) = %q, want %q", filePath, got, want)
	}
}

func writeFile(t testing.TB, filePath, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(%q): %v", filepath.Dir(filePath), err)
	}
	if err := os.WriteFile(filePath, []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", filePath, err)
	}
}
