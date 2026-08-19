// Package link compares and reconciles configured peer artifacts between any
// registered coding agents (see internal/agent), including the canonical
// .agents hub.
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

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/pattern"
	"github.com/jryio/agentlink/internal/safefs"
)

const maxConcurrentChecks = 8

// State classifies a peer-artifact comparison.
type State string

const (
	// StateMissing means exactly one peer is absent; Finding.Peer names it.
	StateMissing State = "missing"
	// StateMissingBoth means both required peers are absent.
	StateMissingBoth State = "missing_both"
	// StateDifferent means normalized peer content differs.
	StateDifferent State = "different"
	// StateError means comparison could not be completed safely.
	StateError State = "error"
)

// Finding is one actionable drift item.
type Finding struct {
	Pair     string            `json:"pair"`
	Relative string            `json:"relative"`
	State    State             `json:"state"`
	Peer     string            `json:"peer,omitempty"`
	Paths    map[string]string `json:"paths"`
	Detail   string            `json:"detail,omitempty"`
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
	pair      config.Pair
	ignored   pattern.Set
	left      string
	right     string
	leftSpec  agent.Spec
	rightSpec agent.Spec
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
		ids := pair.PeerIDs()
		leftSpec, ok := agent.Get(ids[0])
		if !ok {
			return nil, fmt.Errorf("pair %q: unknown agent %q", pair.ID, ids[0])
		}
		rightSpec, ok := agent.Get(ids[1])
		if !ok {
			return nil, fmt.Errorf("pair %q: unknown agent %q", pair.ID, ids[1])
		}
		pairs = append(pairs, pairRuntime{
			pair:      pair,
			ignored:   ignored,
			left:      ids[0],
			right:     ids[1],
			leftSpec:  leftSpec,
			rightSpec: rightSpec,
		})
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
	results := runTasks(tasks)
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
	activations := runTasks(activationTasks)
	return Report{Pairs: results, Activations: activations}
}

func runTasks[T any](tasks []func() T) []T {
	results := make([]T, len(tasks))
	jobs := make(chan int)
	var wait sync.WaitGroup
	for range min(maxConcurrentChecks, len(tasks)) {
		wait.Go(func() {
			for i := range jobs {
				results[i] = tasks[i]()
			}
		})
	}
	for i := range tasks {
		jobs <- i
	}
	close(jobs)
	wait.Wait()
	return results
}

func (e *Engine) checkPair(ctx context.Context, runtime pairRuntime) PairReport {
	pair := runtime.pair
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	if runtime.ignored.Match(".") {
		report.Skipped = true
		report.Reason = "pair root is ignored or intentionally excepted"
		return report
	}
	left := pair.Peers[runtime.left]
	right := pair.Peers[runtime.right]
	if !e.fs.Available(left.Source) || !e.fs.Available(right.Source) {
		report.Skipped = true
		report.Reason = "optional source unavailable"
		return report
	}
	leftRoot, err := e.fs.Root(left.Source)
	if err != nil {
		return errorReport(pair, err)
	}
	rightRoot, err := e.fs.Root(right.Source)
	if err != nil {
		return errorReport(pair, err)
	}
	if pair.Kind == config.KindFile {
		report.Files = 1
		finding := e.compareFile(ctx, runtime, leftRoot, left.Path, rightRoot, right.Path, ".")
		if finding != nil {
			if finding.State != StateMissingBoth || !pair.Optional {
				report.Findings = append(report.Findings, *finding)
			}
		}
		return report
	}
	if pair.Kind == config.KindSiblings {
		return e.checkSiblings(ctx, runtime, leftRoot, rightRoot)
	}
	return e.checkTree(ctx, runtime, leftRoot, rightRoot)
}

