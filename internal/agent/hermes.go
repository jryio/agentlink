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
		SkillsDir:    "",
		NativeAgents: false,
		SkillKeys: []string{
			"name", "description", "version", "author", "license", "platforms",
			"metadata", "required_environment_variables", "required_credential_files",
			"prerequisites",
		},
		HooksFormat:   HookFormatYAML,
		HooksShape:    ShapeFlat,
		HooksWrapper:  WrapperSettings,
		HookEventCase: CaseSnake,
		HookEventMap: map[string]string{
			"PreToolUse":   "pre_tool_call",
			"PostToolUse":  "post_tool_call",
			"SessionStart": "on_session_start",
			"SessionEnd":   "on_session_end",
		},
		HookEvents: []string{
			"pre_tool_call", "post_tool_call", "on_session_start",
			"on_session_end", "subagent_start", "subagent_stop",
		},
		MCPFormat:   MCPFormatYAML,
		MCPTableKey: "mcp_servers",
		MCPEnvField: "env",
	})
}
