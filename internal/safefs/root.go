package safefs

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Root is a named, confined filesystem tree.
type Root struct {
	name string
	path string
	root *os.Root
}

// File describes a regular file beneath a root.
type File struct {
	Path string
	Mode fs.FileMode
	Size int64
}

// Name returns the source name from configuration.
func (r *Root) Name() string { return r.name }

// Path returns the absolute host path used to open this root.
func (r *Root) Path() string { return r.path }

// Abs returns a display path. It must never be used to bypass Root methods.
func (r *Root) Abs(rel string) string {
	return filepath.Join(r.path, filepath.FromSlash(rel))
}

// Stat returns metadata while staying beneath the root.
func (r *Root) Stat(rel string) (fs.FileInfo, error) {
	return r.root.Stat(native(rel))
}

// Lstat returns metadata without following the final symlink.
func (r *Root) Lstat(rel string) (fs.FileInfo, error) {
	return r.root.Lstat(native(rel))
}

// Readlink returns the final symlink target without following it.
func (r *Root) Readlink(rel string) (string, error) {
	return r.root.Readlink(native(rel))
}

// ReadFile reads a bounded regular file.
func (r *Root) ReadFile(rel string, maxSize int64) ([]byte, fs.FileMode, error) {
	info, err := r.root.Stat(native(rel))
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, info.Mode(), fmt.Errorf("%s is not a regular file", rel)
	}
	if info.Size() > maxSize {
		return nil, info.Mode(), fmt.Errorf("%s is %d bytes; limit is %d", rel, info.Size(), maxSize)
	}
	file, err := r.root.Open(native(rel))
	if err != nil {
		return nil, info.Mode(), err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, info.Mode(), err
	}
	if int64(len(data)) > maxSize {
		return nil, info.Mode(), fmt.Errorf("%s grew beyond the %d-byte limit while being read", rel, maxSize)
	}
	return data, info.Mode(), nil
}

// WalkFiles returns regular files and symlinks below base in deterministic slash-path order.
// ignored receives paths relative to base and may skip files or whole trees.
func (r *Root) WalkFiles(base string, maxFiles int, ignored func(string) bool) ([]File, error) {
	info, err := r.root.Stat(native(base))
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", base)
	}
	var files []File
	err = fs.WalkDir(r.root.FS(), base, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == base {
			return nil
		}
		rel := filePath
		if base != "." {
			rel = strings.TrimPrefix(filePath, base+"/")
		}
		if ignored(rel) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && info.Mode()&fs.ModeSymlink == 0 {
			return nil
		}
		if len(files) >= maxFiles {
			return fmt.Errorf("tree contains more than %d files", maxFiles)
		}
		files = append(files, File{Path: rel, Mode: info.Mode(), Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// WriteFileAtomic replaces rel with data without exposing a partial file.
func (r *Root) WriteFileAtomic(rel string, data []byte, mode fs.FileMode) (err error) {
	rel = path.Clean(rel)
	if !fs.ValidPath(rel) || rel == "." {
		return fmt.Errorf("invalid target path %q", rel)
	}
	if info, statErr := r.root.Lstat(native(rel)); statErr == nil && info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refuse to replace symlink %s", rel)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect target %s: %w", rel, statErr)
	}

	dir := path.Dir(rel)
	if dir != "." {
		if err := r.root.MkdirAll(native(dir), 0o750); err != nil {
			return fmt.Errorf("create parent %s: %w", dir, err)
		}
	}
	temp := path.Join(dir, "."+path.Base(rel)+".agentlink-"+strings.ToLower(rand.Text()))
	file, err := r.root.OpenFile(native(temp), os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", rel, err)
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = r.root.Remove(native(temp))
		}
	}()
	if _, err = file.Write(data); err != nil {
		return fmt.Errorf("write temporary file for %s: %w", rel, err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("sync temporary file for %s: %w", rel, err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", rel, err)
	}
	if err = r.root.Rename(native(temp), native(rel)); err != nil {
		return fmt.Errorf("replace %s: %w", rel, err)
	}
	return nil
}

// RemoveFile removes a regular file. Directories and symlinks are refused.
func (r *Root) RemoveFile(rel string) error {
	info, err := r.root.Lstat(native(rel))
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to remove non-regular file %s", rel)
	}
	if err := r.root.Remove(native(rel)); err != nil {
		return err
	}
	return nil
}

// MkdirAll creates a directory tree beneath the root.
func (r *Root) MkdirAll(rel string, mode fs.FileMode) error {
	if !fs.ValidPath(rel) {
		return fmt.Errorf("invalid directory path %q", rel)
	}
	return r.root.MkdirAll(native(rel), mode.Perm())
}

// CopyFile copies one bounded file atomically between roots.
func CopyFile(source *Root, sourcePath string, target *Root, targetPath string, maxSize int64) error {
	info, err := source.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", source.Abs(sourcePath), err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refuse to copy symlink %s; configure a live activation instead", source.Abs(sourcePath))
	}
	data, mode, err := source.ReadFile(sourcePath, maxSize)
	if err != nil {
		return fmt.Errorf("read %s: %w", source.Abs(sourcePath), err)
	}
	if err := target.WriteFileAtomic(targetPath, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", target.Abs(targetPath), err)
	}
	return nil
}

func native(rel string) string {
	return filepath.FromSlash(rel)
}

var _ io.Closer = (*os.Root)(nil)
