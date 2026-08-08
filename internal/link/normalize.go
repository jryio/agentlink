package link

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

var headingPattern = regexp.MustCompile(`(?im)^# .*?(Claude(?: Code)?|Codex).*?$`)

var ignoredSkillFrontmatter = map[string]struct{}{
	"allowed-tools":     {},
	"argument-hint":     {},
	"argument-metadata": {},
	"model":             {},
	"tools":             {},
	"user-invocable":    {},
}

func normalize(name string, data []byte) ([]byte, error) {
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
		return normalizeText(data), nil
	case "instructions":
		text := string(normalizeText(data))
		text = strings.ReplaceAll(text, "Claude Code and Codex", "Claude/Codex")
		text = strings.ReplaceAll(text, "Claude and Codex", "Claude/Codex")
		text = strings.ReplaceAll(text, "Claude Code", "Claude")
		text = headingPattern.ReplaceAllString(text, "# Agent instructions")
		return []byte(text), nil
	case "skill":
		frontmatter, body, ok := splitFrontmatter(data)
		if !ok {
			return normalizeText(data), nil
		}
		var values map[string]any
		if err := yaml.Unmarshal(frontmatter, &values); err != nil {
			return nil, fmt.Errorf("parse skill frontmatter: %w", err)
		}
		for key := range values {
			if _, ignored := ignoredSkillFrontmatter[strings.ToLower(key)]; ignored {
				delete(values, key)
			}
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var canonical strings.Builder
		canonical.WriteString("---\n")
		for _, key := range keys {
			encoded, err := yaml.Marshal(map[string]any{key: values[key]})
			if err != nil {
				return nil, fmt.Errorf("normalize skill frontmatter key %q: %w", key, err)
			}
			canonical.Write(encoded)
		}
		canonical.WriteString("---\n")
		canonical.Write(body)
		return normalizeText([]byte(canonical.String())), nil
	case "hook":
		text := string(normalizeText(data))
		replacer := strings.NewReplacer(
			"remind-claude", "remind-agent",
			"remind-codex", "remind-agent",
			"--agent claude", "--agent agent",
			"--agent codex", "--agent agent",
		)
		return []byte(replacer.Replace(text)), nil
	default:
		return nil, fmt.Errorf("unknown normalizer %q", name)
	}
}

func normalizeText(data []byte) []byte {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.TrimSpace(strings.Join(lines, "\n"))
	if text == "" {
		return nil
	}
	return []byte(text + "\n")
}

func splitFrontmatter(data []byte) ([]byte, []byte, bool) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, nil, false
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return nil, nil, false
	}
	end += 4
	return []byte(text[4:end]), []byte(text[end+5:]), true
}
