package agent

// Evidence: docs/research/goose.md; https://block.github.io/goose
// Goose reads .agents/skills natively. Declarative hooks exist only inside a
// plugin package, so agentlink owns one plugin dir. MCP servers are
// extensions of type stdio|streamable_http in the global config.yaml; the
// project MCP file is not used, so MCP.File stays empty.
func init() {
	Register(Spec{
		ID:           "goose",
		DocsURL:      "https://block.github.io/goose",
		DisplayNames: []string{"Goose"},
		ConfigDir:    "",
		GlobalDir:    "~/.config/goose",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".agents/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "metadata", "argument-hint", "arguments"},
		},
		Hooks: HookSpec{
			File:      ".agents/plugins/agentlink/hooks/hooks.json",
			Format:    DialectJSON,
			Shape:     ShapeGroups,
			Wrapper:   WrapperHooks,
			EventCase: CasePascal,
			Events: []string{
				"SessionStart", "SessionEnd", "Stop", "UserPromptSubmit",
				"PreToolUse", "PostToolUse", "PostToolUseFailure",
			},
		},
		MCP: MCPSpec{
			Format:   DialectYAML,
			TableKey: "extensions",
			EnvField: "env",
		},
	})
}
