package agent

// Evidence: docs/research/amp.md; https://ampcode.com/manual
// Amp's hook equivalents are TypeScript plugins (.amp/plugins/*); there is no
// declarative hook file. MCP servers live under the amp.mcpServers key of
// JSONC settings files.
func init() {
	Register(Spec{
		ID:           "amp",
		DocsURL:      "https://ampcode.com/manual",
		DisplayNames: []string{"Amp"},
		ConfigDir:    ".amp",
		GlobalDir:    "~/.config/amp",
		Instructions: []string{"AGENTS.md"},
		SkillsDir:    ".agents/skills",
		NativeAgents: true,
		SkillKeys:    []string{"name", "description", "mcpServers"},
		HooksFormat:  HookFormatCode,
		MCPFile:      ".amp/settings.json",
		MCPFormat:    MCPFormatJSONC,
		MCPTableKey:  "amp.mcpServers",
		MCPEnvField:  "env",
	})
}
