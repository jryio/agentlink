package link

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jryio/agentlink/internal/config"
)

// ActivationState classifies a live-link activation problem.
type ActivationState string

const (
	// ActivationExpectedMissing means the durable target does not exist.
	ActivationExpectedMissing ActivationState = "expected_missing"
	// ActivationLiveMissing means the live path does not exist.
	ActivationLiveMissing ActivationState = "live_missing"
	// ActivationNotSymlink means the live path is not a symlink.
	ActivationNotSymlink ActivationState = "not_symlink"
	// ActivationWrongTarget means the live link resolves somewhere else.
	ActivationWrongTarget ActivationState = "wrong_target"
	// ActivationError means the activation could not be inspected safely.
	ActivationError ActivationState = "error"
)

// ActivationReport describes one live symlink check. An empty State is clean.
type ActivationReport struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Expected string          `json:"expected"`
	Live     string          `json:"live"`
	State    ActivationState `json:"state,omitempty"`
	Detail   string          `json:"detail,omitempty"`
	Skipped  bool            `json:"skipped,omitempty"`
}

func (e *Engine) checkActivation(activation config.Activation) ActivationReport {
	report := ActivationReport{ID: activation.ID, Name: activationName(activation)}
	if !e.fs.Available(activation.Expected.Source) || !e.fs.Available(activation.Live.Source) {
		report.Skipped = true
		report.Detail = "optional source unavailable"
		return report
	}
	expectedRoot, err := e.fs.Root(activation.Expected.Source)
	if err != nil {
		return activationErrorReport(activation, err)
	}
	liveRoot, err := e.fs.Root(activation.Live.Source)
	if err != nil {
		return activationErrorReport(activation, err)
	}
	report.Expected = expectedRoot.Abs(activation.Expected.Path)
	report.Live = liveRoot.Abs(activation.Live.Path)
	if _, err := expectedRoot.Stat(activation.Expected.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if activation.Optional {
				report.Skipped = true
				report.Detail = "optional durable target is absent"
				return report
			}
			report.State = ActivationExpectedMissing
		} else {
			report.State = ActivationError
		}
		report.Detail = err.Error()
		return report
	}
	info, err := liveRoot.Lstat(activation.Live.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.State = ActivationLiveMissing
		} else {
			report.State = ActivationError
		}
		report.Detail = err.Error()
		return report
	}
	if info.Mode()&os.ModeSymlink == 0 {
		report.State = ActivationNotSymlink
		report.Detail = "live path must be a symlink"
		return report
	}
	target, err := liveRoot.Readlink(activation.Live.Path)
	if err != nil {
		report.State = ActivationError
		report.Detail = err.Error()
		return report
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(report.Live), target)
	}
	if filepath.Clean(target) != filepath.Clean(report.Expected) {
		report.State = ActivationWrongTarget
		report.Detail = "live symlink resolves to " + filepath.Clean(target)
	}
	return report
}

func activationErrorReport(activation config.Activation, err error) ActivationReport {
	return ActivationReport{
		ID: activation.ID, Name: activationName(activation), State: ActivationError, Detail: err.Error(),
	}
}

func activationName(activation config.Activation) string {
	if activation.Name != "" {
		return activation.Name
	}
	return activation.ID
}
