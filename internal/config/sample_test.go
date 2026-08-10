package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleForDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plant    []string // files or dirs to create beneath the temp dir
		contains []string
		omits    []string
	}{
		{
			"empty dir falls back to claude template",
			nil,
			[]string{"project-instructions", "project-skills", "claude: {source: project, path: CLAUDE.md}"},
			[]string{"activations:"},
		},
		{
			"AGENTS.md alone detects nothing",
			[]string{"AGENTS.md"},
			[]string{"project-instructions"},
			[]string{"cursor", "activations:"},
		},
		{
			"claude dir yields instructions pair, hooks pair, and skills activation",
			[]string{".claude/"},
			[]string{"claude-instructions", "claude-hooks", "claude-skills-live", "live: {source: project, path: .claude/skills}"},
			[]string{"project-instructions"},
		},
		{
			"cursor dir yields hooks pair only",
			[]string{".cursor/"},
			[]string{"cursor-hooks", ".cursor/hooks.json"},
			[]string{"cursor-instructions", "activations:"},
		},
		{
			"gemini dir pairs GEMINI.md",
			[]string{".gemini/"},
			[]string{"gemini-instructions", "gemini: {source: project, path: GEMINI.md}"},
			nil,
		},
		{
			"copilot instructions file uses a file pair",
			[]string{".github/copilot-instructions.md"},
			[]string{"copilot-instructions", "kind: file", ".github/copilot-instructions.md"},
			nil,
		},
		{
			"copilot hooks dir detects without instructions file",
			[]string{".github/hooks/"},
			[]string{"copilot-instructions", "copilot-hooks"},
			nil,
		},
		{
			"copilot settings dir detects without instructions file",
			[]string{".github/copilot/"},
			[]string{"copilot-instructions"},
			nil,
		},
		{
			"crush root file detection",
			[]string{"crush.json"},
			[]string{"crush-hooks", "crush: {source: project, path: crush.json}"},
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, name := range test.plant {
				path := filepath.Join(dir, name)
				if strings.HasSuffix(name, "/") {
					if err := os.MkdirAll(path, 0o750); err != nil {
						t.Fatalf("os.MkdirAll(%q): %v", path, err)
					}
					continue
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
					t.Fatalf("os.MkdirAll(%q): %v", filepath.Dir(path), err)
				}
				writeFile(t, path, []byte("# fixture\n"))
			}
			sample, err := SampleFor(dir)
			if err != nil {
				t.Fatalf("SampleFor(): %v", err)
			}
			text := string(sample)
			for _, want := range test.contains {
				if !strings.Contains(text, want) {
					t.Errorf("SampleFor() missing %q:\n%s", want, text)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(text, unwanted) {
					t.Errorf("SampleFor() contains %q:\n%s", unwanted, text)
				}
			}
			configPath := filepath.Join(dir, "agentlink.yaml")
			writeFile(t, configPath, sample)
			if _, err := Load(configPath, dir); err != nil {
				t.Fatalf("generated config does not load: %v\n%s", err, text)
			}
		})
	}
}

func TestSampleForDetectionErrors(t *testing.T) {
	t.Parallel()

	if _, err := SampleFor(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("SampleFor(missing dir) succeeded, want OpenRoot error")
	}

	t.Run("symlink escaping the root errors instead of detecting outside", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(dir, ".claude")); err != nil {
			t.Skipf("os.Symlink() unavailable: %v", err)
		}
		if _, err := SampleFor(dir); err == nil {
			t.Fatal("SampleFor(escaping .claude symlink) succeeded, want confinement error")
		}
	})
}
