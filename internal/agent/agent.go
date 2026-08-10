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

// Dialect identifies the on-disk shape of an agent's declarative documents.
// Hooks and MCP configuration share one vocabulary so a file that allows
// comments (JSONC) is decoded leniently on every path that reads it.
type Dialect string

const (
	// DialectNone means the agent has no declarative document of the kind.
	DialectNone Dialect = ""
	// DialectCode means hooks are in-process code modules; no declarative
	// file exists for agentlink to compare or translate.
	DialectCode Dialect = "code"
	// DialectJSON means strict JSON.
	DialectJSON Dialect = "json"
	// DialectJSONC means JSON with // comments and trailing commas.
	DialectJSONC Dialect = "jsonc"
	// DialectTOML means TOML.
	DialectTOML Dialect = "toml"
	// DialectYAML means YAML.
	DialectYAML Dialect = "yaml"
)

// EventCase is a hook event casing style applied to canonical PascalCase
// event names.
type EventCase string

// Hook event casing styles.
const (
	CasePascal EventCase = "pascal"
	CaseCamel  EventCase = "camel"
	CaseSnake  EventCase = "snake"
)

// TimeoutUnit is the unit of a target's per-hook timeout field
// (HookSpec.TimeoutUnit). Canonical hooks carry seconds; a target whose
// timeout field counts milliseconds must say so explicitly so translation
// rescales instead of passing values through.
type TimeoutUnit string

