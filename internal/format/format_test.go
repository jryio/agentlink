package format

import (
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/agent"
)

func target(t *testing.T, id string) agent.Spec {
	t.Helper()
	spec, ok := agent.Get(id)
	if !ok {
		t.Fatalf("agent %q not registered", id)
	}
	return spec
}

func TestSkillFormat(t *testing.T) {
	t.Parallel()

	canonical := "---\nname: review\ndescription: Review a pull request\nallowed-tools:\n  - Bash\nlicense: MIT\nglobs:\n  - '**/*.go'\n---\n\n# Review\n\nDo it.\n"
	tests := []struct {
		name       string
		target     string
		contains   []string
		omits      []string
		warnSubstr []string
	}{
		{
			"claude keeps allowed-tools and license, drops globs",
			"claude",
			[]string{"name: review", "allowed-tools:", "license: MIT"},
			[]string{"globs"},
			[]string{`drop unsupported frontmatter key globs for claude`},
		},
		{
			"cursor renames globs to paths, drops allowed-tools and license",
			"cursor",
			[]string{"name: review", "paths:"},
			[]string{"allowed-tools", "license", "globs"},
			[]string{`key allowed-tools for cursor`, `key license for cursor`},
		},
		{
			"qodercli emits exactly name and description",
			"qodercli",
			[]string{"name: review", "description: Review a pull request"},
			[]string{"allowed-tools", "license", "globs"},
			nil,
		},
		{
			"omp renames disable-model-invocation and keeps globs",
			"omp",
			[]string{"globs:"},
			[]string{"allowed-tools", "license"},
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			formatter := mustFormatter(t, "skill")
			out, warnings, err := formatter.Format([]byte(canonical), nil, target(t, test.target))
			if err != nil {
				t.Fatalf("Format(): %v", err)
			}
			text := string(out)
			for _, want := range test.contains {
				if !strings.Contains(text, want) {
					t.Errorf("output missing %q:\n%s", want, text)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(text, unwanted) {
					t.Errorf("output contains %q:\n%s", unwanted, text)
				}
			}
			for _, want := range test.warnSubstr {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, want) {
						found = true
					}
				}
				if !found {
					t.Errorf("warnings %v missing %q", warnings, want)
				}
			}
		})
	}
}

func TestSkillCanonicalizePrefersTargetSpelling(t *testing.T) {
	t.Parallel()

	native := "---\nname: review\npaths:\n  - primary/**\nglobs:\n  - legacy/**\n---\nBody\n"
	canonical, err := mustFormatter(t, "skill").Canonicalize(target(t, "cursor"), []byte(native))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	if !strings.Contains(string(canonical), "primary/**") || strings.Contains(string(canonical), "legacy/**") {
		t.Fatalf("Canonicalize() = %q, want Cursor paths to take precedence over legacy globs", canonical)
	}
}

func mustFormatter(t testing.TB, kind string) Formatter {
	t.Helper()
	formatter, ok := For(kind)
	if !ok || formatter == nil {
		t.Fatalf("formatter %q not registered", kind)
	}
	return formatter
}

