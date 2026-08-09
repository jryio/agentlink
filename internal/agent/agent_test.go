package agent

import (
	"testing"
)

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Register with a duplicate ID did not panic")
		}
	}()
	Register(Spec{ID: "claude", DisplayNames: []string{"Claude"}})
}

func TestRegisterInvalidPanics(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
	}{
		{"empty id", Spec{DisplayNames: []string{"X"}}},
		{"no display names", Spec{ID: "x"}},
		{"bad event case", Spec{ID: "x", DisplayNames: []string{"X"}, HookEventCase: "shouty"}},
		{"hooks file without format", Spec{ID: "x", DisplayNames: []string{"X"}, HooksFile: "hooks.json"}},
		{"mcp file without format", Spec{ID: "x", DisplayNames: []string{"X"}, MCPFile: "mcp.json"}},
		{"mcp file without table key", Spec{ID: "x", DisplayNames: []string{"X"}, MCPFile: "mcp.json", MCPFormat: MCPFormatJSON}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("Register(%+v) did not panic", tc.spec)
				}
			}()
			Register(tc.spec)
		})
	}
}

func TestRegistryInvariants(t *testing.T) {
	all := All()
	if len(all) != 19 {
		t.Fatalf("All() = %d specs, want 19 (18 agents + canonical agents)", len(all))
	}
	if !sortedByID(all) {
		t.Fatal("All() is not sorted by ID")
	}
	for _, spec := range all {
		if spec.ID == "" || len(spec.DisplayNames) == 0 {
			t.Errorf("spec %+v missing ID or DisplayNames", spec)
		}
		if spec.DocsURL == "" {
			t.Errorf("spec %q missing DocsURL", spec.ID)
		}
	}
	if _, ok := Get("agents"); !ok {
		t.Fatal("canonical agents peer is not registered")
	}
}

func sortedByID(specs []Spec) bool {
	for i := 1; i < len(specs); i++ {
		if specs[i-1].ID >= specs[i].ID {
			return false
		}
	}
	return true
}

func TestHookEventResolution(t *testing.T) {
	tests := []struct {
		agent     string
		canonical string
		wantName  string
		wantOK    bool
	}{
		{"claude", "PreToolUse", "PreToolUse", true},
		{"cursor", "UserPromptSubmit", "beforeSubmitPrompt", true},
		{"cursor", "Stop", "stop", true},
		{"cursor", "Notification", "notification", false},
		{"copilot", "Stop", "agentStop", true},
		{"copilot", "PostCompact", "postCompact", false},
		{"gemini", "PreToolUse", "BeforeTool", true},
		{"gemini", "UserPromptSubmit", "UserPromptSubmit", false},
		{"hermes", "PreToolUse", "pre_tool_call", true},
		{"hermes", "SubagentStart", "subagent_start", true},
		{"hermes", "Notification", "notification", false},
		{"devin", "PostCompact", "PostCompaction", true},
		{"mastracode", "SubagentStop", "SubagentEnd", true},
		{"crush", "PreToolUse", "PreToolUse", true},
		{"crush", "Stop", "Stop", false},
		{"agents", "PostToolUseFailure", "PostToolUseFailure", true},
	}
	for _, tc := range tests {
		t.Run(tc.agent+"/"+tc.canonical, func(t *testing.T) {
			spec, ok := Get(tc.agent)
			if !ok {
				t.Fatalf("agent %q not registered", tc.agent)
			}
			name, supported := spec.HookEvent(tc.canonical)
			if name != tc.wantName || supported != tc.wantOK {
				t.Fatalf("HookEvent(%q) = %q, %v; want %q, %v", tc.canonical, name, supported, tc.wantName, tc.wantOK)
			}
			if supported {
				back, ok := spec.HookCanonical(name)
				if !ok || back != tc.canonical {
					t.Fatalf("HookCanonical(%q) = %q, %v; want %q, true", name, back, ok, tc.canonical)
				}
			}
		})
	}
}
