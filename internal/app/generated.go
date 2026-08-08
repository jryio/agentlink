package app

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func checkGeneratedWritable(filePath string, force bool) (err error) {
	root, err := os.OpenRoot(filepath.Dir(filePath))
	if err != nil {
		return fmt.Errorf("open destination for %s: %w", filePath, err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	info, err := root.Lstat(filepath.Base(filePath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", filePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to replace symlink %s", filePath)
	}
	if !force {
		return fmt.Errorf("%s already exists; use --force to replace it", filePath)
	}
	return nil
}

func writeGenerated(filePath string, data []byte, force bool) (err error) {
	root, err := os.OpenRoot(filepath.Dir(filePath))
	if err != nil {
		return fmt.Errorf("open destination for %s: %w", filePath, err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := filepath.Base(filePath)
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	temp := "." + name + ".agentlink-" + strings.ToLower(rand.Text())
	file, err := root.OpenFile(temp, flags, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = root.Remove(temp)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", filePath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", filePath, err)
	}
	if info, statErr := root.Lstat(name); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse to replace symlink %s", filePath)
		}
		if !force {
			return fmt.Errorf("%s appeared while initializing; refusing to replace it", filePath)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect %s before replace: %w", filePath, statErr)
	}
	if err := root.Rename(temp, name); err != nil {
		return fmt.Errorf("replace %s: %w", filePath, err)
	}
	return nil
}
