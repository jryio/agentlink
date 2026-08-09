package agent

// Evidence: docs/research/qodercli.md; https://docs.qoder.com/cli
// Qoder's hook schema is Claude-compatible; skill frontmatter is exactly
// name + description.
func init() {
	Register(Spec{
		ID:            "qodercli",
		DocsURL:       "https://docs.qoder.com/cli",
		DisplayNames:  []string{"Qoder"},
		ConfigDir:     ".qoder",
		GlobalDir:     "~/.qoder",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     ".qoder/skills",
		NativeAgents:  true,
		SkillKeys:     []string{"name", "description"},
		HooksFile:     ".qoder/settings.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperSettings,
		HookEventCase: CasePascal,
		HookEvents:    CanonicalEvents,
		MCPFile:       ".mcp.json",
		MCPFormat:     MCPFormatJSON,
		MCPTableKey:   "mcpServers",
		MCPEnvField:   "env",
	})
}
