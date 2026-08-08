package app

import (
	"strings"
	"testing"
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
