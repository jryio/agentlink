package agent

// Evidence: docs/research/gemini.md; https://github.com/google-gemini/gemini-cli
// Gemini names tool hooks BeforeTool/AfterTool, agent lifecycle
// BeforeAgent/AfterAgent, and compaction PreCompress; its per-hook timeout
// counts milliseconds. BeforeModel/AfterModel/BeforeToolSelection have no
// canonical counterpart and round-trip verbatim.
func init() {
	Register(Spec{
		ID:           "gemini",
		DocsURL:      "https://github.com/google-gemini/gemini-cli",
		DisplayNames: []string{"Gemini CLI", "Gemini"},
		ConfigDir:    ".gemini",
		GlobalDir:    "~/.gemini",
		Instructions: []string{"GEMINI.md"},
		Skills: SkillSpec{
			Dir:          ".gemini/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description"},
		},
		Hooks: HookSpec{
			File:      ".gemini/settings.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperSettings,
			EventCase: CasePascal,
			EventMap: map[string]string{
				"PreToolUse":       "BeforeTool",
				"PostToolUse":      "AfterTool",
				"UserPromptSubmit": "BeforeAgent",
				"Stop":             "AfterAgent",
				"PreCompact":       "PreCompress",
			},
			Events: []string{
				"SessionStart", "SessionEnd", "BeforeAgent", "AfterAgent",
				"BeforeModel", "AfterModel", "BeforeToolSelection",
				"BeforeTool", "AfterTool", "PreCompress", "Notification",
			},
			TimeoutUnit: TimeoutMilliseconds,
		},
		MCP: MCPSpec{
			File:     ".gemini/settings.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
