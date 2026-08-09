package agent

// Evidence: docs/research/mastracode.md; https://code.mastra.ai
// mastracode ends subagents with SubagentEnd (not SubagentStop).
func init() {
	Register(Spec{
		ID:                "mastracode",
		DocsURL:           "https://code.mastra.ai",
		DisplayNames:      []string{"Mastra Code"},
		ConfigDir:         ".mastracode",
		GlobalDir:         "~/.mastracode",
		Instructions:      []string{"AGENTS.md"},
		SkillsDir:         ".mastracode/skills",
		NativeAgents:      true,
		SkillKeys:         []string{"name", "description", "license", "compatibility", "user-invocable", "metadata"},
		HooksFile:         ".mastracode/hooks.json",
		HooksFormat:       HookFormatJSON,
		HooksShape:        ShapeFlat,
		HooksWrapper:      WrapperBare,
		HooksTimeoutScale: 1000, // milliseconds
		HooksMatcherShape: "tool_name",
		HookEventCase:     CasePascal,
		HookEventMap:      map[string]string{"SubagentStop": "SubagentEnd"},
		HookEvents: []string{
			"PreToolUse", "PostToolUse", "Stop", "UserPromptSubmit",
			"SessionStart", "SessionEnd", "Notification", "PermissionRequest",
			"SubagentStart", "SubagentEnd",
		},
		MCPFile:     ".mastracode/mcp.json",
		MCPFormat:   MCPFormatJSON,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
