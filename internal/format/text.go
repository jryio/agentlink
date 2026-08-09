package format

import "strings"

// NormalizeText canonicalizes line endings, trailing whitespace, and the
// final newline. Empty input normalizes to nil.
func NormalizeText(data []byte) []byte {
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
