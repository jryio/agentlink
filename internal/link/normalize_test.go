package link

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
)

func testParams(t *testing.T, self, other string) Params {
	t.Helper()
	selfSpec, ok := agent.Get(self)
	if !ok {
		t.Fatalf("agent %q not registered", self)
	}
	otherSpec, ok := agent.Get(other)
	if !ok {
		t.Fatalf("agent %q not registered", other)
	}
	return Params{Self: selfSpec, Other: otherSpec}
}

func TestNormalizePresets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		preset      config.Normalizer
		left        string
		leftParams  Params
		right       string
		rightParams Params
	}{
		{"exact", "exact", "same\r\n", Params{}, "same\r\n", Params{}},
		{"text line endings", "text", "same  \r\n", Params{}, "same\n", Params{}},
		{
			"instruction titles", "instructions",
			"# Claude Code rules\nUse Claude Code and Codex.\n", testParams(t, "claude", "codex"),
			"# Codex rules\nUse Claude and Codex.\n", testParams(t, "codex", "claude"),
		},
		{
			"instruction names overlap", "instructions",
			"# Agent instructions\nUse Oh My Pi.\n", testParams(t, "pi", "omp"),
			"# Agent instructions\nUse Oh My Pi.\n", testParams(t, "omp", "pi"),
		},
		{
			"skill drops tool-only keys", "skill",
			"---\nname: x\nwhen_to_use: Review code.\n---\nBody\n", testParams(t, "claude", "codex"),
			"---\nname: x\n---\nBody\n", testParams(t, "codex", "claude"),
		},
		{
			"skill keeps mutually supported keys", "skill",
			"---\nname: x\nallowed-tools:\n  - Bash\n---\nBody\n", testParams(t, "claude", "codex"),
			"---\nallowed-tools:\n  - Bash\nname: x\n---\nBody\n", testParams(t, "codex", "claude"),
		},
		{
			"skill drops keys cursor lacks", "skill",
			"---\nname: x\nallowed-tools:\n  - Bash\nlicense: MIT\n---\nBody\n", testParams(t, "claude", "cursor"),
			"---\nname: x\n---\nBody\n", testParams(t, "cursor", "claude"),
		},
		{
			"skill maps omp camelCase keys", "skill",
			"---\nname: x\ndisableModelInvocation: true\n---\nBody\n", testParams(t, "omp", "claude"),
			"---\ndisable-model-invocation: true\nname: x\n---\nBody\n", testParams(t, "claude", "omp"),
		},
		{
			"hook claude settings vs codex wrapper", "hook",
			`{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "agentlink remind --agent claude"}]}]}}` + "\n",
			testParams(t, "claude", "codex"),
			`{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "agentlink remind --agent codex"}]}]}}` + "\n",
			testParams(t, "codex", "claude"),
		},
		{
			"hook copilot camelCase event", "hook",
			`{"version": 1, "hooks": {"userPromptSubmitted": [{"type": "command", "command": "agentlink remind --agent copilot"}]}}` + "\n",
			testParams(t, "copilot", "agents"),
			`{"UserPromptSubmit": [{"hooks": [{"type": "command", "command": "agentlink remind --agent agent"}]}]}` + "\n",
			testParams(t, "agents", "copilot"),
		},
		{
			"hook cursor prompt rename", "hook",
			`{"version": 1, "hooks": {"beforeSubmitPrompt": [{"type": "command", "command": "agentlink remind --agent cursor"}]}}` + "\n",
			testParams(t, "cursor", "agents"),
			`{"UserPromptSubmit": [{"hooks": [{"type": "command", "command": "agentlink remind --agent agent"}]}]}` + "\n",
			testParams(t, "agents", "cursor"),
		},
		{
			"hook kimi TOML list", "hook",
			"[[hooks]]\nevent = \"PreToolUse\"\ncommand = \"agentlink remind --agent kimi\"\n",
			testParams(t, "kimi", "agents"),
			`{"PreToolUse": [{"hooks": [{"command": "agentlink remind --agent agent"}]}]}` + "\n",
			testParams(t, "agents", "kimi"),
		},
		{
			"hook mastracode milliseconds and matcher object", "hook",
			`{"PreToolUse": [{"type": "command", "command": "x", "timeout": 5000, "matcher": {"tool_name": "Bash"}}]}` + "\n",
			testParams(t, "mastracode", "agents"),
			`{"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "x", "timeout": 5}]}]}` + "\n",
			testParams(t, "agents", "mastracode"),
		},
		{"presence", "presence", "anything", Params{}, "different", Params{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			left, err := normalize(test.preset, []byte(test.left), test.leftParams)
			if err != nil {
				t.Fatalf("normalize(%q, left): %v", test.preset, err)
			}
			right, err := normalize(test.preset, []byte(test.right), test.rightParams)
			if err != nil {
				t.Fatalf("normalize(%q, right): %v", test.preset, err)
			}
			if !bytes.Equal(left, right) {
				t.Errorf("normalized values differ:\nleft  %s\nright %s", left, right)
			}
		})
	}
}

func TestNormalizeFailures(t *testing.T) {
	t.Parallel()

	if _, err := normalize("text", []byte{'a', 0, 'b'}, Params{}); err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("normalize(binary text) error = %v", err)
	}
	if _, err := normalize("unknown", []byte("text"), Params{}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("normalize(unknown) error = %v", err)
	}
	if _, err := normalize("skill", []byte("---\ninvalid: [\n---\nbody"), Params{}); err == nil || !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("normalize(invalid skill) error = %v", err)
	}
	if _, err := normalize("hook", []byte("not json"), testParams(t, "claude", "codex")); err == nil {
		t.Fatalf("normalize(invalid hook JSON) succeeded")
	}
}
