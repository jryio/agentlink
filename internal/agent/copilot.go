package agent

// Evidence: docs/research/copilot.md; https://docs.github.com/en/copilot/how-tos/copilot-cli
// Copilot merges every *.json in .github/hooks/; agentlink owns exactly one
// file there so sibling hook files survive untouched.
func init() {
	Register(Spec{
		ID:           "copilot",
		DocsURL:      "https://docs.github.com/en/copilot/how-tos/copilot-cli",
		DisplayNames: []string{"GitHub Copilot", "Copilot"},
		ConfigDir:    "",
		GlobalDir:    "~/.copilot",
		Instructions: []string{".github/copilot-instructions.md"},
		// Copilot has no single project config dir; these .github footprints
		// are copilot-specific (docs/research/copilot.md).
		DetectFiles: []string{".github/hooks", ".github/copilot", ".github/agents", ".github/skills", ".github/mcp.json"},
		Skills: SkillSpec{
			Dir:          ".github/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "argument-hint", "allowed-tools", "user-invocable", "disable-model-invocation", "license"},
		},
		Hooks: HookSpec{
			File:         ".github/hooks/hooks.json",
			Format:       DialectJSON,
			Shape:        ShapeFlat,
			Wrapper:      WrapperHooks,
			Version:      1,
			TimeoutField: "timeoutSec",
			EventCase:    CaseCamel,
			EventMap: map[string]string{
				"UserPromptSubmit": "userPromptSubmitted",
				"Stop":             "agentStop",
			},
			Events: []string{
				"sessionStart", "sessionEnd", "userPromptSubmitted", "preToolUse",
				"postToolUse", "postToolUseFailure", "agentStop", "subagentStart",
				"subagentStop", "preCompact", "notification", "permissionRequest",
			},
		},
		MCP: MCPSpec{
			File:     ".mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
