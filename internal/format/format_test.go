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
	for _, id := range []string{"claude", "codex", "cursor", "copilot", "devin", "droid", "mastracode", "gemini", "kimi", "qodercli", "goose"} {
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
