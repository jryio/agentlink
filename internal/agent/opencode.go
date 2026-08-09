package agent

// Evidence: docs/research/opencode.md; https://opencode.ai/docs
// Hooks are JS/TS plugin modules; no declarative file. MCP servers use a
// single-array command and an environment (not env) map.
func init() {
	Register(Spec{
		ID:           "opencode",
		DocsURL:      "https://opencode.ai/docs",
		DisplayNames: []string{"opencode"},
		ConfigDir:    ".opencode",
		GlobalDir:    "~/.config/opencode",
		Instructions: []string{"AGENTS.md"},
		SkillsDir:    ".opencode/skills",
		NativeAgents: true,
		SkillKeys:    []string{"name", "description", "license", "compatibility", "metadata"},
		HooksFormat:  HookFormatCode,
		MCPFile:      "opencode.json",
		MCPFormat:    MCPFormatJSON,
		MCPTableKey:  "mcp",
		MCPEnvField:  "environment",
	})
}
