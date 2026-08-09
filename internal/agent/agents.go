package agent

// agents is the built-in canonical peer: the .agents hub store. Its skills
// layout, hook schema, and MCP shape are the shared vocabulary every other
// target translates to and from.
func init() {
	Register(Spec{
		ID:            "agents",
		DocsURL:       "https://github.com/jryio/agentlink",
		DisplayNames:  []string{"Agent"},
		ConfigDir:     ".agents",
		Instructions:  []string{"AGENTS.md"},
		SkillsDir:     "skills",
		NativeAgents:  true,
		SkillKeys:     U,
		HooksFile:     "hooks.json",
		HooksFormat:   HookFormatJSON,
		HooksShape:    ShapeGroups,
		HooksWrapper:  WrapperBare,
		HookEventCase: CasePascal,
		HookEvents:    CanonicalEvents,
		MCPFile:       "mcp.json",
		MCPFormat:     MCPFormatJSON,
		MCPTableKey:   "mcpServers",
		MCPEnvField:   "env",
	})
}
