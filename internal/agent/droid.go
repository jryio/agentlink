package agent

// Evidence: docs/research/droid.md; https://docs.factory.ai/cli
func init() {
	Register(Spec{
		ID:            "droid",
		DocsURL:       "https://docs.factory.ai/cli",
		DisplayNames:  []string{"Factory Droid", "Droid"},
		ConfigDir:     ".factory",
		GlobalDir:     "~/.factory",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".factory/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "allowed-tools", "enabled", "user-invocable", "disable-model-invocation", "license", "compatibility", "version", "metadata"},
		HooksFile:     ".factory/hooks.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperHooks,
		HookEventCase: CasePascal,
		HookEvents: []string{
			"PreToolUse", "PostToolUse", "UserPromptSubmit", "Notification",
			"Stop", "SubagentStop", "PreCompact", "SessionStart", "SessionEnd",
		},
		MCPFile:     ".factory/mcp.json",
		MCPFormat:   MCPFormatJSON,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