func TestInstructionsFormat(t *testing.T) {
	t.Parallel()

	formatter := mustFormatter(t, "instructions")
	out, _, err := formatter.Format([]byte("# Agent instructions\n\nBody.\n"), nil, target(t, "claude"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	if !strings.HasPrefix(string(out), "# Claude Code instructions\n") {
		t.Fatalf("claude heading = %q", out)
	}
	out, _, err = formatter.Format([]byte("# Custom title\n\nBody.\n"), nil, target(t, "claude"))
	if err != nil {
		t.Fatalf("Format(custom): %v", err)
	}
	if !strings.HasPrefix(string(out), "# Custom title\n") {
		t.Fatalf("custom heading was rewritten: %q", out)
	}

	// Only the title heading canonicalizes wholesale; later H1s keep their
	// other words and get the prose replacement.
	canonical, err := formatter.Canonicalize(target(t, "claude"), []byte("# Claude Code instructions\n\n# Claude Code configuration\n"))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	if want := "# Agent instructions\n\n# Agent configuration\n"; string(canonical) != want {
		t.Fatalf("Canonicalize() = %q, want %q", canonical, want)
	}
}

func TestInstructionsCanonicalizeMatchesWholeNames(t *testing.T) {
	t.Parallel()

	native := "# Pipeline conventions\n\nKeep Pipeline stable; Pi works.\n"
	canonical, err := mustFormatter(t, "instructions").Canonicalize(target(t, "pi"), []byte(native))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	if want := "# Pipeline conventions\n\nKeep Pipeline stable; Agent works.\n"; string(canonical) != want {
		t.Fatalf("Canonicalize() = %q, want %q", canonical, want)
	}
}

func TestHookFormatCursor(t *testing.T) {
	t.Parallel()

	canonical := `{
  "Notification": [{"hooks": [{"type": "command", "command": "notify"}]}],
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "agentlink remind --agent agent", "timeout": 30}]}],
  "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "echo hi"}]}]
}
`
	formatter := mustFormatter(t, "hook")
	out, warnings, err := formatter.Format([]byte(canonical), nil, target(t, "cursor"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	text := string(out)
	for _, want := range []string{`"version": 1`, `"beforeSubmitPrompt"`, `"preToolUse"`, `"matcher": "Bash"`, `"agentlink remind --agent cursor"`} {
		if !strings.Contains(text, want) {
			t.Errorf("cursor output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Notification") {
		t.Errorf("cursor output kept unsupported Notification event:\n%s", text)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Notification") {
		t.Errorf("warnings = %v, want one Notification drop", warnings)
	}
}

func TestHookFormatKimiTOML(t *testing.T) {
	t.Parallel()

	canonical := `{"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "x", "timeout": 5}]}]}`
	formatter := mustFormatter(t, "hook")
	out, _, err := formatter.Format([]byte(canonical), nil, target(t, "kimi"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	text := string(out)
	for _, want := range []string{"[[hooks]]", `event = "PreToolUse"`, `command = "x"`, "timeout = 5", `matcher = "Bash"`} {
		if !strings.Contains(text, want) {
			t.Errorf("kimi output missing %q:\n%s", want, text)
		}
	}
}

func TestHookFormatMergesSettings(t *testing.T) {
	t.Parallel()

	canonical := `{"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "x"}]}]}`
	existing := `{
  "model": "opus",
  "permissions": {"allow": ["Bash(ls:*)"]},
  "hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "stale"}]}]}
}
`
	formatter := mustFormatter(t, "hook")
	out, _, err := formatter.Format([]byte(canonical), []byte(existing), target(t, "claude"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	text := string(out)
	for _, want := range []string{`"model": "opus"`, `"permissions"`, `"PreToolUse"`} {
		if !strings.Contains(text, want) {
			t.Errorf("merged settings missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "stale") {
		t.Errorf("merged settings kept stale hooks:\n%s", text)
	}
}

func TestHookRoundTripComparesClean(t *testing.T) {
	t.Parallel()

	// A canonical document translated to a target must canonicalize back to
	// the identical bytes under pairwise filtering — this is what makes
	// post-sync check clean, even for lossy targets like gemini.
	canonical := `{"PreToolUse":[{"hooks":[{"command":"agentlink remind --agent agent","timeout":30,"type":"command"}],"matcher":"Bash"}],"UserPromptSubmit":[{"hooks":[{"command":"echo hi","type":"command"}]}]}
`
	formatter := mustFormatter(t, "hook")
	for _, id := range []string{"claude", "codex", "cursor", "copilot", "devin", "droid", "mastracode", "gemini", "kimi", "qodercli", "goose", "hermes"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			spec := target(t, id)
			hub := target(t, "agents")
			out, _, err := formatter.Format([]byte(canonical), nil, spec)
			if err != nil {
				t.Fatalf("Format(): %v", err)
			}
			back, err := CanonicalHookDocument(spec, hub, out)
			if err != nil {
				t.Fatalf("CanonicalHookDocument(): %v\n%s", err, out)
			}
			canonicalAgain, err := CanonicalHookDocument(hub, spec, []byte(canonical))
			if err != nil {
				t.Fatalf("CanonicalHookDocument(agents): %v", err)
			}
			if string(back) != string(canonicalAgain) {
				t.Errorf("round trip mismatch:\ntranslated back: %s\ncanonical:      %s", back, canonicalAgain)
			}
		})
	}
}

func TestHookCanonicalizeSpokeSource(t *testing.T) {
	t.Parallel()

	// A spoke settings document canonicalizes to the bare event map: wrapper
	// keys never leak into canonical form. This is what makes a spoke safe
	// as a sync --from source.
	spoke := `{
  "model": "opus",
  "hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "agentlink remind --agent claude", "timeout": 45}]}]}
}
`
	formatter := mustFormatter(t, "hook")
	canonical, err := formatter.Canonicalize(target(t, "claude"), []byte(spoke))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	text := string(canonical)
	for _, want := range []string{`"PostToolUse"`, `"agentlink remind --agent agent"`, `"timeout": 45`} {
		if !strings.Contains(text, want) {
			t.Errorf("canonical missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "model") || strings.Contains(text, "opus") {
		t.Errorf("canonical leaked wrapper keys:\n%s", text)
	}
}

func TestHookGeminiTimeoutMilliseconds(t *testing.T) {
	t.Parallel()

	formatter := mustFormatter(t, "hook")
	canonical := `{"PreToolUse": [{"hooks": [{"type": "command", "command": "x", "timeout": 30}]}]}`
	out, _, err := formatter.Format([]byte(canonical), nil, target(t, "gemini"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	if !strings.Contains(string(out), `"timeout": 30000`) {
		t.Errorf("gemini output must rescale 30s to 30000ms:\n%s", out)
	}

	spoke := `{"hooks": {"BeforeTool": [{"hooks": [{"type": "command", "command": "x", "timeout": 60000}]}]}}`
	back, err := formatter.Canonicalize(target(t, "gemini"), []byte(spoke))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	if !strings.Contains(string(back), `"timeout":60`) && !strings.Contains(string(back), `"timeout": 60`) {
		t.Errorf("canonical must rescale 60000ms to 60s:\n%s", back)
	}
	if !strings.Contains(string(back), `"PreToolUse"`) {
		t.Errorf("canonical must rename BeforeTool to PreToolUse:\n%s", back)
	}
}

func TestHookCanonicalizeJSONC(t *testing.T) {
	t.Parallel()

	// Qoder's settings.json parser accepts // comments; the hook document
	// must decode them instead of failing the comparison.
	spoke := `{
  // editor preference, not agentlink's concern
  "model": "qwen",
  "hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": "x"}]}]},
}
`
	formatter := mustFormatter(t, "hook")
	canonical, err := formatter.Canonicalize(target(t, "qodercli"), []byte(spoke))
	if err != nil {
		t.Fatalf("Canonicalize(JSONC): %v", err)
	}
	if !strings.Contains(string(canonical), `"PreToolUse"`) {
		t.Errorf("canonical missing PreToolUse:\n%s", canonical)
	}
}

func TestHookCommandAgentTokenBoundaries(t *testing.T) {
	t.Parallel()

	formatter := mustFormatter(t, "hook")
	spoke := `{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"remind-claude --agent claude; remind-claude-staging --agent claude-staging"}]}]}}`
	canonical, err := formatter.Canonicalize(target(t, "claude"), []byte(spoke))
	if err != nil {
		t.Fatalf("Canonicalize(): %v", err)
	}
	canonicalCommand := "remind-agent --agent agent; remind-claude-staging --agent claude-staging"
	if !strings.Contains(string(canonical), canonicalCommand) {
		t.Fatalf("Canonicalize() = %s, want command %q", canonical, canonicalCommand)
	}

	rendered, _, err := formatter.Format(canonical, nil, target(t, "codex"))
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	targetCommand := "remind-codex --agent codex; remind-claude-staging --agent claude-staging"
	if !strings.Contains(string(rendered), targetCommand) {
		t.Fatalf("Format() = %s, want command %q", rendered, targetCommand)
	}
}

func TestHookCanonicalizeMalformedEventErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want string
	}{
		{
			"event entries not array",
			`{"hooks": {"PreToolUse": "oops"}}`,
			"hook event entries must be an array",
		},
		{
			"group handlers not array",
			`{"hooks": {"PreToolUse": [{"hooks": "oops"}]}}`,
			"hook group hooks must be an array",
		},
		{
			"handler not object",
			`{"hooks": {"PreToolUse": [{"hooks": ["oops"]}]}}`,
			"hook handler must be an object",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := mustFormatter(t, "hook").Canonicalize(target(t, "claude"), []byte(test.data))
			if err == nil || !strings.Contains(err.Error(), "PreToolUse") || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Canonicalize() = %v, want an error naming PreToolUse with %q", err, test.want)
			}
		})
	}
}
