package hookinput

import (
	"strings"
	"testing"
)

func TestParseJSONPatch(t *testing.T) {
	t.Parallel()

	input := `{"tool_input":{"patch":"*** Update File: CLAUDE.md\n*** Add File: .claude/skills/a/SKILL.md"}}`
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}
	want := []string{".claude/skills/a/SKILL.md", "CLAUDE.md"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Parse() = %v, want %v", got, want)
	}
}

func TestParseLinesDeduplicates(t *testing.T) {
	t.Parallel()

	got, err := Parse(strings.NewReader("AGENTS.md\nCLAUDE.md\nAGENTS.md\n"))
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}
	if want := 2; len(got) != want {
		t.Errorf("len(Parse()) = %d, want %d", len(got), want)
	}
}

func TestParseRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	if _, err := Parse(strings.NewReader(strings.Repeat("x", maxInputSize+1))); err == nil {
		t.Fatal("Parse(oversized) succeeded")
	}
}
