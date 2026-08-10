package agent

// agents is the built-in canonical peer: the .agents hub store. Its skills
// layout, hook schema, and MCP shape are the shared vocabulary every other
// target translates to and from.
func init() {
	Register(Spec{
		ID:           HubID,
		DocsURL:      "https://github.com/jryio/agentlink",
		DisplayNames: []string{"Agent"},
		ConfigDir:    ".agents",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          "skills",
			NativeAgents: true,
			Keys:         U,
		},
		Hooks: HookSpec{
			File:      "hooks.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperBare,
			EventCase: CasePascal,
			Events:    CanonicalEvents,
		},
		MCP: MCPSpec{
			File:     "mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
