package agent

// Evidence: docs/research/cursor.md; https://cursor.com/docs/cli
// Cursor uses lowerCamelCase hook events, spells the prompt-submit event
// beforeSubmitPrompt, and prefers `paths` over the legacy `globs` skill key.
func init() {
	Register(Spec{
		ID:            "cursor",
		DocsURL:       "https://cursor.com/docs/cli",
		DisplayNames:  []string{"Cursor"},
		ConfigDir:     ".cursor",
		GlobalDir:     "~/.cursor",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".cursor/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "paths", "globs", "disable-model-invocation", "metadata", "user-invocable"},
		SkillRenames:  map[string]string{"globs": "paths"},
		HooksFile:     ".cursor/hooks.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeFlat,
		HooksWrapper:  WrapperHooks,
		HooksVersion:  1,
		HookEventCase: CaseCamel,
		HookEventMap:  map[string]string{"UserPromptSubmit": "beforeSubmitPrompt"},
		HookEvents: []string{
			"sessionStart", "sessionEnd", "preToolUse", "postToolUse",
			"postToolUseFailure", "subagentStart", "subagentStop",
			"beforeSubmitPrompt", "preCompact", "stop",
		},
		MCPFile:     ".cursor/mcp.json",
		MCPFormat:   MCPFormatJSON,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
