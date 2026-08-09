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
		SkillsDir:    ".kilo/skills",
		NativeAgents: true,
		SkillKeys:    []string{"name", "description", "license", "compatibility", "metadata"},
		HooksFormat:  HookFormatCode,
		MCPFile:      "kilo.jsonc",
		MCPFormat:    MCPFormatJSONC,
		MCPTableKey:  "mcp",
		MCPEnvField:  "environment",
	})
}
