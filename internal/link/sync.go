package link

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

// Side identifies the peer explicitly chosen as the source of a sync.
type Side string

const (
	// SideClaude selects Claude artifacts as the sync source.
	SideClaude Side = "claude"
	// SideCodex selects Codex artifacts as the sync source.
	SideCodex Side = "codex"
)

// OperationKind is a planned filesystem mutation.
type OperationKind string

const (
	// OperationCopy atomically writes a source file to its peer.
	OperationCopy OperationKind = "copy"
	// OperationDelete removes a target-only regular file during pruning.
	OperationDelete OperationKind = "delete"
	// OperationMkdir creates an empty counterpart tree.
	OperationMkdir OperationKind = "mkdir"
)

// Operation is a human-reviewable sync action.
type Operation struct {
	Kind     OperationKind `json:"kind"`
	Pair     string        `json:"pair"`
	Relative string        `json:"relative"`
	Source   string        `json:"source,omitempty"`
	Target   string        `json:"target"`

	sourceRoot string
	sourcePath string
	targetRoot string
	targetPath string
}

// Plan describes mutations and findings that require manual resolution.
type Plan struct {
	From       Side        `json:"from"`
	Operations []Operation `json:"operations"`
	Unresolved []Finding   `json:"unresolved,omitempty"`
}

// PlanSync creates a safe, deterministic reconciliation plan. Deletions only
// appear when prune is true.
func (e *Engine) PlanSync(ctx context.Context, from Side, prune bool, selected map[string]bool) (Plan, error) {
	if from != SideClaude && from != SideCodex {
		return Plan{}, fmt.Errorf("sync source must be claude or codex")
	}
	plan := Plan{From: from, Operations: make([]Operation, 0)}
	artifactSelection := make(map[string]bool, len(e.doc.Config.Pairs))
	for _, pair := range e.doc.Config.Pairs {
		if len(selected) == 0 || selected[pair.ID] {
			artifactSelection[pair.ID] = true
		}
	}
	if len(artifactSelection) == 0 {
		return plan, nil
	}
	report := e.Check(ctx, artifactSelection)
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	for _, pairReport := range report.Pairs {
		pair, ok := e.doc.Config.PairByID(pairReport.ID)
		if !ok {
			return Plan{}, fmt.Errorf("pair %q disappeared from configuration", pairReport.ID)
		}
		for _, finding := range pairReport.Findings {
			if needsCopy(finding.State, from) && !copyAllowed(pair) {
				finding.Detail = "raw copy disabled for semantic pair; review the adaptation or set sync: copy"
				plan.Unresolved = append(plan.Unresolved, finding)
				continue
			}
			operation, actionable, reason := e.operationFor(pair, finding, from, prune)
			if actionable {
				plan.Operations = append(plan.Operations, operation)
			} else {
				if reason != "" {
					finding.Detail = reason
				}
				plan.Unresolved = append(plan.Unresolved, finding)
			}
		}
	}
	slices.SortFunc(plan.Operations, func(a, b Operation) int {
		if a.Pair != b.Pair {
			return cmpString(a.Pair, b.Pair)
		}
		if a.Relative != b.Relative {
			return cmpString(a.Relative, b.Relative)
		}
		return cmpString(string(a.Kind), string(b.Kind))
	})
	return plan, nil
}

func copyAllowed(pair config.Pair) bool {
	if pair.Sync != "" {
		return pair.Sync == "copy"
	}
	return pair.Normalizer == "" || pair.Normalizer == "exact" || pair.Normalizer == "text"
}

func needsCopy(state State, from Side) bool {
	if state == StateDifferent {
		return true
	}
	return (from == SideClaude && state == StateMissingCodex) || (from == SideCodex && state == StateMissingClaude)
}

