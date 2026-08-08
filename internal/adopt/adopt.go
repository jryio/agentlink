// Package adopt moves project-local agent configuration into a durable .agents tree.
package adopt

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

const (
	maxFiles    = 25_000
	maxFileSize = 64 << 20
)

// Plan describes a non-destructive adoption operation.
type Plan struct {
	Project     string `json:"project"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Directory   bool   `json:"directory"`
	Managed     bool   `json:"managed"`
	Overwrite   bool   `json:"overwrite"`
}

// DestinationFor resolves the managed destination for a project-relative source.
// An explicit destination is always relative to .agents.
func DestinationFor(source, destination string) (string, error) {
	source, err := cleanRelative(source)
	if err != nil {
		return "", fmt.Errorf("source: %w", err)
	}
	if source == ".agents" {
		return "", errors.New("source must name an artifact beneath .agents, not .agents itself")
	}
	if strings.HasPrefix(source, ".agents/") {
		return source, nil
	}
	if destination != "" {
		destination, err = cleanRelative(destination)
		if err != nil {
			return "", fmt.Errorf("destination: %w", err)
		}
		destination = strings.TrimPrefix(destination, ".agents/")
		if destination == ".agents" || destination == "." {
			return "", errors.New("destination must name an artifact beneath .agents")
		}
		return path.Join(".agents", destination), nil
	}

	parts := strings.Split(source, "/")
	if len(parts) >= 2 && parts[1] == "skills" && strings.HasPrefix(parts[0], ".") {
		return path.Join(".agents", strings.Join(parts[1:], "/")), nil
	}
	if len(parts) >= 2 && strings.HasPrefix(parts[0], ".") {
		parts[0] = strings.TrimPrefix(parts[0], ".")
		return path.Join(".agents", strings.Join(parts, "/")), nil
	}
	return path.Join(".agents", source), nil
}

// NewPlan validates source and destination paths relative to project and
// reports whether applying would replace an existing managed artifact.
func NewPlan(project, source, destination string) (plan Plan, err error) {
	project, err = filepath.Abs(project)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project: %w", err)
	}
	source, err = cleanRelative(source)
	if err != nil {
		return Plan{}, fmt.Errorf("source: %w", err)
	}
	destination, err = DestinationFor(source, destination)
	if err != nil {
		return Plan{}, err
	}

	root, err := os.OpenRoot(project)
	if err != nil {
		return Plan{}, fmt.Errorf("open project: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if err := recoverTransaction(root, source, destination); err != nil {
		return Plan{}, err
	}

	sourceInfo, err := root.Lstat(native(source))
	if err != nil {
		return Plan{}, fmt.Errorf("inspect source %s: %w", source, err)
	}
	if !sourceInfo.IsDir() && !sourceInfo.Mode().IsRegular() && sourceInfo.Mode()&fs.ModeSymlink == 0 {
		return Plan{}, fmt.Errorf("source %s is neither a regular file nor a directory", source)
	}

	plan = Plan{
		Project:     project,
		Source:      source,
		Destination: destination,
		Directory:   sourceInfo.IsDir(),
	}
	if source == destination {
		if err := validateManagedArtifact(root, destination); err != nil {
			return Plan{}, err
		}
		plan.Managed = true
		return plan, nil
	}
	if sourceInfo.Mode()&fs.ModeSymlink != 0 {
		link, readErr := root.Readlink(native(source))
		if readErr != nil {
			return Plan{}, fmt.Errorf("read source link %s: %w", source, readErr)
		}
		if linkResolvesTo(source, link, destination) {
			if err := validateManagedArtifact(root, destination); err != nil {
				return Plan{}, err
			}
			plan.Managed = true
			return plan, nil
		}
		return Plan{}, fmt.Errorf("source %s is a symlink; use its real configuration path", source)
	}

	destinationInfo, statErr := root.Lstat(native(destination))
	switch {
	case errors.Is(statErr, os.ErrNotExist):
		return plan, nil
	case statErr != nil:
		return Plan{}, fmt.Errorf("inspect destination %s: %w", destination, statErr)
	case destinationInfo.Mode()&fs.ModeSymlink != 0:
		return Plan{}, fmt.Errorf("destination %s is a symlink; refusing to replace it", destination)
	case sourceInfo.IsDir() && destinationInfo.IsDir():
		managed, linked, matchErr := treeLinksToDestination(root, source, destination)
		if matchErr != nil {
			return Plan{}, matchErr
		}
		if managed && linked {
			plan.Managed = true
			return plan, nil
		}
		plan.Overwrite = true
		return plan, nil
	default:
		plan.Overwrite = true
		return plan, nil
	}
}

// Apply copies a planned source into .agents, then replaces the source with a
// relative symlink. An existing destination requires force.
func Apply(plan Plan, force bool) (err error) {
	root, err := os.OpenRoot(plan.Project)
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if err := recoverTransaction(root, plan.Source, plan.Destination); err != nil {
		return err
	}
	if plan.Managed {
		return nil
	}

	sourceInfo, err := root.Lstat(native(plan.Source))
	if err != nil {
		return fmt.Errorf("inspect source %s: %w", plan.Source, err)
	}
	if sourceInfo.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("source %s changed into a symlink", plan.Source)
	}
	if sourceInfo.IsDir() != plan.Directory || (!sourceInfo.IsDir() && !sourceInfo.Mode().IsRegular()) {
		return fmt.Errorf("source %s changed since planning", plan.Source)
	}

	destinationInfo, statErr := root.Lstat(native(plan.Destination))
	overwrite := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect destination %s: %w", plan.Destination, statErr)
	}
	if overwrite {
		if destinationInfo == nil {
			return fmt.Errorf("inspect destination %s: no file info", plan.Destination)
		}
		if destinationInfo.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("destination %s is a symlink; refusing to replace it", plan.Destination)
		}
	}
	if overwrite != plan.Overwrite {
		return errors.New("destination changed since planning; run adopt again")
	}
	if overwrite && !force {
		return fmt.Errorf("destination %s already exists; rerun with --force --apply to replace it", plan.Destination)
	}

	transaction := newTransaction(plan.Source, plan.Destination, overwrite)
	if err := beginTransaction(root, transaction); err != nil {
		return err
	}
	defer func() {
		if stageErr := removeArtifact(root, transaction.stage); stageErr != nil && err == nil {
			err = fmt.Errorf("remove staged destination %s: %w", transaction.stage, stageErr)
		}
	}()
	if err := copySource(root, plan.Source, transaction.stage); err != nil {
		return err
	}

	if err := root.Rename(native(plan.Source), native(transaction.sourceBackup)); err != nil {
		return fmt.Errorf("stage source %s: %w", plan.Source, err)
	}
	sourceStaged := true
	defer func() {
		if sourceStaged {
			if restoreErr := root.Rename(native(transaction.sourceBackup), native(plan.Source)); restoreErr != nil && err == nil {
				err = fmt.Errorf("restore source %s: %w", plan.Source, restoreErr)
			}
		}
	}()

	if overwrite {
		if err := root.Rename(native(plan.Destination), native(transaction.destinationBackup)); err != nil {
			return fmt.Errorf("stage destination %s: %w", plan.Destination, err)
		}
	}
	destinationStaged := true
	defer func() {
		if !destinationStaged {
			return
		}
		if removeErr := root.RemoveAll(native(plan.Destination)); removeErr != nil && err == nil {
			err = fmt.Errorf("remove replacement destination %s: %w", plan.Destination, removeErr)
			return
		}
		if overwrite {
			if restoreErr := root.Rename(native(transaction.destinationBackup), native(plan.Destination)); restoreErr != nil && err == nil {
				err = fmt.Errorf("restore destination %s: %w", plan.Destination, restoreErr)
			}
		}
	}()
	if err := root.Rename(native(transaction.stage), native(plan.Destination)); err != nil {
		return fmt.Errorf("publish destination %s: %w", plan.Destination, err)
	}

	linkTarget, err := filepath.Rel(filepath.Dir(native(plan.Source)), native(plan.Destination))
	if err != nil {
		return fmt.Errorf("resolve relative link from %s to %s: %w", plan.Source, plan.Destination, err)
	}
	if err := root.Symlink(linkTarget, native(plan.Source)); err != nil {
		return fmt.Errorf("link source %s: %w", plan.Source, err)
	}
	sourceStaged = false
	destinationStaged = false

	if err := finishTransaction(root, transaction); err != nil {
		return err
	}
	return nil
}

func treeLinksToDestination(root *os.Root, source, destination string) (bool, bool, error) {
	entries := 0
	linked := false
	var matches func(string, string) (bool, error)
	matches = func(sourcePath, destinationPath string) (bool, error) {
		sourceInfo, err := root.Lstat(native(sourcePath))
		if err != nil {
			return false, fmt.Errorf("inspect source entry %s: %w", sourcePath, err)
		}
		if sourceInfo.Mode()&fs.ModeSymlink != 0 {
			link, err := root.Readlink(native(sourcePath))
			if err != nil {
				return false, fmt.Errorf("read source link %s: %w", sourcePath, err)
			}
			if !linkResolvesTo(sourcePath, link, destinationPath) {
				return false, nil
			}
			if err := validateManagedArtifact(root, destinationPath); err != nil {
				return false, err
			}
			linked = true
			return true, nil
		}
		destinationInfo, err := root.Lstat(native(destinationPath))
		if err != nil {
			return false, fmt.Errorf("inspect destination entry %s: %w", destinationPath, err)
		}
		if destinationInfo.Mode()&fs.ModeSymlink != 0 ||
			sourceInfo.IsDir() != destinationInfo.IsDir() ||
			sourceInfo.Mode().IsRegular() != destinationInfo.Mode().IsRegular() {
			return false, nil
		}
		if sourceInfo.Mode().IsRegular() {
			return false, nil
		}
		if !sourceInfo.IsDir() {
			return false, nil
		}
		sourceEntries, err := fs.ReadDir(root.FS(), sourcePath)
		if err != nil {
			return false, fmt.Errorf("read source directory %s: %w", sourcePath, err)
		}
		destinationEntries, err := fs.ReadDir(root.FS(), destinationPath)
		if err != nil {
			return false, fmt.Errorf("read destination directory %s: %w", destinationPath, err)
		}
		if len(sourceEntries) != len(destinationEntries) {
			return false, nil
		}
		entries += len(sourceEntries)
		if entries > maxFiles {
			return false, fmt.Errorf("source tree contains more than %d entries", maxFiles)
		}
		for index, sourceEntry := range sourceEntries {
			if sourceEntry.Name() != destinationEntries[index].Name() {
				return false, nil
			}
			matched, err := matches(path.Join(sourcePath, sourceEntry.Name()), path.Join(destinationPath, destinationEntries[index].Name()))
			if err != nil || !matched {
				return matched, err
			}
		}
		return true, nil
	}
	matched, err := matches(source, destination)
	return matched, linked, err
}

func copySource(root *os.Root, source, target string) error {
	files := 0
	return copyNode(root, source, target, make(map[string]bool), &files)
}

func copyNode(root *os.Root, source, target string, active map[string]bool, files *int) error {
	if source == target || strings.HasPrefix(source, target+"/") {
		return fmt.Errorf("source link resolves into staged destination %s", target)
	}
	info, err := root.Lstat(native(source))
	if err != nil {
		return fmt.Errorf("inspect source entry %s: %w", source, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		resolved, err := resolveProjectLink(root, source)
		if err != nil {
			return err
		}
		return copyNode(root, resolved, target, active, files)
	}
	if info.Mode().IsRegular() {
		*files++
		if *files > maxFiles {
			return fmt.Errorf("source tree contains more than %d files", maxFiles)
		}
		return copyFile(root, source, target, info)
	}
	if !info.IsDir() {
		return fmt.Errorf("source tree contains unsupported entry %s", source)
	}
	if active[source] {
		return fmt.Errorf("source tree contains symlink cycle at %s", source)
	}
	active[source] = true
	defer delete(active, source)
	if err := root.MkdirAll(native(target), info.Mode().Perm()); err != nil {
		return fmt.Errorf("create destination directory %s: %w", target, err)
	}
	entries, err := fs.ReadDir(root.FS(), source)
	if err != nil {
		return fmt.Errorf("read source directory %s: %w", source, err)
	}
	for _, entry := range entries {
		if err := copyNode(root, path.Join(source, entry.Name()), path.Join(target, entry.Name()), active, files); err != nil {
			return err
		}
	}
	return nil
}

func resolveProjectLink(root *os.Root, source string) (string, error) {
	link, err := root.Readlink(native(source))
	if err != nil {
		return "", fmt.Errorf("read source link %s: %w", source, err)
	}
	if filepath.IsAbs(link) {
		return "", fmt.Errorf("source tree contains external symlink %s", source)
	}
	resolved, err := cleanRelative(filepath.Join(filepath.Dir(native(source)), link))
	if err != nil {
		return "", fmt.Errorf("source tree symlink %s escapes the project", source)
	}
	return resolved, nil
}

func copyFile(root *os.Root, source, target string, info fs.FileInfo) (err error) {
	if info.Size() > maxFileSize {
		return fmt.Errorf("source file %s is %d bytes; limit is %d", source, info.Size(), maxFileSize)
	}
	if err := root.MkdirAll(native(path.Dir(target)), 0o750); err != nil {
		return fmt.Errorf("create destination parent for %s: %w", target, err)
	}
	input, err := root.Open(native(source))
	if err != nil {
		return fmt.Errorf("open source %s: %w", source, err)
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	output, err := root.OpenFile(native(target), os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create destination %s: %w", target, err)
	}
	defer func() {
		closeErr := output.Close()
		if err != nil {
			_ = root.Remove(native(target))
		}
		err = errors.Join(err, closeErr)
	}()
	written, err := io.Copy(output, io.LimitReader(input, maxFileSize+1))
	if err != nil {
		return fmt.Errorf("copy %s: %w", source, err)
	}
	if written > maxFileSize {
		return fmt.Errorf("source file %s grew beyond the %d-byte limit while copying", source, maxFileSize)
	}
	if err := output.Sync(); err != nil {
		return fmt.Errorf("sync destination %s: %w", target, err)
	}
	return nil
}

func temporaryPath(root *os.Root, dir, prefix string) (string, error) {
	for range 32 {
		candidate := path.Join(dir, prefix+strings.ToLower(rand.Text()))
		_, err := root.Lstat(native(candidate))
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("inspect temporary path %s: %w", candidate, err)
		}
	}
	return "", errors.New("allocate temporary path: too many collisions")
}

func cleanRelative(value string) (string, error) {
	if value == "" {
		return "", errors.New("path must not be empty")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("path %q must be relative to the project", value)
	}
	value = filepath.ToSlash(filepath.Clean(value))
	if value == "." || !fs.ValidPath(value) || strings.Contains(value, `\`) {
		return "", fmt.Errorf("path %q must name an artifact beneath the project", value)
	}
	return value, nil
}

func linkResolvesTo(source, link, destination string) bool {
	if filepath.IsAbs(link) {
		return false
	}
	resolved := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(native(source)), link)))
	return resolved == destination
}

func native(name string) string {
	return filepath.FromSlash(name)
}
