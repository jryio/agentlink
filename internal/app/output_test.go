package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/link"
)

func TestEscapeTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain stays unchanged", "claude/skills/SKILL.md", "claude/skills/SKILL.md"},
		{"ESC is escaped", "a\u001b[31mb", `a\x1b[31mb`},
		{"C0 control byte escaped", "a\x00b", `a\x00b`},
		{"OSC terminator in C1 range escaped", "a\x9db", `a\x9db`},
		{"DEL escaped", "a\x7fb", `a\x7fb`},
		{"newline escaped", "line\nbreak", `line\x0abreak`},
		{"unicode printable preserved", "claude/技能/SKILL.md", "claude/技能/SKILL.md"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := escapeTerminal(test.input); got != test.want {
				t.Errorf("escapeTerminal(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestEscapeTerminalLeavesNoControlBytes(t *testing.T) {
	t.Parallel()

	// Every byte the escaper treats as terminal control must never survive:
	// C0 (0x00-0x1F), DEL (0x7F), and C1 (0x80-0x9F).
	check := func(r byte) {
		t.Helper()
		input := "p" + string(r) + "q"
		got := escapeTerminal(input)
		if strings.Contains(got, string(r)) {
			t.Fatalf("escapeTerminal(%q) = %q, control byte %#x still present", input, got, r)
		}
	}
	for r := byte(0); r < 0x20; r++ {
		check(r)
	}
	check(0x7f)
	for r := byte(0x80); r <= 0x9f; r++ {
		check(r)
	}
}

// Control bytes in every field a repository or configuration author could
// control: filenames, paths, and diagnostic text.
const (
	escapeBody = "a\u001b[31mESC"
	nulBody    = "b\x00nul"
	oscBody    = "c\x9dosc"
	delBody    = "d\x7fd"
)

// sinkApp returns an application writing human output to buffered stdout and
// stderr for exercising printReport/printPlan/printGuard.
func sinkApp() (*application, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	return &application{
		streams: Streams{In: strings.NewReader(""), Out: out, Err: err, CWD: "."},
		global:  globalOptions{},
	}, out, err
}

// assertNoHostileControl fails if any of the hostile control bytes injected by
// the fixture (NUL, ESC, DEL, and the bare C1 OSC byte) survive raw in output.
// Ordinary formatter whitespace such as '\n' is not attacker-controlled and is
// not treated as a leak.
func assertNoHostileControl(t *testing.T, got string) {
	t.Helper()
	for _, raw := range []string{"\x00", "\x1b", "\x9d", "\x7f"} {
		if strings.Contains(got, raw) {
			t.Fatalf("output leaked raw control byte %q:\n%q", raw, got)
		}
	}
}

func TestPrintReportEscapesControls(t *testing.T) {
	t.Parallel()

	app, out, _ := sinkApp()
	report := link.Report{
		Pairs: []link.PairReport{
			{ID: "tree", Findings: []link.Finding{{
				Pair: "tree", Relative: string(escapeBody), State: link.StateDifferent,
				Claude: string(nulBody), Codex: string(oscBody), Detail: string(delBody),
			}}},
			{ID: "skipped", Skipped: true, Reason: string(escapeBody)},
		},
		Activations: []link.ActivationReport{{
			ID: "act", State: "drifted", Expected: string(nulBody), Live: string(oscBody), Detail: string(delBody),
		}},
	}
	if err := app.printReport(report); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	got := out.String()
	assertNoHostileControl(t, got)
	for _, escape := range []string{`\x1b`, `\x00`, `\x9d`, `\x7f`} {
		if !strings.Contains(got, escape) {
			t.Errorf("printReport output missing escaped %q:\n%s", escape, got)
		}
	}
}

func TestPrintPlanEscapesControls(t *testing.T) {
	t.Parallel()

	app, out, _ := sinkApp()
	plan := link.Plan{
		From: link.SideClaude,
		Operations: []link.Operation{
			{Kind: link.OperationCopy, Pair: "p", Relative: string(escapeBody), Source: string(nulBody), Target: string(oscBody)},
			{Kind: link.OperationMkdir, Target: string(delBody)},
		},
		Unresolved: []link.Finding{{Pair: "p", Relative: string(escapeBody), Detail: string(oscBody)}},
	}
	if err := app.printPlan(plan, false); err != nil {
		t.Fatalf("printPlan: %v", err)
	}
	got := out.String()
	assertNoHostileControl(t, got)
	for _, escape := range []string{`\x1b`, `\x00`, `\x9d`, `\x7f`} {
		if !strings.Contains(got, escape) {
			t.Errorf("printPlan output missing escaped %q:\n%s", escape, got)
		}
	}
}

func TestPrintGuardEscapesControls(t *testing.T) {
	t.Parallel()

	violations := []link.Violation{{
		Pair: "v", Relative: string(escapeBody), State: link.StateDifferent,
		Message: string(nulBody), Counterpart: string(oscBody),
	}}

	app, out, errBuf := sinkApp()
	if err := app.printGuard(violations, "human", false); err != nil {
		t.Fatalf("printGuard(human, block): %v", err)
	}
	assertNoHostileControl(t, errBuf.String())
	if !strings.Contains(errBuf.String(), `\x1b`) {
		t.Fatalf("printGuard(human, block) missing escaped ESC:\n%s", errBuf.String())
	}

	out.Reset()
	if err := app.printGuard(violations, "human", true); err != nil {
		t.Fatalf("printGuard(human, remind): %v", err)
	}
	assertNoHostileControl(t, out.String())
	if !strings.Contains(out.String(), `\x1b`) {
		t.Fatalf("printGuard(human, remind) missing escaped ESC:\n%s", out.String())
	}
}

func TestPrintGuardCodexJSONEscapesControls(t *testing.T) {
	t.Parallel()

	app, out, _ := sinkApp()
	violations := []link.Violation{{
		Pair: "v", Relative: string(escapeBody), State: link.StateDifferent,
		Message: string(nulBody), Counterpart: string(oscBody),
	}}
	if err := app.printGuard(violations, "codex", true); err != nil {
		t.Fatalf("printGuard(codex, remind): %v", err)
	}
	got := out.String()
	assertNoHostileControl(t, got)
	if !json.Valid([]byte(got)) {
		t.Fatalf("printGuard(codex) output is not valid JSON:\n%s", got)
	}
	var envelope map[string]map[string]string
	if err := json.Unmarshal([]byte(got), &envelope); err != nil {
		t.Fatalf("printGuard(codex) JSON unmarshal: %v", err)
	}
	context := envelope["hookSpecificOutput"]["additionalContext"]
	if !strings.Contains(context, `\x1b`) {
		t.Errorf("codex additionalContext missing escaped ESC, got %q", context)
	}
	assertNoHostileControl(t, context)
}