// Apply executes a plan through confined roots. Plans are intentionally
// process-local: callers should create and apply them without serialization.
func (e *Engine) Apply(ctx context.Context, plan Plan) error {
	for _, operation := range plan.Operations {
		if err := ctx.Err(); err != nil {
			return err
		}
		targetRoot, err := e.fs.Root(operation.targetRoot)
		if err != nil {
			return fmt.Errorf("%s %s: %w", operation.Kind, operation.Target, err)
		}
		switch operation.Kind {
		case OperationCopy:
			sourceRoot, err := e.fs.Root(operation.sourceRoot)
			if err != nil {
				return fmt.Errorf("copy %s: %w", operation.Source, err)
			}
			if err := safefs.CopyFile(
				sourceRoot,
				operation.sourcePath,
				targetRoot,
				operation.targetPath,
				e.doc.Config.MaxFileSize(),
			); err != nil {
				return fmt.Errorf("copy pair %q path %q: %w", operation.Pair, operation.Relative, err)
			}
		case OperationDelete:
			if err := targetRoot.RemoveFile(operation.targetPath); err != nil {
				return fmt.Errorf("delete pair %q path %q: %w", operation.Pair, operation.Relative, err)
			}
		case OperationMkdir:
			if err := targetRoot.MkdirAll(operation.targetPath, 0o750); err != nil {
				return fmt.Errorf("create pair %q tree: %w", operation.Pair, err)
			}
		default:
			return fmt.Errorf("unknown operation %q", operation.Kind)
		}
	}
	return nil
}

func (e *Engine) operationFor(pair config.Pair, finding Finding, from Side, prune bool) (Operation, bool, string) {
	source := pair.Claude
	target := pair.Codex
	copyState := StateMissingCodex
	deleteState := StateMissingClaude
	if from == SideCodex {
		source, target = target, source
		copyState, deleteState = deleteState, copyState
	}
	operation := Operation{
		Pair:       pair.ID,
		Relative:   finding.Relative,
		sourceRoot: source.Source,
		sourcePath: peerPath(pair, source, finding.Relative),
		targetRoot: target.Source,
		targetPath: peerPath(pair, target, finding.Relative),
	}
	sourceRoot, sourceErr := e.fs.Root(source.Source)
	targetRoot, targetErr := e.fs.Root(target.Source)
	if sourceErr != nil {
		return Operation{}, false, "sync source is unavailable: " + sourceErr.Error()
	}
	if targetErr != nil {
		return Operation{}, false, "sync target is unavailable: " + targetErr.Error()
	}
	operation.Source = sourceRoot.Abs(operation.sourcePath)
	operation.Target = targetRoot.Abs(operation.targetPath)
	switch finding.State {
	case StateDifferent, copyState:
		if finding.Relative == "." && pair.Kind == "tree" {
			operation.Kind = OperationMkdir
			operation.Source = ""
			return operation, true, ""
		}
		if reason := copySafetyReason(sourceRoot, operation.sourcePath, targetRoot, operation.targetPath); reason != "" {
			return Operation{}, false, reason
		}
		operation.Kind = OperationCopy
		return operation, true, ""
	case deleteState:
		if !prune || finding.Relative == "." {
			return Operation{}, false, ""
		}
		info, err := targetRoot.Lstat(operation.targetPath)
		if err != nil {
			return Operation{}, false, "inspect prune target: " + err.Error()
		}
		if !info.Mode().IsRegular() {
			return Operation{}, false, "refuse to prune non-regular target " + operation.Target
		}
		operation.Kind = OperationDelete
		operation.Source = ""
		return operation, true, ""
	default:
		return Operation{}, false, ""
	}
}

func copySafetyReason(sourceRoot *safefs.Root, sourcePath string, targetRoot *safefs.Root, targetPath string) string {
	sourceInfo, err := sourceRoot.Lstat(sourcePath)
	if err != nil {
		return "inspect sync source: " + err.Error()
	}
	if !sourceInfo.Mode().IsRegular() {
		return "refuse to copy non-regular source " + sourceRoot.Abs(sourcePath) + "; use an activation for symlinks"
	}
	targetInfo, err := targetRoot.Lstat(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		return "inspect sync target: " + err.Error()
	}
	if !targetInfo.Mode().IsRegular() {
		return "refuse to replace non-regular target " + targetRoot.Abs(targetPath)
	}
	return ""
}

func cmpString(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
