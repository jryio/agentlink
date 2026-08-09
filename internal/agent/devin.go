package agent

// Evidence: docs/research/devin.md; https://docs.devin.ai/cli
// Devin spells the post-compaction event PostCompaction and has no
// PreCompact counterpart.
func init() {
	Register(Spec{
		ID:            "devin",
		DocsURL:       "https://docs.devin.ai/cli",
		DisplayNames:  []string{"Devin"},
		ConfigDir:     ".devin",
		GlobalDir:     "~/.config/devin",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".devin/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "argument-hint", "model", "subagent", "agent", "allowed-tools", "permissions", "triggers"},
		HooksFile:     ".devin/hooks.v1.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperBare,
		HookEventCase: CasePascal,
		HookEventMap:  map[string]string{"PostCompact": "PostCompaction"},
		HookEvents: []string{
			"PreToolUse", "PostToolUse", "PermissionRequest", "UserPromptSubmit",
			"Stop", "PostCompaction", "SessionStart", "SessionEnd",
		},
		MCPFile:     ".devin/mcp_config.json",
		MCPFormat:   MCPFormatJSONC,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
