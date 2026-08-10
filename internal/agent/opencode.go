package agent

// Evidence: docs/research/opencode.md; https://opencode.ai/docs
// Hooks are JS/TS plugin modules; no declarative file. MCP servers use a
// single-array command and an environment (not env) map. Project config may
// be opencode.json or opencode.jsonc — both parse as JSONC.
func init() {
	Register(Spec{
		ID:           "opencode",
		DocsURL:      "https://opencode.ai/docs",
		DisplayNames: []string{"opencode"},
		ConfigDir:    ".opencode",
		GlobalDir:    "~/.config/opencode",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".opencode/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "license", "compatibility", "metadata"},
		},
		Hooks: HookSpec{Format: DialectCode},
		MCP: MCPSpec{
			File:     "opencode.json",
			Format:   DialectJSONC,
			TableKey: "mcp",
			EnvField: "environment",
		},
	})
}
