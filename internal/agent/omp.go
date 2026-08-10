package agent

// Evidence: docs/research/omp.md; https://github.com/can1357/oh-my-pi
// Hooks are in-process TS/JS factory modules (.omp/hooks/pre|post/*); there is
// no declarative hook file. omp writes disable-model-invocation camelCase.
func init() {
	Register(Spec{
		ID:           "omp",
		DocsURL:      "https://github.com/can1357/oh-my-pi",
		DisplayNames: []string{"Oh My Pi", "OMP"},
		ConfigDir:    ".omp",
		GlobalDir:    "~/.omp/agent",
		Instructions: []string{"AGENTS.md"},
		Skills: SkillSpec{
			Dir:          ".omp/skills",
			NativeAgents: true,
			Keys:         []string{"name", "description", "globs", "alwaysApply", "hide", "disableModelInvocation"},
			Renames:      map[string]string{"disable-model-invocation": "disableModelInvocation"},
		},
		Hooks: HookSpec{Format: DialectCode},
		MCP: MCPSpec{
			File:     ".omp/mcp.json",
			Format:   DialectJSON,
			TableKey: "mcpServers",
			EnvField: "env",
		},
	})
}
