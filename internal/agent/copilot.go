package agent

// Evidence: docs/research/copilot.md; https://docs.github.com/en/copilot/how-tos/copilot-cli
// Copilot merges every *.json in .github/hooks/; agentlink owns exactly one
// file there so sibling hook files survive untouched.
func init() {
	Register(Spec{
		ID:                "copilot",
		DocsURL:           "https://docs.github.com/en/copilot/how-tos/copilot-cli",
		DisplayNames:      []string{"GitHub Copilot", "Copilot"},
		ConfigDir:         "",
		GlobalDir:         "~/.copilot",
		Instructions:      []string{".github/copilot-instructions.md"},
		SkillsDir:         ".github/skills",
		NativeAgents:      true,
		SkillKeys:         []string{"name", "description", "argument-hint", "allowed-tools", "user-invocable", "disable-model-invocation", "license"},
		HooksFile:         ".github/hooks/hooks.json",
		HooksFormat:       HookFormatJSON,
		HooksShape:        ShapeFlat,
		HooksWrapper:      WrapperHooks,
		HooksVersion:      1,
		HooksTimeoutField: "timeoutSec",
		HookEventCase:     CaseCamel,
		HookEventMap: map[string]string{
			"UserPromptSubmit": "userPromptSubmitted",
			"Stop":             "agentStop",
		},
		HookEvents: []string{
			"sessionStart", "sessionEnd", "userPromptSubmitted", "preToolUse",
			"postToolUse", "postToolUseFailure", "agentStop", "subagentStart",
			"subagentStop", "preCompact", "notification", "permissionRequest",
		},
		MCPFile:     ".mcp.json",
		MCPFormat:   MCPFormatJSON,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
