package agent

// Evidence: docs/research/gemini.md; https://github.com/google-gemini/gemini-cli
// Gemini names tool hooks BeforeTool/AfterTool and compaction PreCompress;
// it has no prompt-submit, stop, or subagent hook events.
func init() {
	Register(Spec{
		ID:            "gemini",
		DocsURL:       "https://github.com/google-gemini/gemini-cli",
		DisplayNames:  []string{"Gemini CLI", "Gemini"},
		ConfigDir:     ".gemini",
		GlobalDir:     "~/.gemini",
		Instructions:  []string{"GEMINI.md"},
		SkillsDir:     ".gemini/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description"},
		HooksFile:     ".gemini/settings.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperSettings,
		HookEventCase: CasePascal,
		HookEventMap: map[string]string{
			"PreToolUse":  "BeforeTool",
			"PostToolUse": "AfterTool",
			"PreCompact":  "PreCompress",
		},
		HookEvents: []string{
			"SessionStart", "SessionEnd", "BeforeTool", "AfterTool",
			"PreCompress", "Notification",
		},
		MCPFile:     ".gemini/settings.json",
		MCPFormat:   MCPFormatJSON,
		MCPTableKey: "mcpServers",
		MCPEnvField: "env",
	})
}
