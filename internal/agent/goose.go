package agent

// Evidence: docs/research/goose.md; https://block.github.io/goose
// Goose reads .agents/skills natively. Declarative hooks exist only inside a
// plugin package, so agentlink owns one plugin dir. MCP servers are
// extensions of type stdio|streamable_http in the global config.yaml; the
// project MCP file is not used, so MCPFile stays empty.
func init() {
	Register(Spec{
		ID:            "goose",
		DocsURL:       "https://block.github.io/goose",
		DisplayNames:  []string{"Goose"},
		ConfigDir:     "",
		GlobalDir:     "~/.config/goose",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".agents/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "metadata", "argument-hint", "arguments"},
		HooksFile:     ".agents/plugins/agentlink/hooks/hooks.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperHooks,
		HookEventCase: CasePascal,
		HookEvents: []string{
			"SessionStart", "SessionEnd", "Stop", "UserPromptSubmit",
			"PreToolUse", "PostToolUse", "PostToolUseFailure",
		},
		MCPFormat:   MCPFormatYAML,
		MCPTableKey: "extensions",
		MCPEnvField: "env",
	})
}
