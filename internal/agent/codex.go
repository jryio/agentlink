package agent

// Evidence: docs/research/codex.md; https://developers.openai.com/codex/cli
func init() {
	Register(Spec{
		ID:            "codex",
		DocsURL:       "https://developers.openai.com/codex/cli",
		DisplayNames:  []string{"OpenAI Codex", "Codex"},
		ConfigDir:     ".codex",
		GlobalDir:     "~/.codex",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".codex/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "license", "compatibility", "metadata", "allowed-tools"},
		HooksFile:     ".codex/hooks.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperHooks,
		HookEventCase: CasePascal,
		HookEvents: []string{
			"SessionStart", "SessionEnd", "SubagentStart", "SubagentStop",
			"PreToolUse", "PermissionRequest", "PostToolUse", "PreCompact",
			"PostCompact", "UserPromptSubmit", "Stop",
		},
		MCPFile:     ".codex/config.toml",
		MCPFormat:   MCPFormatTOML,
		MCPTableKey: "mcp_servers",
		MCPEnvField: "env",
	})
}
