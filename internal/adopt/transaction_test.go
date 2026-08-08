package adopt

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPlanRejectsDanglingManagedLink(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.claude): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".agents", "skills"), filepath.Join(project, ".claude", "skills")); err != nil {
		t.Fatalf("os.Symlink(.claude/skills): %v", err)
	}

	if _, err := NewPlan(project, ".claude/skills", ""); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewPlan() = %v, want missing managed destination error", err)
	}
}

func TestNewPlanRejectsLinkedManagedDestination(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".store", "skills", "SKILL.md"), "# Skill\n")
	if err := os.MkdirAll(filepath.Join(project, ".agents"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.agents): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".store", "skills"), filepath.Join(project, ".agents", "skills")); err != nil {
		t.Fatalf("os.Symlink(.agents/skills): %v", err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.claude): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", ".agents", "skills"), filepath.Join(project, ".claude", "skills")); err != nil {
		t.Fatalf("os.Symlink(.claude/skills): %v", err)
	}

	if _, err := NewPlan(project, ".claude/skills", ""); err == nil || !contains(err, "not a regular file or directory") {
		t.Fatalf("NewPlan() = %v, want linked managed destination rejection", err)
	}
}

func TestNewPlanDoesNotTreatDanglingCloudxSkillLinkAsManaged(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".agents", "skills"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(.agents/skills): %v", err)
	}
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
	if plan.Managed || !plan.Overwrite {
		t.Fatalf("NewPlan() = %#v, want unmanaged overwrite plan", plan)
	}
}

func TestNewPlanDoesNotTreatEmptyDirectoriesAsManaged(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	for _, directory := range []string{
		filepath.Join(project, ".agents", "skills"),
		filepath.Join(project, ".claude", "skills"),
	} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			t.Fatalf("os.MkdirAll(%q): %v", directory, err)
		}
	}

	plan, err := NewPlan(project, ".claude/skills", "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if plan.Managed || !plan.Overwrite {
		t.Fatalf("NewPlan() = %#v, want unmanaged overwrite plan", plan)
	}
}

func TestNewPlanRecoversInterruptedPublish(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "SKILL.md"), "# Source\n")
	writeFile(t, filepath.Join(project, ".agents", "skills", "SKILL.md"), "# Original\n")
	transaction := newTransaction(".claude/skills", ".agents/skills", true)
	beginTestTransaction(t, project, transaction)
	renameProjectPath(t, project, transaction.Source, transaction.sourceBackup)
	renameProjectPath(t, project, transaction.Destination, transaction.destinationBackup)
	writeFile(t, filepath.Join(project, filepath.FromSlash(transaction.Destination), "SKILL.md"), "# Published\n")

	plan, err := NewPlan(project, transaction.Source, "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if !plan.Overwrite || plan.Managed {
		t.Fatalf("NewPlan() = %#v, want recovered overwrite plan", plan)
	}
	if got := string(readProjectFile(t, project, filepath.Join(".claude", "skills", "SKILL.md"))); got != "# Source\n" {
		t.Fatalf("restored source = %q, want original source", got)
	}
	if got := string(readProjectFile(t, project, filepath.Join(".agents", "skills", "SKILL.md"))); got != "# Original\n" {
		t.Fatalf("restored destination = %q, want original destination", got)
	}
	assertProjectPathAbsent(t, project, transaction.record)
	assertProjectPathAbsent(t, project, transaction.sourceBackup)
	assertProjectPathAbsent(t, project, transaction.destinationBackup)
}

func TestNewPlanRecoversInterruptedNewDestination(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "SKILL.md"), "# Source\n")
	transaction := newTransaction(".claude/skills", ".agents/skills", false)
	beginTestTransaction(t, project, transaction)
	renameProjectPath(t, project, transaction.Source, transaction.sourceBackup)
	writeFile(t, filepath.Join(project, filepath.FromSlash(transaction.Destination), "SKILL.md"), "# Published\n")

	plan, err := NewPlan(project, transaction.Source, "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if plan.Overwrite || plan.Managed {
		t.Fatalf("NewPlan() = %#v, want recovered plan without destination", plan)
	}
	if got := string(readProjectFile(t, project, filepath.Join(".claude", "skills", "SKILL.md"))); got != "# Source\n" {
		t.Fatalf("restored source = %q, want original source", got)
	}
	assertProjectPathAbsent(t, project, transaction.Destination)
	assertProjectPathAbsent(t, project, transaction.record)
}

func TestNewPlanFinishesPublishedTransaction(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	writeFile(t, filepath.Join(project, ".claude", "skills", "SKILL.md"), "# Source\n")
	writeFile(t, filepath.Join(project, ".agents", "skills", "SKILL.md"), "# Original\n")
	transaction := newTransaction(".claude/skills", ".agents/skills", true)
	beginTestTransaction(t, project, transaction)
	renameProjectPath(t, project, transaction.Source, transaction.sourceBackup)
	renameProjectPath(t, project, transaction.Destination, transaction.destinationBackup)
	writeFile(t, filepath.Join(project, filepath.FromSlash(transaction.Destination), "SKILL.md"), "# Published\n")
	if err := os.Symlink(filepath.Join("..", ".agents", "skills"), filepath.Join(project, ".claude", "skills")); err != nil {
		t.Fatalf("os.Symlink(.claude/skills): %v", err)
	}

	plan, err := NewPlan(project, transaction.Source, "")
	if err != nil {
		t.Fatalf("NewPlan(): %v", err)
	}
	if !plan.Managed {
		t.Fatalf("NewPlan() = %#v, want completed managed plan", plan)
	}
	if got := string(readProjectFile(t, project, filepath.Join(".agents", "skills", "SKILL.md"))); got != "# Published\n" {
		t.Fatalf("published destination = %q, want published content", got)
	}
	assertProjectPathAbsent(t, project, transaction.record)
	assertProjectPathAbsent(t, project, transaction.sourceBackup)
	assertProjectPathAbsent(t, project, transaction.destinationBackup)
}

func beginTestTransaction(t testing.TB, project string, transaction transaction) {
	t.Helper()
	root, err := os.OpenRoot(project)
	if err != nil {
		t.Fatalf("os.OpenRoot(%q): %v", project, err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Errorf("root.Close(): %v", err)
		}
	}()
	if err := beginTransaction(root, transaction); err != nil {
		t.Fatalf("beginTransaction(): %v", err)
	}
}

func renameProjectPath(t testing.TB, project, oldPath, newPath string) {
	t.Helper()
	root, err := os.OpenRoot(project)
	if err != nil {
		t.Fatalf("os.OpenRoot(%q): %v", project, err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Errorf("root.Close(): %v", err)
		}
	}()
	if err := root.Rename(filepath.FromSlash(oldPath), filepath.FromSlash(newPath)); err != nil {
		t.Fatalf("root.Rename(%q, %q): %v", oldPath, newPath, err)
	}
}

func assertProjectPathAbsent(t testing.TB, project, name string) {
	t.Helper()
	root, err := os.OpenRoot(project)
	if err != nil {
		t.Fatalf("os.OpenRoot(%q): %v", project, err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Errorf("root.Close(): %v", err)
		}
	}()
	if _, err := root.Lstat(filepath.FromSlash(name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("root.Lstat(%q) = %v, want not exist", name, err)
	}
}

func contains(err error, text string) bool {
	return err != nil && len(text) > 0 && strings.Contains(err.Error(), text)
}
