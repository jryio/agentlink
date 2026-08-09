package hookinput

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseJSONPatch(t *testing.T) {
	t.Parallel()

	input := `{"tool_input":{"patch":"*** Update File: CLAUDE.md\n*** Add File: .claude/skills/a/SKILL.md"}}`
	got, err := Parse(t.Context(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}
	want := []string{".claude/skills/a/SKILL.md", "CLAUDE.md"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Parse() = %v, want %v", got, want)
	}
}

func TestParseAgentEnvelopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"claude post-tool-use",
			`{"session_id": "s", "hook_event_name": "PostToolUse", "tool_input": {"file_path": "CLAUDE.md"}}`,
			[]string{"CLAUDE.md"},
		},
		{
			"cursor after-file-edit",
			`{"hook_event_name": "afterFileEdit", "file_path": "src/a.go", "edits": [{"old_string": "x"}]}`,
			[]string{"src/a.go"},
		},
		{
			"cursor attachments nested",
			`{"hook_event_name": "beforeSubmitPrompt", "attachments": [{"type": "file", "file_path": "docs/b.md"}]}`,
			[]string{"docs/b.md"},
		},
		{
			"bare file field",
			`{"file": ".agents/hooks.json"}`,
			[]string{".agents/hooks.json"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(t.Context(), strings.NewReader(test.input))
			if err != nil {
				t.Fatalf("Parse(): %v", err)
			}
			if strings.Join(got, "|") != strings.Join(test.want, "|") {
				t.Errorf("Parse() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestParseLinesDeduplicates(t *testing.T) {
	t.Parallel()

	got, err := Parse(t.Context(), strings.NewReader("AGENTS.md\nCLAUDE.md\nAGENTS.md\n"))
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}
	if want := 2; len(got) != want {
		t.Errorf("len(Parse()) = %d, want %d", len(got), want)
	}
}

func TestParseRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	if _, err := Parse(t.Context(), strings.NewReader(strings.Repeat("x", maxInputSize+1))); err == nil {
		t.Fatal("Parse(oversized) succeeded")
	}
}

func TestParseCancellationClosesBlockingInput(t *testing.T) {
	t.Parallel()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	var wait sync.WaitGroup
	t.Cleanup(wait.Wait)
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	wait.Go(func() {
		_, parseErr := Parse(ctx, reader)
		result <- parseErr
	})
	cancel()
	timer := time.NewTimer(5 * time.Second)
	t.Cleanup(func() { timer.Stop() })
	select {
	case parseErr := <-result:
		if !errors.Is(parseErr, context.Canceled) {
			t.Fatalf("Parse(canceled) error = %v, want context.Canceled", parseErr)
		}
		wait.Wait()
	case <-timer.C:
		t.Fatal("Parse(canceled) remained blocked")
	}
}
