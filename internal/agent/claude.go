package agent

// Evidence: docs/research/claude.md; https://code.claude.com/docs/en/overview
func init() {
	Register(Spec{
		ID:           "claude",
		DocsURL:      "https://code.claude.com/docs/en/overview",
		DisplayNames: []string{"Claude Code", "Claude"},
		ConfigDir:    ".claude",
		GlobalDir:    "~/.claude",
		Instructions: []string{"CLAUDE.md"},
		SkillsDir:    ".claude/skills",
		NativeAgents: false,
		SkillKeys: []string{
			"name", "description", "when_to_use", "argument-hint", "arguments",
			"disable-model-invocation", "user-invocable", "allowed-tools",
			"disallowed-tools", "model", "effort", "context", "agent",
			"background", "hooks", "paths", "shell", "metadata", "license",
			"compatibility",
		},
		HooksFile:     ".claude/settings.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperSettings,
		HookEventCase: CasePascal,
		HookEvents:    CanonicalEvents,
		MCPFile:       ".mcp.json",
		MCPFormat:     MCPFormatJSON,
		MCPTableKey:   "mcpServers",
		MCPEnvField:   "env",
	})
}
