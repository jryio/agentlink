package agent

// Evidence: docs/research/hermes.md; https://github.com/NousResearch/hermes-agent
// Hermes has no project config dir: hooks and MCP servers live only in the
// global ~/.hermes/config.yaml. Event names are snake_case with *_tool_call
// spellings for the tool events.
func init() {
	Register(Spec{
		ID:           "hermes",
		DocsURL:      "https://github.com/NousResearch/hermes-agent",
		DisplayNames: []string{"Hermes"},
		ConfigDir:    "",
		GlobalDir:    "~/.hermes",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			NativeAgents: false,
			Keys: []string{
				"name", "description", "version", "author", "license", "platforms",
				"metadata", "required_environment_variables", "required_credential_files",
				"prerequisites",
			},
		},
		Hooks: HookSpec{
			Format:    DialectYAML,
			Shape:     ShapeFlat,
			Wrapper:   WrapperSettings,
			EventCase: CaseSnake,
			EventMap: map[string]string{
				"PreToolUse":   "pre_tool_call",
				"PostToolUse":  "post_tool_call",
				"SessionStart": "on_session_start",
				"SessionEnd":   "on_session_end",
			},
			Events: []string{
				"pre_tool_call", "post_tool_call", "on_session_start",
				"on_session_end", "subagent_start", "subagent_stop",
			},
		},
		MCP: MCPSpec{
			Format:   DialectYAML,
			TableKey: "mcp_servers",
			EnvField: "env",
		},
	})
}
