package link

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/format"
	"github.com/jryio/agentlink/internal/safefs"
)

// Side identifies the peer explicitly chosen as the source of a sync. It is a
// registered agent ID (see internal/agent); the canonical hub is "agents".
type Side string

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
	// Transform names the formatter kind applied to source bytes before the
	// write ("skill", "instructions", "hook"); empty means raw copy.
	Transform string `json:"transform,omitempty"`
	// Detail carries formatter warnings (dropped events or keys) into the
	// human-reviewable plan.
	Detail string `json:"detail,omitempty"`

	sourceRoot string
	sourcePath string
	targetRoot string
	targetPath string
	data       []byte
	mode       fs.FileMode
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
	if _, ok := agent.Get(string(from)); !ok {
		return Plan{}, fmt.Errorf("sync source must be a registered agent, got %q", string(from))
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
		ids := pair.PeerIDs()
		fromPeer := string(from) == ids[0] || string(from) == ids[1]
		for _, finding := range pairReport.Findings {
			switch {
			case !fromPeer && finding.State != StateMissingBoth:
				finding.Detail = fmt.Sprintf("sync source %q is not a peer of this pair", string(from))
				plan.Unresolved = append(plan.Unresolved, finding)
				continue
			case needsCopy(finding, from) && !copyAllowed(pair):
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
	if err := validateOperationTargets(plan.Operations); err != nil {
		return Plan{}, err
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

func validateOperationTargets(operations []Operation) error {
	targets := make(map[string]Operation, len(operations))
	for _, operation := range operations {
		if previous, exists := targets[operation.Target]; exists {
			return fmt.Errorf(
				"sync pairs %q and %q target the same path %s",
				previous.Pair,
				operation.Pair,
				operation.Target,
			)
		}
		targets[operation.Target] = operation
	}
	return nil
}

func copyAllowed(pair config.Pair) bool {
	if pair.Sync != "" {
		return pair.Sync == config.SyncCopy || pair.Sync == config.SyncTranslate
	}
	return pair.Normalizer == "" || pair.Normalizer == config.NormalizerExact || pair.Normalizer == config.NormalizerText
}

func needsCopy(finding Finding, from Side) bool {
	if finding.State == StateDifferent {
		return true
	}
	return finding.State == StateMissing && finding.Peer != string(from)
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
			if operation.Transform != "" {
				if err := targetRoot.WriteFileAtomic(operation.targetPath, operation.data, operation.mode); err != nil {
					return fmt.Errorf("translate pair %q path %q: %w", operation.Pair, operation.Relative, err)
				}
				continue
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
	ids := pair.PeerIDs()
	otherID := ids[0]
	if otherID == string(from) {
		otherID = ids[1]
	}
	source := pair.Peers[string(from)]
	target := pair.Peers[otherID]
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
	if finding.State == StateMissing && finding.Peer == string(from) {
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
	}
	if finding.State != StateDifferent && (finding.State != StateMissing || finding.Peer != otherID) {
		return Operation{}, false, ""
	}
	if finding.Relative == "." && pair.Kind == config.KindTree {
		operation.Kind = OperationMkdir
		operation.Source = ""
		return operation, true, ""
	}
	if reason := copySafetyReason(sourceRoot, operation.sourcePath, targetRoot, operation.targetPath); reason != "" {
		return Operation{}, false, reason
	}
	operation.Kind = OperationCopy
	if pair.Sync == config.SyncTranslate {
		formatter, ok := format.For(string(pair.Normalizer))
		if !ok {
			return Operation{}, false, "no formatter for kind " + string(pair.Normalizer)
		}
		data, mode, err := sourceRoot.ReadFile(operation.sourcePath, e.doc.Config.MaxFileSize())
		if err != nil {
			return Operation{}, false, "read translate source: " + err.Error()
		}
		// The source may be either peer, not only the canonical hub: a spoke
		// document must be canonicalized before the target formatter runs, or
		// spoke wrappers and renamed fields would be misread as canonical
		// content and clobber the target.
		sourceSpec, ok := agent.Get(string(from))
		if !ok {
			return Operation{}, false, "unknown sync source agent " + string(from)
		}
		canonical, err := formatter.Canonicalize(sourceSpec, data)
		if err != nil {
			return Operation{}, false, "canonicalize " + string(pair.Normalizer) + " source: " + err.Error()
		}
		var existing []byte
		targetMode := mode // new files inherit the source mode
		current, currentMode, readErr := targetRoot.ReadFile(operation.targetPath, e.doc.Config.MaxFileSize())
		switch {
		case readErr == nil:
			existing = current
			targetMode = currentMode // preserve an existing file's mode
		case errors.Is(readErr, os.ErrNotExist):
			// absent target: format into a fresh document
		default:
			return Operation{}, false, "read translate target: " + readErr.Error()
		}
		targetSpec, _ := agent.Get(otherID)
		translated, warnings, err := formatter.Format(canonical, existing, targetSpec)
		if err != nil {
			return Operation{}, false, "translate " + string(pair.Normalizer) + ": " + err.Error()
		}
		operation.Transform = string(pair.Normalizer)
		operation.Detail = strings.Join(warnings, "; ")
		operation.data = translated
		operation.mode = targetMode
	}
	return operation, true, ""
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
