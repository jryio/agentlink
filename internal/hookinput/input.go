// Package hookinput extracts changed paths from editor and agent hook payloads.
package hookinput

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
)

var patchPrefixes = [...]string{
	"*** Add File: ",
	"*** Update File: ",
	"*** Delete File: ",
	"*** Move to: ",
}

const maxInputSize = 8 << 20

// Parse accepts JSON hook payloads or one-path-per-line input.
func Parse(ctx context.Context, reader io.Reader) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if closer, ok := reader.(io.ReadCloser); ok {
		stopClose := context.AfterFunc(ctx, func() { _ = closer.Close() })
		defer stopClose()
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxInputSize+1))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	if len(data) > maxInputSize {
		return nil, fmt.Errorf("changed-path input exceeds %d bytes", maxInputSize)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var payload any
	if json.Unmarshal(data, &payload) == nil {
		paths := make(map[string]struct{})
		visit(payload, paths)
		result := keys(paths)
		slices.Sort(result)
		return result, nil
	}
	paths := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			paths[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	result := keys(paths)
	slices.Sort(result)
	return result, nil
}

func visit(value any, paths map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch key {
			case "file_path", "filePath", "path":
				if text, ok := child.(string); ok && text != "" {
					paths[text] = struct{}{}
				}
			case "patch", "input":
				if text, ok := child.(string); ok {
					patchPaths(text, paths)
				}
			default:
				visit(child, paths)
			}
		}
	case []any:
		for _, child := range typed {
			visit(child, paths)
		}
	}
}

func patchPaths(text string, paths map[string]struct{}) {
	for _, line := range strings.Split(text, "\n") {
		for _, prefix := range patchPrefixes {
			if strings.HasPrefix(line, prefix) {
				paths[strings.TrimSpace(strings.TrimPrefix(line, prefix))] = struct{}{}
				break
			}
		}
	}
}

func keys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}
