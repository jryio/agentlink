// Package safefs provides root-confined access to configured source trees.
package safefs

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/jryio/agentlink/internal/config"
)

// Set owns the open roots for one command. Call Close when finished.
type Set struct {
	roots       map[string]*Root
	unavailable map[string]error
}

// Open confines every configured source with os.Root. Optional missing roots
// are recorded as unavailable; all other failures abort the operation.
func Open(doc *config.Document) (*Set, error) {
	set := &Set{
		roots:       make(map[string]*Root, len(doc.Roots)),
		unavailable: make(map[string]error),
	}
	names := make([]string, 0, len(doc.Roots))
	for name := range doc.Roots {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		root, err := os.OpenRoot(doc.Roots[name])
		if err != nil {
			if doc.Config.Sources[name].Optional && errors.Is(err, os.ErrNotExist) {
				set.unavailable[name] = err
				continue
			}
			_ = set.Close()
			return nil, fmt.Errorf("open source %q at %s: %w", name, doc.Roots[name], err)
		}
		set.roots[name] = &Root{name: name, path: doc.Roots[name], root: root}
	}
	return set, nil
}

// Root returns a configured open source.
func (s *Set) Root(name string) (*Root, error) {
	if root := s.roots[name]; root != nil {
		return root, nil
	}
	if err := s.unavailable[name]; err != nil {
		return nil, fmt.Errorf("source %q is unavailable: %w", name, err)
	}
	return nil, fmt.Errorf("source %q is not configured", name)
}

// Available reports whether a source root was opened.
func (s *Set) Available(name string) bool {
	return s.roots[name] != nil
}

// Unavailable returns optional sources that could not be opened.
func (s *Set) Unavailable() map[string]error {
	result := make(map[string]error, len(s.unavailable))
	for name, err := range s.unavailable {
		result[name] = err
	}
	return result
}

// Close releases all source handles.
func (s *Set) Close() error {
	errs := make([]error, 0, len(s.roots))
	for _, root := range s.roots {
		if err := root.root.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close source %q: %w", root.name, err))
		}
	}
	return errors.Join(errs...)
}
