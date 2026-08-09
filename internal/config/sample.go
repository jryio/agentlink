package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
)

// SampleFor returns a starter configuration for the project rooted at dir,
// scoped hub-and-spoke to the coding agents detected there. Detection is
// filesystem-only and confined to dir via os.Root: config directories,
// distinctive instruction files, and declared root files. AGENTS.md alone
// never marks an agent present — it is the canonical hub file and nearly
// universal. When nothing is detected the default agents↔claude template is
// emitted so the file always validates.
func SampleFor(dir string) ([]byte, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open project directory for agent detection: %w", err)
	}
	defer func() { _ = root.Close() }()

	var out strings.Builder
	out.WriteString(`# yaml-language-server: $schema=./agentlink.schema.json
# Canonical artifacts live in .agents (and AGENTS.md); each detected agent is
# peered against that hub. Most agents read .agents/skills natively — only
# agents that cannot need an activation or pair here.
version: 1

sources:
  project:
    root: .
    relative_to: config

pairs:
`)
	var activations strings.Builder
	detected := 0
	for _, spec := range agent.All() {
		if spec.ID == "agents" {
			continue
		}
		present, err := detectedIn(root, spec)
		if err != nil {
			return nil, fmt.Errorf("detect agent %s: %w", spec.ID, err)
		}
		if !present {
			continue
		}
		detected++
		if primary := primaryInstruction(spec); primary != "" && primary != "AGENTS.md" {
			kind := "siblings"
			if path.Base(primary) != primary {
				kind = "file"
			}
			fmt.Fprintf(&out, `  - id: %s-instructions
    kind: %s
    peers:
      agents: {source: project, path: AGENTS.md}
      %s: {source: project, path: %s}
    normalizer: instructions
    sync: translate
    optional: true

`, spec.ID, kind, spec.ID, primary)
		}
		if !spec.NativeAgents && spec.SkillsDir != "" {
			fmt.Fprintf(&activations, `  - id: %s-skills-live
    expected: {source: project, path: .agents/skills}
    live: {source: project, path: %s}
    optional: true
`, spec.ID, spec.SkillsDir)
		}
		if spec.HooksFile != "" {
			fmt.Fprintf(&out, `  - id: %s-hooks
    kind: file
    peers:
      agents: {source: project, path: .agents/hooks.json}
      %s: {source: project, path: %s}
    normalizer: hook
    sync: translate
    optional: true

`, spec.ID, spec.ID, spec.HooksFile)
		}
	}
	if detected == 0 {
		out.WriteString(`  - id: project-instructions
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

`)
	}
	out.WriteString(`# mcp_servers: peer the canonical .agents/mcp.json against an agent's MCP
# file to compare server wiring (command/args/transport/url and environment
# key names — never secret values). See docs/configuration.md.
#
# Agents with global-only configuration (hermes; kimi hooks/MCP) pair through a
# source rooted at your home directory instead of the project.
`)
	if activations.Len() > 0 {
		out.WriteString("\nactivations:\n")
		out.WriteString(activations.String())
	}
	out.WriteString(`
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
	return []byte(out.String()), nil
}

func primaryInstruction(spec agent.Spec) string {
	if len(spec.Instructions) == 0 {
		return ""
	}
	return spec.Instructions[0]
}

// detectedIn reports whether the agent's footprint exists beneath the
// confined project root. Stat errors other than absence (for example a
// symlink escaping the root) propagate.
func detectedIn(root *os.Root, spec agent.Spec) (bool, error) {
	stat := func(name string) (fs.FileInfo, error) {
		info, err := root.Stat(filepath.FromSlash(name))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		return info, nil
	}
	if spec.ConfigDir != "" {
		info, err := stat(spec.ConfigDir)
		if err != nil {
			return false, err
		}
		if info != nil && info.IsDir() {
			return true, nil
		}
	}
	for _, name := range spec.Instructions {
		if name == "AGENTS.md" {
			continue
		}
		info, err := stat(name)
		if err != nil {
			return false, err
		}
		if info != nil && !info.IsDir() {
			return true, nil
		}
	}
	for _, name := range spec.DetectFiles {
		info, err := stat(name)
		if err != nil {
			return false, err
		}
		if info != nil {
			return true, nil
		}
	}
	return false, nil
}
