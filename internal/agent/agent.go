// Package agent registers the coding-agent targets agentlink can compare and
// translate. Per-agent variation is tabular: each target contributes one Spec
// of directory names, frontmatter keys, hook event names, and MCP wiring.
// Behavioral variation lives in the format-keyed formatters (internal/format)
// and the parameterized normalizers (internal/link), not in per-agent code.
package agent

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// HookFormat identifies the on-disk shape of an agent's declarative hooks.
type HookFormat string

const (
	// HookFormatNone means the agent has no hook mechanism.
	HookFormatNone HookFormat = ""
	// HookFormatCode means hooks are in-process code modules; no declarative
	// file exists for agentlink to compare or translate.
	HookFormatCode HookFormat = "code"
	// HookFormatJSON means a JSON hooks document.
	HookFormatJSON HookFormat = "json"
	// HookFormatTOML means TOML hook entries.
	HookFormatTOML HookFormat = "toml"
	// HookFormatYAML means YAML hook entries.
	HookFormatYAML HookFormat = "yaml"
)

// MCPFormat identifies the on-disk shape of an agent's MCP configuration.
type MCPFormat string

const (
	// MCPFormatNone means the agent has no MCP configuration.
	MCPFormatNone MCPFormat = ""
	// MCPFormatJSON means strict JSON.
	MCPFormatJSON MCPFormat = "json"
	// MCPFormatJSONC means JSON with comments and trailing commas.
	MCPFormatJSONC MCPFormat = "jsonc"
	// MCPFormatTOML means TOML.
	MCPFormatTOML MCPFormat = "toml"
	// MCPFormatYAML means YAML.
	MCPFormatYAML MCPFormat = "yaml"
)

// Hook event casing styles applied to canonical PascalCase event names.
const (
	CasePascal = "pascal"
	CaseCamel  = "camel"
	CaseSnake  = "snake"
)

// U is the universal skill-frontmatter compare set: the keys a canonical
// .agents SKILL.md may carry. Comparison keeps a key only when it is in U and
// understood by both peers, so tool-only keys never masquerade as drift.
var U = []string{
	"name",
	"description",
	"license",
	"compatibility",
	"metadata",
	"allowed-tools",
	"user-invocable",
	"disable-model-invocation",
	"argument-hint",
}

// Hook document structure styles.
const (
	ShapeGroups = "groups" // event → [{matcher, hooks: [...]}]
	ShapeFlat   = "flat"   // event → [{command, matcher?, ...}]
	ShapeList   = "list"   // [{event, command, matcher?, ...}] (kimi [[hooks]])

	WrapperBare     = "bare"     // the file is the event map
	WrapperHooks    = "hooks"    // the file wraps the map as {"hooks": {...}}
	WrapperSettings = "settings" // the map lives under "hooks" in a larger settings doc
)

// CanonicalEvents is the hook event vocabulary of the canonical .agents
// hooks.json: the Claude-derived PascalCase set that most targets mirror.
var CanonicalEvents = []string{
	"SessionStart",
	"SessionEnd",
	"UserPromptSubmit",
	"PreToolUse",
	"PostToolUse",
	"PostToolUseFailure",
	"PermissionRequest",
	"Notification",
	"SubagentStart",
	"SubagentStop",
	"Stop",
	"PreCompact",
	"PostCompact",
}

// Spec describes one coding-agent target. All fields are data; translation
// and comparison behavior derives from them generically.
type Spec struct {
	ID           string            // kebab id, e.g. "claude"
	DocsURL      string            // primary documentation URL
	DisplayNames []string          // instruction-prose names, longest first: ["Claude Code", "Claude"]
	ConfigDir    string            // ".claude"; "" when config lives in root files or is global-only
	GlobalDir    string            // "~/.claude"; "" if none
	Instructions []string          // project instruction filenames in priority order
	DetectFiles  []string          // extra project-root files that mark this agent present (for agents without a config dir)
	SkillsDir    string            // project skills dir, e.g. ".claude/skills"; "" = no project skills
	NativeAgents bool              // tool discovers .agents/skills itself
	SkillKeys    []string          // SKILL.md frontmatter keys the tool understands, as written on disk
	SkillRenames map[string]string // canonical key -> target key (e.g. cursor {"globs": "paths"})
	HooksFile    string            // project hooks file; "" when hooks are code-only or global-only
	HooksFormat  HookFormat
	// HooksShape is "groups" for the Claude three-level form (event → matcher
	// groups → handlers) or "flat" for event → handler entries that carry
	// their own matcher.
	HooksShape string
	// HooksWrapper is "bare" when the file is the event map itself, "hooks"
	// when the file wraps it as {"hooks": {...}}, or "settings" when the hooks
	// object lives under the "hooks" key of a larger settings document that
	// must be merged on write, never replaced.
	HooksWrapper string
	// HooksVersion is the file-format version emitted with wrapper "hooks"
	// (1 for cursor/copilot; 0 omits the version field).
	HooksVersion int
	// HooksTimeoutField names the per-hook timeout key; "" means "timeout"
	// (copilot uses "timeoutSec").
	HooksTimeoutField string
	// HooksTimeoutScale multiplies canonical seconds into the target's units
	// (mastracode uses milliseconds: 1000). Zero means 1.
	HooksTimeoutScale int
	// HooksMatcherShape is "" for a plain string matcher or "tool_name" for
	// mastracode's {"tool_name": regex} object form.
	HooksMatcherShape string
	HookEventCase     string            // CasePascal | CaseCamel | CaseSnake
	HookEventMap      map[string]string // canonical event -> target event for exceptions the case transform misses
	HookEvents        []string          // effective target event names the tool supports (post-transform)
	MCPFile           string            // project MCP config file; "" = none or global-only
	MCPFormat         MCPFormat
	MCPTableKey       string // "mcpServers" | "mcp_servers" | "mcp" | "amp.mcpServers" | "extensions"
	MCPEnvField       string // env map field; "env" default, "environment" for opencode/kilo; "" = env compare unsupported
}

