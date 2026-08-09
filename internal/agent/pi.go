package agent

// Evidence: docs/research/pi.md; https://github.com/badlogic/pi-mono
// Hooks are in-process TypeScript extensions (.pi/extensions/*.ts); there is
// no declarative hook file. pi has no MCP support by design.
func init() {
	Register(Spec{
		ID:           "pi",
		DocsURL:      "https://github.com/badlogic/pi-mono",
		DisplayNames: []string{"Pi"},
		ConfigDir:    ".pi",
		GlobalDir:    "~/.pi/agent",
		Instructions: []string{"AGENTS.md"},
		SkillsDir:    ".pi/skills",
		NativeAgents: true,
		SkillKeys: []string{
			"name", "description", "license", "compatibility", "metadata",
			"allowed-tools", "disable-model-invocation",
		},
		HooksFormat: HookFormatCode,
		MCPFormat:   MCPFormatNone,
	})
}