const (
	// TimeoutSeconds is the default: the target's timeout field is seconds.
	TimeoutSeconds TimeoutUnit = ""
	// TimeoutMilliseconds means the target's timeout field is milliseconds
	// (gemini, mastracode).
	TimeoutMilliseconds TimeoutUnit = "ms"
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

// HookShape is a hook document structure style.
type HookShape string

// Hook document structure styles.
const (
	ShapeGroups HookShape = "groups" // event → [{matcher, hooks: [...]}]
	ShapeFlat   HookShape = "flat"   // event → [{command, matcher?, ...}]
	ShapeList   HookShape = "list"   // [{event, command, matcher?, ...}] (kimi [[hooks]])
)

// HookWrapper is the envelope a hooks document wraps the event map in.
type HookWrapper string

// Hook document envelopes.
const (
	WrapperBare     HookWrapper = "bare"     // the file is the event map
	WrapperHooks    HookWrapper = "hooks"    // the file wraps the map as {"hooks": {...}}
	WrapperSettings HookWrapper = "settings" // the map lives under "hooks" in a larger settings doc
)

// MatcherShape is the encoding of a per-hook matcher.
type MatcherShape string

const (
	// MatcherPlain is a plain string matcher (the zero value).
	MatcherPlain MatcherShape = ""
	// MatcherToolName is mastracode's {"tool_name": regex} object form.
	MatcherToolName MatcherShape = "tool_name"
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
// and comparison behavior derives from them generically. Per-capability
// variation is grouped so an invalid combination is scoped to one subsystem:
// Skills, Hooks, and MCP each carry only the fields their subsystem reads.
type Spec struct {
	ID           string   // kebab id, e.g. "claude"
	DocsURL      string   // primary documentation URL
	DisplayNames []string // instruction-prose names, longest first: ["Claude Code", "Claude"]
	ConfigDir    string   // ".claude"; "" when config lives in root files or is global-only
	GlobalDir    string   // "~/.claude"; "" if none
	Instructions []string // project instruction filenames in priority order
	DetectFiles  []string // extra project-root files that mark this agent present (for agents without a config dir)
	Skills       SkillSpec
	Hooks        HookSpec
	MCP          MCPSpec
}

// SkillSpec describes how a target stores skills and which SKILL.md
// frontmatter keys it understands.
type SkillSpec struct {
	Dir          string            // project skills dir, e.g. ".claude/skills"; "" = no project skills
	NativeAgents bool              // tool discovers .agents/skills itself
	Keys         []string          // SKILL.md frontmatter keys the tool understands, as written on disk
	Renames      map[string]string // canonical key -> target key (e.g. cursor {"globs": "paths"})
}

// HookSpec describes a target's declarative hook document: dialect,
// structure, event vocabulary, and timeout encoding.
type HookSpec struct {
	File   string  // project hooks file; "" when hooks are code-only or global-only
	Format Dialect // DialectCode means hooks are in-process code modules; no declarative file
	// Shape is ShapeGroups for the Claude three-level form (event → matcher
	// groups → handlers), ShapeFlat for event → handler entries that carry
	// their own matcher, or ShapeList for a flat array of hook tables.
	Shape HookShape
	// Wrapper is WrapperBare when the file is the event map itself,
	// WrapperHooks when the file wraps it as {"hooks": {...}}, or
	// WrapperSettings when the hooks object lives under the "hooks" key of a
	// larger settings document that must be merged on write, never replaced.
	Wrapper HookWrapper
	// Version is the file-format version emitted with WrapperHooks
	// (1 for cursor/copilot; 0 omits the version field).
	Version int
	// TimeoutField names the per-hook timeout key; "" means "timeout"
	// (copilot uses "timeoutSec").
	TimeoutField string
	// TimeoutUnit is TimeoutMilliseconds when the target's timeout field
	// counts milliseconds (gemini, mastracode); the zero value means seconds,
	// matching the canonical hooks.json.
	TimeoutUnit TimeoutUnit
	// MatcherShape is MatcherPlain for a plain string matcher or
	// MatcherToolName for mastracode's {"tool_name": regex} object form.
	MatcherShape MatcherShape
	EventCase    EventCase         // casing applied to canonical PascalCase event names
	EventMap     map[string]string // canonical event -> target event for exceptions the case transform misses
	Events       []string          // effective target event names the tool supports (post-transform)
}

// MCPSpec describes how a target records MCP server wiring.
type MCPSpec struct {
	File     string // project MCP config file; "" = none or global-only
	Format   Dialect
	TableKey string // "mcpServers" | "mcp_servers" | "mcp" | "amp.mcpServers" | "extensions"
	EnvField string // env map field; "env" default, "environment" for opencode/kilo; "" = env compare unsupported
}

// HubID is the reserved registry ID of the canonical .agents hub store.
const HubID = "agents"

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
	switch s.Hooks.EventCase {
	case "", CasePascal, CaseCamel, CaseSnake:
	default:
		return fmt.Errorf("Hooks.EventCase %q is not pascal, camel, or snake", s.Hooks.EventCase)
	}
	switch s.Hooks.Format {
	case DialectNone, DialectCode:
		// Hooks are absent or code-only: no declarative document exists, so
		// every document-describing field must be zero.
		if s.Hooks.File != "" {
			return fmt.Errorf("Hooks.File %q set with hooks format %q", s.Hooks.File, s.Hooks.Format)
		}
		if s.Hooks.Shape != "" {
			return fmt.Errorf("Hooks.Shape %q set with hooks format %q", s.Hooks.Shape, s.Hooks.Format)
		}
		if s.Hooks.Wrapper != "" {
			return fmt.Errorf("Hooks.Wrapper %q set with hooks format %q", s.Hooks.Wrapper, s.Hooks.Format)
		}
		if s.Hooks.Version != 0 {
			return fmt.Errorf("Hooks.Version %d set with hooks format %q", s.Hooks.Version, s.Hooks.Format)
		}
		if s.Hooks.TimeoutField != "" {
			return fmt.Errorf("Hooks.TimeoutField %q set with hooks format %q", s.Hooks.TimeoutField, s.Hooks.Format)
		}
		if s.Hooks.TimeoutUnit != TimeoutSeconds {
			return fmt.Errorf("Hooks.TimeoutUnit %q set with hooks format %q", s.Hooks.TimeoutUnit, s.Hooks.Format)
		}
		if s.Hooks.MatcherShape != MatcherPlain {
			return fmt.Errorf("Hooks.MatcherShape %q set with hooks format %q", s.Hooks.MatcherShape, s.Hooks.Format)
		}
		if s.Hooks.EventCase != "" {
			return fmt.Errorf("Hooks.EventCase %q set with hooks format %q", s.Hooks.EventCase, s.Hooks.Format)
		}
		if len(s.Hooks.EventMap) > 0 {
			return fmt.Errorf("Hooks.EventMap set with hooks format %q", s.Hooks.Format)
		}
		if len(s.Hooks.Events) > 0 {
			return fmt.Errorf("Hooks.Events set with hooks format %q", s.Hooks.Format)
		}
	case DialectJSON, DialectJSONC, DialectTOML, DialectYAML:
		switch s.Hooks.Shape {
		case ShapeGroups, ShapeFlat, ShapeList:
		default:
			return fmt.Errorf("Hooks.Shape %q is not groups, flat, or list", s.Hooks.Shape)
		}
		switch s.Hooks.Wrapper {
		case WrapperBare, WrapperHooks, WrapperSettings:
		default:
			return fmt.Errorf("Hooks.Wrapper %q is not bare, hooks, or settings", s.Hooks.Wrapper)
		}
	default:
		return fmt.Errorf("Hooks.Format %q is not a known dialect", s.Hooks.Format)
	}
	switch s.Hooks.TimeoutUnit {
	case TimeoutSeconds, TimeoutMilliseconds:
	default:
		return fmt.Errorf("Hooks.TimeoutUnit %q is not %q or empty (seconds)", s.Hooks.TimeoutUnit, TimeoutMilliseconds)
	}
	switch s.Hooks.MatcherShape {
	case MatcherPlain, MatcherToolName:
	default:
		return fmt.Errorf("Hooks.MatcherShape %q is not %q or empty (plain string)", s.Hooks.MatcherShape, MatcherToolName)
	}
	for canonical, target := range s.Hooks.EventMap {
		if !slices.Contains(CanonicalEvents, canonical) {
			return fmt.Errorf("Hooks.EventMap key %q is not a canonical event", canonical)
		}
		if !slices.Contains(s.Hooks.Events, target) {
			return fmt.Errorf("Hooks.EventMap maps %q to %q, which is not in Hooks.Events", canonical, target)
		}
	}
	switch s.MCP.Format {
	case DialectNone:
		if s.MCP.File != "" {
			return fmt.Errorf("MCP.File %q set without an MCP format", s.MCP.File)
		}
		if s.MCP.TableKey != "" {
			return fmt.Errorf("MCP.TableKey %q set without an MCP format", s.MCP.TableKey)
		}
		if s.MCP.EnvField != "" {
			return fmt.Errorf("MCP.EnvField %q set without an MCP format", s.MCP.EnvField)
		}
	case DialectJSON, DialectJSONC, DialectTOML, DialectYAML:
		if s.MCP.File != "" && s.MCP.TableKey == "" {
			return fmt.Errorf("MCP.TableKey is required with MCP.File %q", s.MCP.File)
		}
	default:
		return fmt.Errorf("MCP.Format %q is not a known dialect", s.MCP.Format)
	}
	return nil
}

// HookEvent resolves a canonical event to the target's effective name. The
// second result reports whether the target supports the event at all;
// unsupported events are dropped with a warning during translation.
func (s Spec) HookEvent(canonical string) (string, bool) {
	name := s.hookEventName(canonical)
	return name, slices.Contains(s.Hooks.Events, name)
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
	if mapped, ok := s.Hooks.EventMap[canonical]; ok {
		return mapped
	}
	return transformCase(canonical, s.Hooks.EventCase)
}

func transformCase(name string, style EventCase) string {
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
