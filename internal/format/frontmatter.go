package format

import (
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Split separates YAML frontmatter from a markdown body. ok is false when
// the document has no frontmatter block.
func Split(data []byte) (frontmatter, body []byte, ok bool) {
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

// Parse decodes a frontmatter block into a key-value map.
func Parse(frontmatter []byte) (map[string]any, error) {
	values := make(map[string]any)
	if err := yaml.Unmarshal(frontmatter, &values); err != nil {
		return nil, fmt.Errorf("parse skill frontmatter: %w", err)
	}
	return values, nil
}

// Canonical emits frontmatter values as a sorted, deterministic YAML block
// followed by the body, then text-normalizes the result.
func Canonical(values map[string]any, body []byte) ([]byte, error) {
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
	return NormalizeText([]byte(canonical.String())), nil
}
