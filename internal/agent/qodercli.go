package agent

// Evidence: docs/research/qodercli.md; https://docs.qoder.com/cli
// Qoder's hook schema is Claude-compatible; skill frontmatter is exactly
// name + description. Its settings.json parser accepts // comments, so the
// hooks document decodes as JSONC.
func init() {
	Register(Spec{
		ID:           "qodercli",
		DocsURL:      "https://docs.qoder.com/cli",
		DisplayNames: []string{"Qoder"},
		ConfigDir:    ".qoder",
		GlobalDir:    "~/.qoder",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".qoder/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description"},
		},
		Hooks: HookSpec{
			File:      ".qoder/settings.json",
			Format:    DialectJSONC,
			Shape:     ShapeGroups,
			Wrapper:   WrapperSettings,
			EventCase: CasePascal,
			Events:    CanonicalEvents,
		},
		MCP: MCPSpec{
			File:     ".mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
