package agent

// Evidence: docs/research/cursor.md; https://cursor.com/docs/cli
// Cursor uses lowerCamelCase hook events, spells the prompt-submit event
// beforeSubmitPrompt, and prefers `paths` over the legacy `globs` skill key.
func init() {
	Register(Spec{
		ID:           "cursor",
		DocsURL:      "https://cursor.com/docs/cli",
		DisplayNames: []string{"Cursor"},
		ConfigDir:    ".cursor",
		GlobalDir:    "~/.cursor",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".cursor/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "paths", "globs", "disable-model-invocation", "metadata", "user-invocable"},
			Renames:      map[string]string{"globs": "paths"},
		},
		Hooks: HookSpec{
			File:      ".cursor/hooks.json",
			Format:    DialectJSON,
			Shape:     ShapeFlat,
			Wrapper:   WrapperHooks,
			Version:   1,
			EventCase: CaseCamel,
			EventMap:  map[string]string{"UserPromptSubmit": "beforeSubmitPrompt"},
			Events: []string{
				"sessionStart", "sessionEnd", "preToolUse", "postToolUse",
				"postToolUseFailure", "subagentStart", "subagentStop",
				"beforeSubmitPrompt", "preCompact", "stop",
			},
		},
		MCP: MCPSpec{
			File:     ".cursor/mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
