package link

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/format"
)

// Params carries the two peer agent specs a normalization runs between. Self
// is the side being normalized; Other participates in pairwise rules (the
// skill frontmatter intersection).
type Params struct {
	Self  agent.Spec
	Other agent.Spec
}

func normalize(name config.Normalizer, data []byte, p Params) ([]byte, error) {
	if name == "" {
		name = config.NormalizerText
	}
	if name != config.NormalizerExact && bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("normalizer %q cannot compare binary content", name)
	}
	switch name {
	case config.NormalizerPresence:
		return nil, nil
	case config.NormalizerExact:
		return bytes.Clone(data), nil
	case config.NormalizerText:
		return format.NormalizeText(data), nil
	case config.NormalizerInstructions:
		// Both peers' names must converge in one longest-first pass. Applying
		// them separately lets a short name consume a longer counterpart name.
		return format.ConvergeDisplayNames(format.NormalizeText(data), p.Self, p.Other), nil
	case config.NormalizerSkill:
		canonical, err := formatterCanonicalize(name, p.Self, data)
		if err != nil {
			return nil, err
		}
		frontmatter, body, ok := format.Split(canonical)
		if !ok {
			return canonical, nil
		}
		values, err := format.Parse(frontmatter)
		if err != nil {
			return nil, err
		}
		kept := make(map[string]any, len(values))
		for key, value := range values {
			if !portableSkillKey(key, p) {
				continue
			}
			kept[key] = value
		}
		return format.Canonical(kept, body)
	case config.NormalizerHook:
		return format.CanonicalHookDocument(p.Self, p.Other, data)
	default:
		return nil, fmt.Errorf("unknown normalizer %q", name)
	}
}

// formatterCanonicalize runs the skill formatter's canonicalization.
func formatterCanonicalize(name config.Normalizer, self agent.Spec, data []byte) ([]byte, error) {
	formatter, ok := format.For(string(name))
	if !ok {
		return nil, fmt.Errorf("no formatter for normalizer %q", name)
	}
	return formatter.Canonicalize(self, data)
}

// portableSkillKey reports whether a canonical frontmatter key is part of the
// universal compare set and understood by both peers. Skills.Keys hold each
// agent's on-disk spelling, so membership is tested after mapping both sets
// to canonical names.
func portableSkillKey(key string, p Params) bool {
	if !skillKeyIn(key, agent.U) {
		return false
	}
	return skillKeyIn(key, canonicalSkillKeys(p.Self)) && skillKeyIn(key, canonicalSkillKeys(p.Other))
}

func canonicalSkillKeys(spec agent.Spec) []string {
	keys := make([]string, 0, len(spec.Skills.Keys))
	for _, key := range spec.Skills.Keys {
		canonical := key
		for canonicalName, targetName := range spec.Skills.Renames {
			if strings.EqualFold(targetName, key) {
				canonical = canonicalName
				break
			}
		}
		keys = append(keys, canonical)
	}
	return keys
}

func skillKeyIn(key string, keys []string) bool {
	for _, candidate := range keys {
		if candidate == key {
			return true
		}
	}
	return false
}
