package agent

// Evidence: docs/research/crush.md; https://github.com/charmbracelet/crush
// crush prefers the executable crushrc, which agentlink never writes; the
// legacy declarative crush.json remains the compare/translate target. Only
// the PreToolUse hook event exists.
func init() {
	Register(Spec{
		ID:            "crush",
		DocsURL:       "https://github.com/charmbracelet/crush",
		DisplayNames:  []string{"Crush"},
		ConfigDir:     "",
		GlobalDir:     "~/.config/crush",
		Instructions:  []string{"AGENTS.md"},
		DetectFiles:   []string{"crush.json", ".crush.json", "crushrc", ".crushrc"},
		SkillsDir:     ".crush/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description", "user-invocable", "disable-model-invocation", "license", "compatibility", "metadata"},
		HooksFile:     "crush.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeFlat,
		HooksWrapper:  WrapperSettings,
		HookEventCase: CasePascal,
		HookEvents:    []string{"PreToolUse"},
		MCPFile:       "crush.json",
		MCPFormat:     MCPFormatJSON,
		MCPTableKey:   "mcp",
		MCPEnvField:   "env",
	})
}