func (e *Engine) checkSiblings(
	ctx context.Context,
	runtime pairRuntime,
	leftRoot, rightRoot *safefs.Root,
) PairReport {
	pair := runtime.pair
	left := pair.Peers[runtime.left]
	right := pair.Peers[runtime.right]
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	leftDirs, err := siblingDirs(leftRoot, left.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if err != nil {
		report.Findings = append(report.Findings, e.finding(runtime, ".", StateError, "", leftRoot, rightRoot, "discover "+runtime.left+" siblings: "+err.Error()))
		return report
	}
	rightDirs, err := siblingDirs(rightRoot, right.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if err != nil {
		report.Findings = append(report.Findings, e.finding(runtime, ".", StateError, "", leftRoot, rightRoot, "discover "+runtime.right+" siblings: "+err.Error()))
		return report
	}
	dirs := make(map[string]struct{}, len(leftDirs)+len(rightDirs))
	for dir := range leftDirs {
		dirs[dir] = struct{}{}
	}
	for dir := range rightDirs {
		dirs[dir] = struct{}{}
	}
	if len(dirs) == 0 {
		if !pair.Optional {
			report.Findings = append(report.Findings, e.finding(runtime, ".", StateMissingBoth, "", leftRoot, rightRoot, "no sibling instruction files found"))
		}
		return report
	}
	directories := mapsKeys(dirs)
	slices.Sort(directories)
	for _, dir := range directories {
		if err := ctx.Err(); err != nil {
			report.Findings = append(report.Findings, e.finding(runtime, dir, StateError, "", leftRoot, rightRoot, err.Error()))
			break
		}
		finding := e.compareFile(
			ctx,
			runtime,
			leftRoot,
			peerPath(pair, left, dir),
			rightRoot,
			peerPath(pair, right, dir),
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
	leftRoot, rightRoot *safefs.Root,
) PairReport {
	pair := runtime.pair
	left := pair.Peers[runtime.left]
	right := pair.Peers[runtime.right]
	report := PairReport{ID: pair.ID, Name: pairName(pair)}
	leftFiles, leftExists, leftIgnored, leftErr := treeFiles(leftRoot, left.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	rightFiles, rightExists, rightIgnored, rightErr := treeFiles(rightRoot, right.Path, e.doc.Config.MaxFiles(), runtime.ignored)
	if leftErr != nil {
		report.Findings = append(report.Findings, e.finding(runtime, ".", StateError, "", leftRoot, rightRoot, leftErr.Error()))
		return report
	}
	if rightErr != nil {
		report.Findings = append(report.Findings, e.finding(runtime, ".", StateError, "", leftRoot, rightRoot, rightErr.Error()))
		return report
	}
	if !leftExists && !rightExists {
		if !pair.Optional {
			report.Findings = append(report.Findings, e.finding(runtime, ".", StateMissingBoth, "", leftRoot, rightRoot, "both trees are missing"))
		}
		return report
	}
	paths := make(map[string]struct{}, len(leftFiles)+len(rightFiles))
	switch pair.Base {
	case runtime.left:
		for rel := range leftFiles {
			paths[rel] = struct{}{}
		}
	case runtime.right:
		for rel := range rightFiles {
			paths[rel] = struct{}{}
		}
	default:
		for rel := range leftFiles {
			paths[rel] = struct{}{}
		}
		for rel := range rightFiles {
			paths[rel] = struct{}{}
		}
	}
	relativePaths := mapsKeys(paths)
	slices.Sort(relativePaths)
	for _, rel := range relativePaths {
		if err := ctx.Err(); err != nil {
			report.Findings = append(report.Findings, e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, err.Error()))
			break
		}
		finding := e.compareFile(
			ctx,
			runtime,
			leftRoot,
			path.Join(left.Path, rel),
			rightRoot,
			path.Join(right.Path, rel),
			rel,
		)
		if finding != nil {
			report.Findings = append(report.Findings, *finding)
		}
		report.Files++
	}
	if len(paths) == 0 && leftExists != rightExists && !leftIgnored && !rightIgnored {
		missing := runtime.right
		if !leftExists {
			missing = runtime.left
		}
		detail := "empty counterpart tree is missing"
		if pair.Base != "" && missing == pair.Base {
			detail = "base tree is missing"
		}
		report.Findings = append(report.Findings, e.finding(runtime, ".", StateMissing, missing, leftRoot, rightRoot, detail))
	}
	return report
}

func (e *Engine) compareFile(
	ctx context.Context,
	runtime pairRuntime,
	leftRoot *safefs.Root,
	leftPath string,
	rightRoot *safefs.Root,
	rightPath, rel string,
) *Finding {
	if err := ctx.Err(); err != nil {
		finding := e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, err.Error())
		return &finding
	}
	leftData, _, leftErr := leftRoot.ReadFile(leftPath, e.doc.Config.MaxFileSize())
	rightData, _, rightErr := rightRoot.ReadFile(rightPath, e.doc.Config.MaxFileSize())
	leftMissing := errors.Is(leftErr, os.ErrNotExist)
	rightMissing := errors.Is(rightErr, os.ErrNotExist)
	switch {
	case leftErr != nil && !leftMissing:
		finding := e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, "read "+runtime.left+" peer: "+leftErr.Error())
		return &finding
	case rightErr != nil && !rightMissing:
		finding := e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, "read "+runtime.right+" peer: "+rightErr.Error())
		return &finding
	case leftMissing && rightMissing:
		finding := e.finding(runtime, rel, StateMissingBoth, "", leftRoot, rightRoot, "both files are missing")
		return &finding
	case leftMissing:
		finding := e.finding(runtime, rel, StateMissing, runtime.left, leftRoot, rightRoot, runtime.left+" counterpart is missing")
		return &finding
	case rightMissing:
		finding := e.finding(runtime, rel, StateMissing, runtime.right, leftRoot, rightRoot, runtime.right+" counterpart is missing")
		return &finding
	}
	leftNormalized, err := normalize(runtime.pair.Normalizer, leftData, Params{Self: runtime.leftSpec, Other: runtime.rightSpec})
	if err != nil {
		finding := e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, "normalize "+runtime.left+" peer: "+err.Error())
		return &finding
	}
	rightNormalized, err := normalize(runtime.pair.Normalizer, rightData, Params{Self: runtime.rightSpec, Other: runtime.leftSpec})
	if err != nil {
		finding := e.finding(runtime, rel, StateError, "", leftRoot, rightRoot, "normalize "+runtime.right+" peer: "+err.Error())
		return &finding
	}
	if sha256.Sum256(leftNormalized) != sha256.Sum256(rightNormalized) {
		finding := e.finding(runtime, rel, StateDifferent, "", leftRoot, rightRoot, "normalized content differs")
		return &finding
	}
	return nil
}

func treeFiles(
	root *safefs.Root,
	base string,
	maxFiles int,
	ignored pattern.Set,
) (map[string]safefs.File, bool, bool, error) {
	ignoredAny := false
	files, err := root.WalkFiles(base, maxFiles, func(rel string) bool {
		matched := ignored.Match(rel)
		ignoredAny = ignoredAny || matched
		return matched
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, false, nil
	}
	if err != nil {
		return nil, true, ignoredAny, err
	}
	result := make(map[string]safefs.File, len(files))
	for _, file := range files {
		result[file.Path] = file
	}
	return result, true, ignoredAny, nil
}

func (e *Engine) finding(
	runtime pairRuntime,
	rel string,
	state State,
	peer string,
	leftRoot, rightRoot *safefs.Root,
	detail string,
) Finding {
	pair := runtime.pair
	return Finding{
		Pair:     pair.ID,
		Relative: rel,
		State:    state,
		Peer:     peer,
		Paths: map[string]string{
			runtime.left:  leftRoot.Abs(peerPath(pair, pair.Peers[runtime.left], rel)),
			runtime.right: rightRoot.Abs(peerPath(pair, pair.Peers[runtime.right], rel)),
		},
		Detail: detail,
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
	if pair.Kind == config.KindSiblings {
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
