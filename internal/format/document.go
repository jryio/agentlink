package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/jryio/agentlink/internal/agent"
)

// DecodeDocument parses one declarative document in the given dialect into a
// generic map. JSONC input is stripped of comments and trailing commas
// first; JSON numbers decode as json.Number so timeout rescaling stays exact.
func DecodeDocument(dialect agent.Dialect, data []byte) (map[string]any, error) {
	result := make(map[string]any)
	switch dialect {
	case agent.DialectJSON, agent.DialectJSONC:
		if dialect == agent.DialectJSONC {
			data = stripJSONC(data)
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&result); err != nil {
			return nil, fmt.Errorf("decode JSON: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("decode JSON: multiple top-level values")
			}
			return nil, fmt.Errorf("decode JSON: invalid trailing content: %w", err)
		}
	case agent.DialectTOML:
		if _, err := toml.Decode(string(data), &result); err != nil {
			return nil, fmt.Errorf("decode TOML: %w", err)
		}
	case agent.DialectYAML:
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&result); err != nil {
			return nil, fmt.Errorf("decode YAML: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("decode YAML: multiple documents")
			}
			return nil, fmt.Errorf("decode YAML: invalid trailing content: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported document dialect %q", dialect)
	}
	return result, nil
}

// EncodeDocument renders a document in the given dialect. JSON output is
// deterministic: sorted keys, two-space indent, trailing newline.
func EncodeDocument(dialect agent.Dialect, doc map[string]any) ([]byte, error) {
	switch dialect {
	case agent.DialectJSON, agent.DialectJSONC:
		return encodeJSON(doc)
	case agent.DialectTOML:
		var buffer bytes.Buffer
		if err := toml.NewEncoder(&buffer).Encode(doc); err != nil {
			return nil, fmt.Errorf("encode TOML: %w", err)
		}
		return buffer.Bytes(), nil
	case agent.DialectYAML:
		out, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("encode YAML: %w", err)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported document dialect %q", dialect)
}

// encodeJSON emits a deterministic, sorted, indented JSON document.
func encodeJSON(value any) ([]byte, error) {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return append(out, '\n'), nil
}

// stripJSONC removes // and /* */ comments and trailing commas while
// preserving string contents.
func stripJSONC(data []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case inString:
			out.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
		case c == '"':
			inString = true
			out.WriteByte(c)
		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out.WriteByte('\n')
			}
		case c == '/' && i+1 < len(data) && data[i+1] == '*':
			for i+1 < len(data) && (data[i] != '*' || data[i+1] != '/') {
				i++
			}
			i++
		case c == ',':
			// Drop a comma whose next significant byte closes a container.
			j := i + 1
			for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\n' || data[j] == '\r') {
				j++
			}
			if j < len(data) && (data[j] == '}' || data[j] == ']') {
				continue
			}
			out.WriteByte(c)
		default:
			out.WriteByte(c)
		}
	}
	return out.Bytes()
}
