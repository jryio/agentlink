package config

import (
	_ "embed"
	"slices"
)

//go:embed schema.json
var schema []byte

// Schema returns a copy of the editor/validation schema.
func Schema() []byte { return slices.Clone(schema) }

// Sample returns a small project-local configuration with editor schema wiring.
func Sample() []byte {
	return []byte(`# yaml-language-server: $schema=./agentlink.schema.json
version: 1

sources:
  project:
    root: .
    relative_to: config

pairs:
  - id: project-instructions
    name: Project instructions
    kind: siblings
    peers:
      agents: {source: project, path: AGENTS.md}
      claude: {source: project, path: CLAUDE.md}
    normalizer: instructions
    sync: translate
    optional: true

  - id: project-skills
    name: Project skills
    kind: tree
    peers:
      agents: {source: project, path: .agents/skills}
      claude: {source: project, path: .claude/skills}
    normalizer: skill
    sync: translate
    optional: true

ignore:
  - "**/.DS_Store"
  - "**/.git/**"
  - "**/.hg/**"
  - "**/.svn/**"
  - "**/cache/**"
  - "**/node_modules/**"
  - "**/tmp/**"
  - "**/vendor/**"
`)
}