var registry = map[string]Spec{}

// Register adds a target. It panics on a duplicate ID or an invalid spec; it
// is called from init() in the per-agent files of this package.
func Register(s Spec) {
	if err := s.validate(); err != nil {
		panic(fmt.Sprintf("agent: invalid spec %q: %v", s.ID, err))
	}
	if _, dup := registry[s.ID]; dup {
		panic(fmt.Sprintf("agent: duplicate registration %q", s.ID))
	}
	registry[s.ID] = s
}

// Get returns the spec registered under id.
func Get(id string) (Spec, bool) {
	spec, ok := registry[id]
	return spec, ok
}

// All returns every registered spec sorted by ID.
func All() []Spec {
	specs := make([]Spec, 0, len(registry))
	for _, spec := range registry {
		specs = append(specs, spec)
	}
	slices.SortFunc(specs, func(a, b Spec) int { return strings.Compare(a.ID, b.ID) })
	return specs
}

func (s Spec) validate() error {
	if s.ID == "" {
		return fmt.Errorf("ID is required")
	}
	if len(s.DisplayNames) == 0 {
		return fmt.Errorf("DisplayNames is required")
	}
	switch s.HookEventCase {
	case "", CasePascal, CaseCamel, CaseSnake:
	default:
		return fmt.Errorf("HookEventCase %q is not pascal, camel, or snake", s.HookEventCase)
	}
	if (s.HooksFormat == HookFormatNone || s.HooksFormat == HookFormatCode) && s.HooksFile != "" {
		return fmt.Errorf("HooksFile %q set with hooks format %q", s.HooksFile, s.HooksFormat)
	}
	if s.HooksFormat == HookFormatJSON || s.HooksFormat == HookFormatTOML || s.HooksFormat == HookFormatYAML {
		if s.HooksShape != ShapeGroups && s.HooksShape != ShapeFlat && s.HooksShape != ShapeList {
			return fmt.Errorf("HooksShape %q is not groups, flat, or list", s.HooksShape)
		}
		if s.HooksWrapper != WrapperBare && s.HooksWrapper != WrapperHooks && s.HooksWrapper != WrapperSettings {
			return fmt.Errorf("HooksWrapper %q is not bare, hooks, or settings", s.HooksWrapper)
		}
	}
	if s.MCPFormat == MCPFormatNone && s.MCPFile != "" {
		return fmt.Errorf("MCPFile %q set without an MCP format", s.MCPFile)
	}
	if s.MCPFormat != MCPFormatNone && s.MCPFile != "" && s.MCPTableKey == "" {
		return fmt.Errorf("MCPTableKey is required with MCPFile %q", s.MCPFile)
	}
	return nil
}

// HookEvent resolves a canonical event to the target's effective name. The
// second result reports whether the target supports the event at all;
// unsupported events are dropped with a warning during translation.
func (s Spec) HookEvent(canonical string) (string, bool) {
	name := s.hookEventName(canonical)
	return name, slices.Contains(s.HookEvents, name)
}

// HookCanonical resolves a target event name back to its canonical form.
func (s Spec) HookCanonical(target string) (string, bool) {
	for _, canonical := range CanonicalEvents {
		if s.hookEventName(canonical) == target {
			return canonical, true
		}
	}
	return "", false
}

func (s Spec) hookEventName(canonical string) string {
	if mapped, ok := s.HookEventMap[canonical]; ok {
		return mapped
	}
	return transformCase(canonical, s.HookEventCase)
}

func transformCase(name, style string) string {
	switch style {
	case CaseCamel:
		runes := []rune(name)
		runes[0] = unicode.ToLower(runes[0])
		return string(runes)
	case CaseSnake:
		var out strings.Builder
		for i, r := range name {
			if unicode.IsUpper(r) && i > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(unicode.ToLower(r))
		}
		return out.String()
	default:
		return name
	}
}
