package link

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/format"
)

// Params carries the two peer agent specs a normalization runs between. Self
// is the side being normalized; Other participates in pairwise rules (the
// skill frontmatter intersection).
type Params struct {
	Self  agent.Spec
	Other agent.Spec
}

func normalize(name string, data []byte, p Params) ([]byte, error) {
	if name == "" {
		name = "text"
	}
	if name != "exact" && bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("normalizer %q cannot compare binary content", name)
	}
	switch name {
	case "presence":
		return nil, nil
	case "exact":
		return bytes.Clone(data), nil
	case "text":
		return format.NormalizeText(data), nil
	case "instructions":
		text := string(format.NormalizeText(data))
		// Both peers' display names converge on the canonical "Agent" token,
		// in headings and prose, longest names first — so identical prose
		// mentioning either tool normalizes identically on both sides.
		names := displayNames(p.Self, p.Other)
		if len(names) > 0 {
			quoted := make([]string, 0, len(names))
			for _, display := range names {
				quoted = append(quoted, regexp.QuoteMeta(display))
			}
			heading := regexp.MustCompile(`(?im)^# .*?(` + strings.Join(quoted, "|") + `).*?$`)
			text = heading.ReplaceAllString(text, "# Agent instructions")
			pairs := make([]string, 0, 2*len(names))
			for _, display := range names {
				pairs = append(pairs, display, "Agent")
			}
			text = strings.NewReplacer(pairs...).Replace(text)
		}
		return []byte(text), nil
	case "skill":
		frontmatter, body, ok := format.Split(data)
		if !ok {
			return format.NormalizeText(data), nil
		}
		values, err := format.Parse(frontmatter)
		if err != nil {
			return nil, err
		}
		inverse := make(map[string]string, len(p.Self.SkillRenames))
		for canonical, target := range p.Self.SkillRenames {
			inverse[strings.ToLower(target)] = canonical
		}
		kept := make(map[string]any, len(values))
		for key, value := range values {
			lower := strings.ToLower(key)
			if canonical, ok := inverse[lower]; ok {
				lower = canonical
			}
			if !portableSkillKey(lower, p) {
				continue
			}
			kept[lower] = value
		}
		return format.Canonical(kept, body)
	case "hook":
		return format.CanonicalHookDocument(p.Self, p.Other, data)
	default:
		return nil, fmt.Errorf("unknown normalizer %q", name)
	}
}

// displayNames returns both specs' display names deduplicated, longest first
// (so "Claude Code" replaces before "Claude").
func displayNames(specs ...agent.Spec) []string {
	seen := make(map[string]bool)
	var names []string
	for _, spec := range specs {
		for _, name := range spec.DisplayNames {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	slices.SortStableFunc(names, func(a, b string) int { return len(b) - len(a) })
	return names
}

// portableSkillKey reports whether a canonical frontmatter key is part of the
// universal compare set and understood by both peers. SkillKeys hold each
// agent's on-disk spelling, so membership is tested after mapping both sets
// to canonical names.
func portableSkillKey(key string, p Params) bool {
	if !skillKeyIn(key, agent.U) {
		return false
	}
	return skillKeyIn(key, canonicalSkillKeys(p.Self)) && skillKeyIn(key, canonicalSkillKeys(p.Other))
}

func canonicalSkillKeys(spec agent.Spec) []string {
	keys := make([]string, 0, len(spec.SkillKeys))
	for _, key := range spec.SkillKeys {
		canonical := key
		for canonicalName, targetName := range spec.SkillRenames {
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
