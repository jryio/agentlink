package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIEndToEnd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	writeAppFile(t, filepath.Join(dir, "CLAUDE.md"), "# Project instructions\n\nKeep it simple.\n")

	_, _, err := runCLIErr(t, dir, nil, "check")
	assertExitCode(t, err, exitDrift)
	if _, statErr := os.Stat(filepath.Join(dir, "AGENTS.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("AGENTS.md exists before sync: %v", statErr)
	}

	out, _ := runCLI(t, dir, nil, "sync", "--from", "claude")
	if !strings.Contains(out, "Plan only") {
		t.Fatalf("sync preview output = %q, want plan marker", out)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "AGENTS.md")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("AGENTS.md exists after preview: %v", statErr)
	}

	runCLI(t, dir, nil, "sync", "--from", "claude", "--apply")
	out, _ = runCLI(t, dir, nil, "check")
	if !strings.Contains(out, "clean") {
		t.Fatalf("check output = %q, want clean", out)
	}

	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(docs): %v", err)
	}
	writeAppFile(t, filepath.Join(dir, "docs", "CLAUDE.md"), "# Nested instructions\n")
	_, _, err = runCLIErr(t, dir, nil, "check")
	assertExitCode(t, err, exitDrift)
	runCLI(t, dir, nil, "sync", "--from", "claude", "--apply")
	if _, statErr := os.Stat(filepath.Join(dir, "docs", "AGENTS.md")); statErr != nil {
		t.Fatalf("nested AGENTS.md was not created: %v", statErr)
	}

	writeAppFile(t, filepath.Join(dir, "CLAUDE.md"), "# Project instructions\n\nChanged.\n")
	_, stderr, err := runCLIErr(t, dir, nil, "guard", "CLAUDE.md")
	assertExitCode(t, err, exitDrift)
	if !strings.Contains(stderr, "blocked") || !strings.Contains(stderr, "AGENTS.md") {
		t.Fatalf("guard stderr = %q, want actionable counterpart", stderr)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	_, _, err := runCLIErr(t, dir, nil, "init")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second init error = %v, want exists error", err)
	}
}

func TestInitRejectsCollidingOutputPathsBeforeWriting(t *testing.T) {
	t.Parallel()

	for _, force := range []bool{false, true} {
		name := "without force"
		if force {
			name = "with force"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			args := []string{"init"}
			if force {
				args = append(args, "--force")
			}
			args = append(args, "agentlink.schema.json")
			_, _, err := runCLIErr(t, dir, nil, args...)
			assertExitCode(t, err, exitUsage)
			if _, statErr := os.Stat(filepath.Join(dir, "agentlink.schema.json")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("colliding output exists after rejected init: %v", statErr)
			}
		})
	}
}

func TestRemindCodexJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	writeAppFile(t, filepath.Join(dir, "CLAUDE.md"), "# Project instructions\n")
	out, _ := runCLI(t, dir, strings.NewReader(`{"file_path":"CLAUDE.md"}`), "remind", "--agent", "codex")
	if !strings.Contains(out, `"hookEventName": "PostToolUse"`) || !strings.Contains(out, "AGENTS.md") {
		t.Fatalf("remind output = %q, want Codex hook payload", out)
	}
}

