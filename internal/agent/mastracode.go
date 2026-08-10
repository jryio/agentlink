package agent

// Evidence: docs/research/mastracode.md; https://code.mastra.ai
// mastracode ends subagents with SubagentEnd (not SubagentStop).
func init() {
	Register(Spec{
		ID:           "mastracode",
		DocsURL:      "https://code.mastra.ai",
		DisplayNames: []string{"Mastra Code"},
		ConfigDir:    ".mastracode",
		GlobalDir:    "~/.mastracode",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".mastracode/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "license", "compatibility", "user-invocable", "metadata"},
		},
		Hooks: HookSpec{
			File:         ".mastracode/hooks.json",
			Format:       DialectJSON,
			Shape:        ShapeFlat,
			Wrapper:      WrapperBare,
			TimeoutUnit:  TimeoutMilliseconds,
			MatcherShape: MatcherToolName,
			EventCase:    CasePascal,
			EventMap:     map[string]string{"SubagentStop": "SubagentEnd"},
			Events: []string{
				"PreToolUse", "PostToolUse", "Stop", "UserPromptSubmit",
				"SessionStart", "SessionEnd", "Notification", "PermissionRequest",
				"SubagentStart", "SubagentEnd",
			},
		},
		MCP: MCPSpec{
			File:     ".mastracode/mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
