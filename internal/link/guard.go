package link

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jryio/agentlink/internal/config"
)

// Violation explains why a touched peer artifact is unsafe to finalize.
type Violation struct {
	Pair        string `json:"pair"`
	Relative    string `json:"relative"`
	State       State  `json:"state"`
	Message     string `json:"message"`
	Counterpart string `json:"counterpart,omitempty"`
}

// Guard checks only configured artifacts named in changed. It has no VCS
// dependency: integrations may provide paths from Git, another VCS, an editor,
// or a file-sync event stream.
func (e *Engine) Guard(ctx context.Context, changed []string) ([]Violation, error) {
	touched, err := e.touched(changed)
	if err != nil {
		return nil, err
	}
	if len(touched) == 0 {
		return nil, nil
	}
	selected := make(map[string]bool, len(touched))
	for pair := range touched {
		selected[pair] = true
	}
	report := e.Check(ctx, selected)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var violations []Violation
	for _, pairReport := range report.Pairs {
		relatives := touched[pairReport.ID]
		for _, finding := range pairReport.Findings {
			if finding.Relative != "." && !relatives[finding.Relative] {
				continue
			}
			keys := make([]string, 0, len(finding.Paths))
			for id := range finding.Paths {
				keys = append(keys, id)
			}
			slices.Sort(keys)
			counterpart := ""
			if finding.State == StateMissing && finding.Peer != "" {
				counterpart = finding.Paths[finding.Peer]
			} else if len(keys) == 2 {
				counterpart = finding.Paths[keys[0]] + " ↔ " + finding.Paths[keys[1]]
			}
			violations = append(violations, Violation{
				Pair:        finding.Pair,
				Relative:    finding.Relative,
				State:       finding.State,
				Message:     finding.Detail,
				Counterpart: counterpart,
			})
		}
	}
	slices.SortFunc(violations, func(a, b Violation) int {
		if a.Pair != b.Pair {
			return cmpString(a.Pair, b.Pair)
		}
		return cmpString(a.Relative, b.Relative)
	})
	return violations, nil
}

func (e *Engine) touched(changed []string) (map[string]map[string]bool, error) {
	result := make(map[string]map[string]bool)
	for _, raw := range changed {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		changedPath := raw
		if !filepath.IsAbs(changedPath) {
			changedPath = filepath.Join(e.doc.CWD, changedPath)
		}
		changedPath, err := filepath.Abs(changedPath)
		if err != nil {
			return nil, fmt.Errorf("resolve changed path %q: %w", raw, err)
		}
		changedPath = filepath.Clean(changedPath)
		for _, runtime := range e.pairs {
			pair := runtime.pair
			if pair.Kind == config.KindSiblings {
				for _, endpoint := range pair.Peers {
					rootPath := e.doc.Roots[endpoint.Source]
					rel, relErr := filepath.Rel(rootPath, changedPath)
					if relErr != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
						continue
					}
					rel = filepath.ToSlash(rel)
					if filepath.Base(changedPath) != endpoint.Path || runtime.ignored.Match(rel) {
						continue
					}
					dir := filepath.ToSlash(filepath.Dir(rel))
					if result[pair.ID] == nil {
						result[pair.ID] = make(map[string]bool)
					}
					result[pair.ID][dir] = true
				}
				continue
			}
			for _, endpoint := range pair.Peers {
				rootPath := e.doc.Roots[endpoint.Source]
				base := filepath.Join(rootPath, filepath.FromSlash(endpoint.Path))
				relative, match := endpointRelative(changedPath, base, pair.Kind)
				if !match || runtime.ignored.Match(relative) {
					continue
				}
				if result[pair.ID] == nil {
					result[pair.ID] = make(map[string]bool)
				}
				result[pair.ID][relative] = true
			}
		}
		for _, server := range e.doc.Config.MCPServers {
			for _, peer := range server.Peers {
				configuredPath := filepath.Join(e.doc.Roots[peer.Config.Source], filepath.FromSlash(peer.Config.Path))
				if changedPath != filepath.Clean(configuredPath) {
					continue
				}
				if result[server.ID] == nil {
					result[server.ID] = make(map[string]bool)
				}
				result[server.ID][mcpRelative(server)] = true
			}
		}
	}
	return result, nil
}

func endpointRelative(changedPath, base string, kind config.Kind) (string, bool) {
	if kind == config.KindFile {
		return ".", changedPath == filepath.Clean(base)
	}
	rel, err := filepath.Rel(base, changedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if rel == "." {
		return ".", true
	}
	return filepath.ToSlash(rel), true
}
