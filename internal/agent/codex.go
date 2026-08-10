package agent

// Evidence: docs/research/codex.md; https://developers.openai.com/codex/cli
func init() {
	Register(Spec{
		ID:           "codex",
		DocsURL:      "https://developers.openai.com/codex/cli",
		DisplayNames: []string{"OpenAI Codex", "Codex"},
		ConfigDir:    ".codex",
		GlobalDir:    "~/.codex",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			// Codex discovers .agents/skills itself; it has no skills dir of
			// its own (docs/research/codex.md), so Dir stays empty.
			NativeAgents: true,
			Keys:         []string{"name", "description", "license", "compatibility", "metadata", "allowed-tools"},
		},
		Hooks: HookSpec{
			File:      ".codex/hooks.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperHooks,
			EventCase: CasePascal,
			Events: []string{
				"SessionStart", "SessionEnd", "SubagentStart", "SubagentStop",
				"PreToolUse", "PermissionRequest", "PostToolUse", "PreCompact",
				"PostCompact", "UserPromptSubmit", "Stop",
			},
		},
		MCP: MCPSpec{
			File:     ".codex/config.toml",
			Format:   DialectTOML,
			TableKey: "mcp_servers",
			EnvField: "env",
		},
	})
}
