package link

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func TestActivationDetectsWrongLiveTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	expected := filepath.Join(dir, "tracked", "codex", "skills")
	if err := os.MkdirAll(expected, 0o750); err != nil {
		t.Fatalf("os.MkdirAll(expected): %v", err)
	}
	liveDir := filepath.Join(dir, ".codex")
	if err := os.MkdirAll(liveDir, 0o750); err != nil {
		t.Fatalf("os.MkdirAll(live): %v", err)
	}
	live := filepath.Join(liveDir, "skills")
	if err := os.Symlink(filepath.Join("..", "tracked", "codex", "skills"), live); err != nil {
		t.Skipf("os.Symlink() unavailable: %v", err)
	}
	doc := testDocument(dir)
	doc.Config.Activations = []config.Activation{{
		ID:       "codex-skills-live",
		Expected: config.Endpoint{Source: "workspace", Path: "tracked/codex/skills"},
		Live:     config.Endpoint{Source: "workspace", Path: ".codex/skills"},
	}}
	roots, err := safefs.Open(doc)
	if err != nil {
		t.Fatalf("safefs.Open(): %v", err)
	}
	t.Cleanup(func() {
		if err := roots.Close(); err != nil {
			t.Errorf("roots.Close(): %v", err)
		}
	})
	engine, err := New(doc, roots)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	report := engine.Check(t.Context(), map[string]bool{"codex-skills-live": true})
	if !report.Clean() {
		t.Fatalf("Check(clean activation) = %+v", report)
	}

	if err := os.Remove(live); err != nil {
		t.Fatalf("os.Remove(live): %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "somewhere-else"), live); err != nil {
		t.Fatalf("os.Symlink(wrong): %v", err)
	}
	report = engine.Check(t.Context(), map[string]bool{"codex-skills-live": true})
	if report.Clean() || report.Activations[0].State != ActivationWrongTarget {
		t.Fatalf("Check(wrong activation) = %+v, want wrong target", report)
	}
}

func TestActivationStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		optional bool
		setup    func(testing.TB, string)
		state    ActivationState
		skipped  bool
	}{
		{"expected missing", false, func(testing.TB, string) {}, ActivationExpectedMissing, false},
		{"optional expected missing", true, func(testing.TB, string) {}, "", true},
		{"live missing", false, func(t testing.TB, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, "expected"), 0o750); err != nil {
				t.Fatalf("os.MkdirAll(expected): %v", err)
			}
		}, ActivationLiveMissing, false},
		{"live regular file", false, func(t testing.TB, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, "expected"), 0o750); err != nil {
				t.Fatalf("os.MkdirAll(expected): %v", err)
			}
			writeTestFile(t, filepath.Join(dir, "live"), "not a symlink")
		}, ActivationNotSymlink, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			test.setup(t, dir)
			doc := testDocument(dir)
			doc.Config.Activations = []config.Activation{{
				ID: "live-check", Expected: config.Endpoint{Source: "workspace", Path: "expected"},
				Live: config.Endpoint{Source: "workspace", Path: "live"}, Optional: test.optional,
			}}
			roots, err := safefs.Open(doc)
			if err != nil {
				t.Fatalf("safefs.Open(): %v", err)
			}
			t.Cleanup(func() {
				if err := roots.Close(); err != nil {
					t.Errorf("roots.Close(): %v", err)
				}
			})
			engine, err := New(doc, roots)
			if err != nil {
				t.Fatalf("New(): %v", err)
			}
			report := engine.Check(t.Context(), map[string]bool{"live-check": true})
			if got := report.Activations[0]; got.State != test.state || got.Skipped != test.skipped {
				t.Errorf("ActivationReport = %+v, want state %q skipped %v", got, test.state, test.skipped)
			}
		})
	}
}
