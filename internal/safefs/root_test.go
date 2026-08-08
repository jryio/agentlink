package safefs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/config"
)

func TestRootOperations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	doc := &config.Document{
		Roots: map[string]string{"local": dir, "cloud": filepath.Join(dir, "missing")},
		Config: config.Config{Sources: map[string]config.Source{
			"local": {Root: dir}, "cloud": {Root: "missing", Optional: true},
		}},
	}
	roots, err := Open(doc)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	t.Cleanup(func() {
		if err := roots.Close(); err != nil {
			t.Errorf("Set.Close(): %v", err)
		}
	})
	if roots.Available("cloud") {
		t.Fatal("optional missing cloud source is available")
	}
	if len(roots.Unavailable()) != 1 {
		t.Fatalf("Unavailable() = %v, want cloud", roots.Unavailable())
	}
	if _, err := roots.Root("cloud"); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("Root(cloud) error = %v", err)
	}
	if _, err := roots.Root("unknown"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("Root(unknown) error = %v", err)
	}

	root, err := roots.Root("local")
	if err != nil {
		t.Fatalf("Root(local): %v", err)
	}
	if root.Name() != "local" || root.Path() != dir {
		t.Errorf("Root identity = %q %q, want local %q", root.Name(), root.Path(), dir)
	}
	if err := root.MkdirAll("tree/nested", 0o750); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}
	if err := root.WriteFileAtomic("tree/nested/a.txt", []byte("alpha"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomic(): %v", err)
	}
	data, mode, err := root.ReadFile("tree/nested/a.txt", 10)
	if err != nil {
		t.Fatalf("ReadFile(): %v", err)
	}
	if string(data) != "alpha" || mode.Perm() != 0o640 {
		t.Errorf("ReadFile() = %q %o, want alpha 640", data, mode.Perm())
	}
	if _, _, err := root.ReadFile("tree/nested/a.txt", 2); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("ReadFile(over limit) error = %v", err)
	}
	if _, err := root.Stat("tree/nested/a.txt"); err != nil {
		t.Fatalf("Stat(): %v", err)
	}
	if _, err := root.Lstat("tree/nested/a.txt"); err != nil {
		t.Fatalf("Lstat(): %v", err)
	}

	if err := root.WriteFileAtomic("tree/ignored.log", []byte("ignore"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic(ignored): %v", err)
	}
	files, err := root.WalkFiles("tree", 10, func(path string) bool { return strings.HasSuffix(path, ".log") })
	if err != nil {
		t.Fatalf("WalkFiles(): %v", err)
	}
	if len(files) != 1 || files[0].Path != "nested/a.txt" {
		t.Errorf("WalkFiles() = %+v, want nested/a.txt", files)
	}
	if _, err := root.WalkFiles("tree", 0, func(string) bool { return false }); err == nil || !strings.Contains(err.Error(), "more than 0") {
		t.Fatalf("WalkFiles(limit) error = %v", err)
	}

	if err := CopyFile(root, "tree/nested/a.txt", root, "copy.txt", 10); err != nil {
		t.Fatalf("CopyFile(): %v", err)
	}
	if err := root.RemoveFile("copy.txt"); err != nil {
		t.Fatalf("RemoveFile(): %v", err)
	}
	if err := root.RemoveFile("tree"); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("RemoveFile(directory) error = %v", err)
	}

	symlink := filepath.Join(dir, "target-link")
	if err := os.Symlink("tree/nested/a.txt", symlink); err == nil {
		if target, err := root.Readlink("target-link"); err != nil || target != "tree/nested/a.txt" {
			t.Errorf("Readlink() = %q, %v", target, err)
		}
		if err := root.WriteFileAtomic("target-link", []byte("replace"), 0o600); err == nil || !strings.Contains(err.Error(), "refuse") {
			t.Fatalf("WriteFileAtomic(symlink) error = %v", err)
		}
		if err := CopyFile(root, "target-link", root, "copied-link", 1024); err == nil || !strings.Contains(err.Error(), "refuse to copy symlink") {
			t.Fatalf("CopyFile(symlink) error = %v", err)
		}
	}
}

func TestOpenRequiredMissingRootFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	doc := &config.Document{
		Roots:  map[string]string{"required": filepath.Join(dir, "missing")},
		Config: config.Config{Sources: map[string]config.Source{"required": {Root: "missing"}}},
	}
	if _, err := Open(doc); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Open(required missing) error = %v, want not exist", err)
	}
}
