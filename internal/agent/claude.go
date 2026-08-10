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
		Skills: SkillSpec{
			Dir:          ".claude/skills",
			NativeAgents: false,
			Keys: []string{
				"name", "description", "when_to_use", "argument-hint", "arguments",
				"disable-model-invocation", "user-invocable", "allowed-tools",
				"disallowed-tools", "model", "effort", "context", "agent",
				"background", "hooks", "paths", "shell", "metadata", "license",
				"compatibility",
			},
		},
		Hooks: HookSpec{
			File:      ".claude/settings.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperSettings,
			EventCase: CasePascal,
			Events:    CanonicalEvents,
		},
		MCP: MCPSpec{
			File:     ".mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
