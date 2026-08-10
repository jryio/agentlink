package agent

// Evidence: docs/research/devin.md; https://docs.devin.ai/cli
// Devin spells the post-compaction event PostCompaction and has no
// PreCompact counterpart.
func init() {
	Register(Spec{
		ID:           "devin",
		DocsURL:      "https://docs.devin.ai/cli",
		DisplayNames: []string{"Devin"},
		ConfigDir:    ".devin",
		GlobalDir:    "~/.config/devin",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".devin/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "argument-hint", "model", "subagent", "agent", "allowed-tools", "permissions", "triggers"},
		},
		Hooks: HookSpec{
			File:      ".devin/hooks.v1.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperBare,
			EventCase: CasePascal,
			EventMap:  map[string]string{"PostCompact": "PostCompaction"},
			Events: []string{
				"PreToolUse", "PostToolUse", "PermissionRequest", "UserPromptSubmit",
				"Stop", "PostCompaction", "SessionStart", "SessionEnd",
			},
		},
		MCP: MCPSpec{
			File:     ".devin/mcp_config.json",
			Format:   DialectJSONC,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
