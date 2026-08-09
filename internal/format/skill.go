package format

import (
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
		if renamed, ok := target.SkillRenames[lower]; ok {
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
	for _, supported := range target.SkillKeys {
		if supported == key {
			return true
		}
	}
	return false
}

func init() {
	Register(skillFormatter{})
}
