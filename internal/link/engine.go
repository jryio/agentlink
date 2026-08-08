// Package link compares and reconciles configured Claude/Codex peer artifacts.
package link

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"sync"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/pattern"
	"github.com/jryio/agentlink/internal/safefs"
)

// State classifies a peer-artifact comparison.
type State string

const (
	// StateMissingClaude means only the Claude peer is absent.
	StateMissingClaude State = "missing_claude"
	// StateMissingCodex means only the Codex peer is absent.
	StateMissingCodex State = "missing_codex"
	// StateMissingBoth means both required peers are absent.
	StateMissingBoth State = "missing_both"
	// StateDifferent means normalized peer content differs.
	StateDifferent State = "different"
	// StateError means comparison could not be completed safely.
	StateError State = "error"
)

// Finding is one actionable drift item.
type Finding struct {
	Pair     string `json:"pair"`
	Relative string `json:"relative"`
	State    State  `json:"state"`
	Claude   string `json:"claude"`
	Codex    string `json:"codex"`
	Detail   string `json:"detail,omitempty"`
}

// PairReport summarizes one configured pair.
type PairReport struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Files    int       `json:"files"`
	Skipped  bool      `json:"skipped,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Findings []Finding `json:"findings,omitempty"`
}

// Report is a deterministic point-in-time drift report.
type Report struct {
	Pairs       []PairReport       `json:"pairs"`
	Activations []ActivationReport `json:"activations,omitempty"`
}

// Clean reports whether no drift or comparison errors were found.
func (r Report) Clean() bool {
	for _, pair := range r.Pairs {
		if len(pair.Findings) > 0 {
			return false
		}
	}
	for _, activation := range r.Activations {
		if activation.State != "" {
			return false
		}
	}
	return true
}

// FindingCount returns the number of actionable findings.
func (r Report) FindingCount() int {
	count := 0
	for _, pair := range r.Pairs {
		count += len(pair.Findings)
	}
	for _, activation := range r.Activations {
		if activation.State != "" {
			count++
		}
	}
	return count
}

type pairRuntime struct {
	pair    config.Pair
	ignored pattern.Set
}

// Engine holds validated configuration and confined source roots.
type Engine struct {
	doc   *config.Document
	fs    *safefs.Set
	pairs []pairRuntime
}

// New constructs a comparison engine.
func New(doc *config.Document, roots *safefs.Set) (*Engine, error) {
	exceptions := make(map[string][]string)
	for _, exception := range doc.Config.Exceptions {
		exceptions[exception.Pair] = append(exceptions[exception.Pair], exception.Paths...)
	}
	pairs := make([]pairRuntime, 0, len(doc.Config.Pairs))
	for _, pair := range doc.Config.Pairs {
		patterns := slices.Concat(doc.Config.Ignore, pair.Ignore, exceptions[pair.ID])
		ignored, err := pattern.Compile(patterns)
		if err != nil {
			return nil, fmt.Errorf("compile ignore patterns for pair %q: %w", pair.ID, err)
		}
		pairs = append(pairs, pairRuntime{pair: pair, ignored: ignored})
	}
	return &Engine{doc: doc, fs: roots, pairs: pairs}, nil
}

// Check compares all selected pairs. An empty selection checks everything.
func (e *Engine) Check(ctx context.Context, selected map[string]bool) Report {
	tasks := make([]func() PairReport, 0, len(e.pairs)+len(e.doc.Config.MCPServers))
	for _, runtime := range e.pairs {
		if len(selected) > 0 && !selected[runtime.pair.ID] {
			continue
		}
		tasks = append(tasks, func() PairReport {
			if err := ctx.Err(); err != nil {
				return errorReport(runtime.pair, err)
			}
			return e.checkPair(ctx, runtime)
		})
	}
	for _, server := range e.doc.Config.MCPServers {
		if len(selected) > 0 && !selected[server.ID] {
			continue
		}
		tasks = append(tasks, func() PairReport {
			if err := ctx.Err(); err != nil {
				return mcpErrorReport(server, err)
			}
			return e.checkMCP(server)
		})
	}
	results := make([]PairReport, len(tasks))
	var wait sync.WaitGroup
	for i, task := range tasks {
		wait.Go(func() { results[i] = task() })
	}
	wait.Wait()
	activationTasks := make([]func() ActivationReport, 0, len(e.doc.Config.Activations))
	for _, activation := range e.doc.Config.Activations {
		if len(selected) > 0 && !selected[activation.ID] {
			continue
		}
		activationTasks = append(activationTasks, func() ActivationReport {
			if err := ctx.Err(); err != nil {
				return activationErrorReport(activation, err)
			}
			return e.checkActivation(activation)
		})
	}
	activations := make([]ActivationReport, len(activationTasks))
	for i, task := range activationTasks {
		wait.Go(func() { activations[i] = task() })
	}
	wait.Wait()
	return Report{Pairs: results, Activations: activations}
}

func (e *Engine) checkPair(ctx context.Context, runtime pairRuntime) PairReport {
	pair := runtime.pair
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	if runtime.ignored.Match(".") {
		report.Skipped = true
		report.Reason = "pair root is ignored or intentionally excepted"
		return report
	}
	if !e.fs.Available(pair.Claude.Source) || !e.fs.Available(pair.Codex.Source) {
		report.Skipped = true
		report.Reason = "optional source unavailable"
		return report
	}
	claudeRoot, err := e.fs.Root(pair.Claude.Source)
	if err != nil {
		return errorReport(pair, err)
	}
	codexRoot, err := e.fs.Root(pair.Codex.Source)
	if err != nil {
		return errorReport(pair, err)
	}
	if pair.Kind == "file" {
		report.Files = 1
		finding := e.compareFile(ctx, runtime, claudeRoot, pair.Claude.Path, codexRoot, pair.Codex.Path, ".")
		if finding != nil {
			if finding.State != StateMissingBoth || !pair.Optional {
				report.Findings = append(report.Findings, *finding)
			}
		}
		return report
	}
	if pair.Kind == "siblings" {
		return e.checkSiblings(ctx, runtime, claudeRoot, codexRoot)
	}
	return e.checkTree(ctx, runtime, claudeRoot, codexRoot)
}

func (e *Engine) checkSiblings(
	ctx context.Context,
	runtime pairRuntime,
	claudeRoot, codexRoot *safefs.Root,
) PairReport {
	pair := runtime.pair
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	claudeDirs, err := siblingDirs(claudeRoot, pair.Claude.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if err != nil {
		report.Findings = append(report.Findings, e.finding(pair, ".", StateError, claudeRoot, codexRoot, "discover Claude siblings: "+err.Error()))
		return report
	}
	codexDirs, err := siblingDirs(codexRoot, pair.Codex.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if err != nil {
		report.Findings = append(report.Findings, e.finding(pair, ".", StateError, claudeRoot, codexRoot, "discover Codex siblings: "+err.Error()))
		return report
	}
	dirs := make(map[string]struct{}, len(claudeDirs)+len(codexDirs))
	for dir := range claudeDirs {
		dirs[dir] = struct{}{}
	}
	for dir := range codexDirs {
		dirs[dir] = struct{}{}
	}
	if len(dirs) == 0 {
		if !pair.Optional {
			report.Findings = append(report.Findings, e.finding(pair, ".", StateMissingBoth, claudeRoot, codexRoot, "no sibling instruction files found"))
		}
		return report
	}
	directories := mapsKeys(dirs)
	slices.Sort(directories)
	for _, dir := range directories {
		if err := ctx.Err(); err != nil {
			report.Findings = append(report.Findings, e.finding(pair, dir, StateError, claudeRoot, codexRoot, err.Error()))
			break
		}
		finding := e.compareFile(
			ctx,
			runtime,
			claudeRoot,
			peerPath(pair, pair.Claude, dir),
			codexRoot,
			peerPath(pair, pair.Codex, dir),
			dir,
		)
		if finding != nil {
			report.Findings = append(report.Findings, *finding)
		}
		report.Files++
	}
	return report
}

func (e *Engine) checkTree(
	ctx context.Context,
	runtime pairRuntime,
	claudeRoot, codexRoot *safefs.Root,
) PairReport {
	pair := runtime.pair
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	claudeFiles, claudeExists, claudeErr := treeFiles(claudeRoot, pair.Claude.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	codexFiles, codexExists, codexErr := treeFiles(codexRoot, pair.Codex.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if claudeErr != nil {
		report.Findings = append(report.Findings, e.finding(pair, ".", StateError, claudeRoot, codexRoot, claudeErr.Error()))
		return report
	}
	if codexErr != nil {
		report.Findings = append(report.Findings, e.finding(pair, ".", StateError, claudeRoot, codexRoot, codexErr.Error()))
		return report
	}
	if !claudeExists && !codexExists {
		if !pair.Optional {
			report.Findings = append(report.Findings, e.finding(pair, ".", StateMissingBoth, claudeRoot, codexRoot, "both trees are missing"))
		}
		return report
	}
	paths := make(map[string]struct{}, len(claudeFiles)+len(codexFiles))
	for rel := range claudeFiles {
		paths[rel] = struct{}{}
	}
	for rel := range codexFiles {
		paths[rel] = struct{}{}
	}
	relativePaths := mapsKeys(paths)
	slices.Sort(relativePaths)
	for _, rel := range relativePaths {
		if err := ctx.Err(); err != nil {
			report.Findings = append(report.Findings, e.finding(pair, rel, StateError, claudeRoot, codexRoot, err.Error()))
			break
		}
		finding := e.compareFile(
			ctx,
			runtime,
			claudeRoot,
			path.Join(pair.Claude.Path, rel),
			codexRoot,
			path.Join(pair.Codex.Path, rel),
			rel,
		)
		if finding != nil {
			report.Findings = append(report.Findings, *finding)
		}
		report.Files++
	}
	if len(paths) == 0 && claudeExists != codexExists && !pair.Optional {
		state := StateMissingCodex
		if !claudeExists {
			state = StateMissingClaude
		}
		report.Findings = append(report.Findings, e.finding(pair, ".", state, claudeRoot, codexRoot, "empty counterpart tree is missing"))
	}
	return report
}

func (e *Engine) compareFile(
	ctx context.Context,
	runtime pairRuntime,
	claudeRoot *safefs.Root,
	claudePath string,
	codexRoot *safefs.Root,
	codexPath, rel string,
) *Finding {
	if err := ctx.Err(); err != nil {
		finding := e.finding(runtime.pair, rel, StateError, claudeRoot, codexRoot, err.Error())
		return &finding
	}
	claudeData, _, claudeErr := claudeRoot.ReadFile(claudePath, e.doc.Config.MaxFileSize())
	codexData, _, codexErr := codexRoot.ReadFile(codexPath, e.doc.Config.MaxFileSize())
	claudeMissing := errors.Is(claudeErr, os.ErrNotExist)
	codexMissing := errors.Is(codexErr, os.ErrNotExist)
	switch {
	case claudeErr != nil && !claudeMissing:
		finding := e.finding(runtime.pair, rel, StateError, claudeRoot, codexRoot, "read Claude peer: "+claudeErr.Error())
		return &finding
	case codexErr != nil && !codexMissing:
		finding := e.finding(runtime.pair, rel, StateError, claudeRoot, codexRoot, "read Codex peer: "+codexErr.Error())
		return &finding
	case claudeMissing && codexMissing:
		finding := e.finding(runtime.pair, rel, StateMissingBoth, claudeRoot, codexRoot, "both files are missing")
		return &finding
	case claudeMissing:
		finding := e.finding(runtime.pair, rel, StateMissingClaude, claudeRoot, codexRoot, "Claude counterpart is missing")
		return &finding
	case codexMissing:
		finding := e.finding(runtime.pair, rel, StateMissingCodex, claudeRoot, codexRoot, "Codex counterpart is missing")
		return &finding
	}
	claudeNormalized, err := normalize(runtime.pair.Normalizer, claudeData)
	if err != nil {
		finding := e.finding(runtime.pair, rel, StateError, claudeRoot, codexRoot, "normalize Claude peer: "+err.Error())
		return &finding
	}
	codexNormalized, err := normalize(runtime.pair.Normalizer, codexData)
	if err != nil {
		finding := e.finding(runtime.pair, rel, StateError, claudeRoot, codexRoot, "normalize Codex peer: "+err.Error())
		return &finding
	}
	if sha256.Sum256(claudeNormalized) != sha256.Sum256(codexNormalized) {
		finding := e.finding(runtime.pair, rel, StateDifferent, claudeRoot, codexRoot, "normalized content differs")
		return &finding
	}
	return nil
}

func treeFiles(root *safefs.Root, base string, maxFiles int, ignored pattern.Set) (map[string]safefs.File, bool, error) {
	files, err := root.WalkFiles(base, maxFiles, ignored.Match)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, err
	}
	result := make(map[string]safefs.File, len(files))
	for _, file := range files {
		result[file.Path] = file
	}
	return result, true, nil
}

func (e *Engine) finding(
	pair config.Pair,
	rel string,
	state State,
	claudeRoot, codexRoot *safefs.Root,
	detail string,
) Finding {
	return Finding{
		Pair:     pair.ID,
		Relative: rel,
		State:    state,
		Claude:   claudeRoot.Abs(peerPath(pair, pair.Claude, rel)),
		Codex:    codexRoot.Abs(peerPath(pair, pair.Codex, rel)),
		Detail:   detail,
	}
}

func errorReport(pair config.Pair, err error) PairReport {
	return PairReport{
		ID:   pair.ID,
		Name: pairName(pair),
		Findings: []Finding{{
			Pair: pair.ID, State: StateError, Relative: ".", Detail: err.Error(),
		}},
	}
}

func endpointPath(base, rel string) string {
	if rel == "." {
		return base
	}
	return path.Join(base, rel)
}

func peerPath(pair config.Pair, endpoint config.Endpoint, rel string) string {
	if pair.Kind == "siblings" {
		if rel == "." {
			return endpoint.Path
		}
		return path.Join(rel, endpoint.Path)
	}
	return endpointPath(endpoint.Path, rel)
}

func siblingDirs(root *safefs.Root, fileName string, maxFiles int, ignored pattern.Set) (map[string]struct{}, error) {
	files, err := root.WalkFiles(".", maxFiles, ignored.Match)
	if err != nil {
		return nil, err
	}
	dirs := make(map[string]struct{})
	for _, file := range files {
		if path.Base(file.Path) == fileName {
			dirs[path.Dir(file.Path)] = struct{}{}
		}
	}
	return dirs, nil
}

func pairName(pair config.Pair) string {
	if pair.Name != "" {
		return pair.Name
	}
	return pair.ID
}

func mapsKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
