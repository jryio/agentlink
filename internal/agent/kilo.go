package agent

// Evidence: docs/research/kilo.md; https://kilo.ai/docs
// Hooks are TS/JS plugin modules (.kilo/plugin/*); no declarative file.
func init() {
	Register(Spec{
		ID:           "kilo",
		DocsURL:      "https://kilo.ai/docs",
		DisplayNames: []string{"Kilo"},
		ConfigDir:    ".kilo",
		GlobalDir:    "~/.config/kilo",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".kilo/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "license", "compatibility", "metadata"},
		},
		Hooks: HookSpec{Format: DialectCode},
		MCP: MCPSpec{
			File:     "kilo.jsonc",
			Format:   DialectJSONC,
			TableKey: "mcp",
			EnvField: "environment",
		},
	})
}
