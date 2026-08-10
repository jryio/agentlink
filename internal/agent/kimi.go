package agent

// Evidence: docs/research/kimi.md; https://moonshotai.github.io/kimi-cli
// Hooks live only in the global ~/.kimi/config.toml as [[hooks]] tables; MCP
// lives only in global ~/.kimi/mcp.json. Endpoints must come from user config
// rooted at the home directory.
func init() {
	Register(Spec{
		ID:           "kimi",
		DocsURL:      "https://moonshotai.github.io/kimi-cli",
		DisplayNames: []string{"Kimi"},
		ConfigDir:    ".kimi",
		GlobalDir:    "~/.kimi",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".kimi/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "license", "compatibility", "metadata", "type"},
		},
		Hooks: HookSpec{
			Format:    DialectTOML,
			Shape:     ShapeList,
			Wrapper:   WrapperSettings,
			EventCase: CasePascal,
			Events: []string{
				"PreToolUse", "PostToolUse", "PostToolUseFailure", "UserPromptSubmit",
				"Stop", "SessionStart", "SessionEnd", "SubagentStart", "SubagentStop",
				"PreCompact", "PostCompact", "Notification",
			},
		},
		MCP: MCPSpec{
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
