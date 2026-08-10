package format

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
)

// skillFormatter rewrites canonical SKILL.md frontmatter for a target: keys
// the target does not understand are dropped (name/description always stay)
// and canonical keys are renamed to the target's spelling.
type skillFormatter struct{}

func (skillFormatter) Kind() string { return "skill" }

func (skillFormatter) Format(canonical, _ []byte, target agent.Spec) ([]byte, []string, error) {
	frontmatter, body, ok := Split(canonical)
	if !ok {
		return NormalizeText(canonical), nil, nil
	}
	values, err := Parse(frontmatter)
	if err != nil {
		return nil, nil, err
	}
	kept := make(map[string]any, len(values))
	var warnings []string
	for key, value := range values {
		lower := strings.ToLower(key)
		out := lower
		if renamed, ok := target.Skills.Renames[lower]; ok {
			out = renamed
		}
		if lower != "name" && lower != "description" && !skillKeySupported(target, out) {
			warnings = append(warnings, "drop unsupported frontmatter key "+key+" for "+target.ID)
			continue
		}
		kept[out] = value
	}
	out, err := Canonical(kept, body)
	if err != nil {
		return nil, nil, err
	}
	return out, warnings, nil
}

func skillKeySupported(target agent.Spec, key string) bool {
	for _, supported := range target.Skills.Keys {
		if supported == key {
			return true
		}
	}
	return false
}

// Canonicalize maps a SKILL.md in self's native spelling back to canonical
// frontmatter keys. Every key is kept — deciding which keys two peers share
// is comparison's job, not translation's.
func (skillFormatter) Canonicalize(self agent.Spec, data []byte) ([]byte, error) {
	frontmatter, body, ok := Split(data)
	if !ok {
		return NormalizeText(data), nil
	}
	values, err := Parse(frontmatter)
	if err != nil {
		return nil, err
	}
	inverse := make(map[string]string, len(self.Skills.Renames))
	for canonical, target := range self.Skills.Renames {
		inverse[strings.ToLower(target)] = canonical
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	kept := make(map[string]any, len(values))
	origins := make(map[string]string, len(values))
	usesTargetKey := make(map[string]bool, len(values))
	for _, key := range keys {
		canonical := strings.ToLower(key)
		targetKey := false
		if renamed, ok := inverse[canonical]; ok {
			canonical = renamed
			targetKey = true
		}
		if origin, exists := origins[canonical]; exists {
			switch {
			case targetKey && !usesTargetKey[canonical]:
				// The target's preferred spelling wins over an accepted
				// legacy spelling (for example Cursor paths over globs).
			case !targetKey && usesTargetKey[canonical]:
				continue
			default:
				return nil, fmt.Errorf("skill frontmatter keys %q and %q both map to %q", origin, key, canonical)
			}
		}
		kept[canonical] = values[key]
		origins[canonical] = key
		usesTargetKey[canonical] = targetKey
	}
	return Canonical(kept, body)
}

func init() {
	Register(skillFormatter{})
}
