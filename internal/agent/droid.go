package agent

// Evidence: docs/research/droid.md; https://docs.factory.ai/cli
func init() {
	Register(Spec{
		ID:           "droid",
		DocsURL:      "https://docs.factory.ai/cli",
		DisplayNames: []string{"Factory Droid", "Droid"},
		ConfigDir:    ".factory",
		GlobalDir:    "~/.factory",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".factory/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "allowed-tools", "enabled", "user-invocable", "disable-model-invocation", "license", "compatibility", "version", "metadata"},
		},
		Hooks: HookSpec{
			File:      ".factory/hooks.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperHooks,
			EventCase: CasePascal,
			Events: []string{
				"PreToolUse", "PostToolUse", "UserPromptSubmit", "Notification",
				"Stop", "SubagentStop", "PreCompact", "SessionStart", "SessionEnd",
			},
		},
		MCP: MCPSpec{
			File:     ".factory/mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