func TestInformationalCommands(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	for _, command := range []string{"validate", "doctor", "list", "help", "version"} {
		out, _ := runCLI(t, dir, nil, command)
		if strings.TrimSpace(out) == "" {
			t.Errorf("Run(%s) produced no output", command)
		}
	}
	flagHelp, _ := runCLI(t, dir, nil, "check", "--help")
	if !strings.Contains(flagHelp, "agentlink help") {
		t.Fatalf("check --help output = %q", flagHelp)
	}
	runCLI(t, dir, nil, "guard")

	schema, _ := runCLI(t, dir, nil, "schema")
	if !json.Valid([]byte(schema)) {
		t.Fatal("schema command did not emit valid JSON")
	}
	listJSON, _ := runCLI(t, dir, nil, "--json", "list")
	if !json.Valid([]byte(listJSON)) || !strings.Contains(listJSON, `"project-instructions"`) {
		t.Fatalf("JSON list output = %q", listJSON)
	}
	checkJSON, _ := runCLI(t, dir, nil, "--json", "check")
	if !json.Valid([]byte(checkJSON)) {
		t.Fatalf("JSON check output = %q", checkJSON)
	}
	doctorJSON, _ := runCLI(t, dir, nil, "--json", "doctor")
	if !json.Valid([]byte(doctorJSON)) {
		t.Fatalf("JSON doctor output = %q", doctorJSON)
	}
	planJSON, _ := runCLI(t, dir, nil, "--json", "sync", "--from", "claude")
	if !json.Valid([]byte(planJSON)) {
		t.Fatalf("JSON sync output = %q", planJSON)
	}
	validateJSON, _ := runCLI(t, dir, nil, "--json", "validate")
	if !json.Valid([]byte(validateJSON)) || !strings.Contains(validateJSON, `"valid": true`) {
		t.Fatalf("JSON validate output = %q", validateJSON)
	}
	guardJSON, _ := runCLI(t, dir, nil, "--json", "guard")
	if !json.Valid([]byte(guardJSON)) || !strings.Contains(guardJSON, `"violations": []`) {
		t.Fatalf("JSON guard output = %q", guardJSON)
	}
	initJSON, _ := runCLI(t, dir, nil, "--json", "init", "--force")
	if !json.Valid([]byte(initJSON)) || !strings.Contains(initJSON, `"schema"`) {
		t.Fatalf("JSON init output = %q", initJSON)
	}
	quietList, _ := runCLI(t, dir, nil, "--quiet", "list")
	if quietList != "" {
		t.Fatalf("quiet list output = %q, want empty", quietList)
	}

	runCLI(t, dir, nil, "init", "--force")
	_, _, err := runCLIErr(t, dir, nil, "unknown-command")
	assertExitCode(t, err, exitUsage)
	_, _, err = runCLIErr(t, dir, nil, "--format", "xml", "check")
	assertExitCode(t, err, exitUsage)
}

func TestExitError(t *testing.T) {
	t.Parallel()

	cause := errors.New("cause")
	err := &ExitError{Code: 7, Err: cause}
	if got := err.Error(); got != "cause" {
		t.Errorf("ExitError.Error() = %q, want cause", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("ExitError did not unwrap cause")
	}
	if got := (&ExitError{Code: 9}).Error(); got != "exit status 9" {
		t.Errorf("silent ExitError.Error() = %q", got)
	}
}

func TestCommandUsageErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runCLI(t, dir, nil, "init")
	tests := [][]string{
		{"check", "extra"},
		{"check", "--unknown"},
		{"check", "--pair", "missing"},
		{"sync"},
		{"sync", "--from", "claude", "extra"},
		{"guard", "--agent", "robot", "CLAUDE.md"},
		{"list", "extra"},
		{"validate", "extra"},
		{"doctor", "extra"},
		{"schema", "extra"},
		{"init", "one", "two"},
		{"--config"},
		{"--format"},
	}
	for _, args := range tests {
		_, _, err := runCLIErr(t, dir, nil, args...)
		assertExitCode(t, err, exitUsage)
	}

	if err := Run(t.Context(), nil, "test", Streams{}); err == nil {
		t.Fatal("Run() with nil streams succeeded")
	}
}

func runCLI(t testing.TB, cwd string, input *strings.Reader, args ...string) (string, string) {
	t.Helper()
	out, stderr, err := runCLIErr(t, cwd, input, args...)
	if err != nil {
		t.Fatalf("Run(%v): %v\nstdout: %s\nstderr: %s", args, err, out, stderr)
	}
	return out, stderr
}

func runCLIErr(t testing.TB, cwd string, input *strings.Reader, args ...string) (string, string, error) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	reader := strings.NewReader("")
	if input != nil {
		reader = input
	}
	err := Run(t.Context(), args, "test", Streams{In: reader, Out: &stdout, Err: &stderr, CWD: cwd})
	return stdout.String(), stderr.String(), err
}

func assertExitCode(t testing.TB, err error, want int) {
	t.Helper()
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != want {
		t.Fatalf("error = %v, want ExitError code %d", err, want)
	}
}

func writeAppFile(t testing.TB, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("os.MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q): %v", path, err)
	}
}
