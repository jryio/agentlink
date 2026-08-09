package link

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func (e *Engine) checkMCP(server config.MCPServer) PairReport {
	report := PairReport{ID: server.ID, Name: mcpName(server), Files: 2}
	ids := server.PeerIDs()
	left, right := server.Peers[ids[0]], server.Peers[ids[1]]
	leftSpec, _ := agent.Get(ids[0])
	rightSpec, _ := agent.Get(ids[1])
	if !e.fs.Available(left.Config.Source) || !e.fs.Available(right.Config.Source) {
		report.Skipped = true
		report.Reason = "optional source unavailable"
		return report
	}
	leftRoot, err := e.fs.Root(left.Config.Source)
	if err != nil {
		return mcpErrorReport(server, err)
	}
	rightRoot, err := e.fs.Root(right.Config.Source)
	if err != nil {
		return mcpErrorReport(server, err)
	}
	leftConfig, leftErr := readStructured(leftRoot, left.Config.Path, e.doc.Config.MaxFileSize(), leftSpec.MCPFormat)
	rightConfig, rightErr := readStructured(rightRoot, right.Config.Path, e.doc.Config.MaxFileSize(), rightSpec.MCPFormat)
	leftMissing := errors.Is(leftErr, os.ErrNotExist)
	rightMissing := errors.Is(rightErr, os.ErrNotExist)
	switch {
	case leftErr != nil && !leftMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateError, "", leftRoot, rightRoot, "read "+ids[0]+" MCP configuration: "+leftErr.Error()))
		return report
	case rightErr != nil && !rightMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateError, "", leftRoot, rightRoot, "read "+ids[1]+" MCP configuration: "+rightErr.Error()))
		return report
	case leftMissing && rightMissing:
		if !server.Optional {
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, "", leftRoot, rightRoot, "both MCP configuration files are missing"))
		}
		return report
	case leftMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[0], leftRoot, rightRoot, ids[0]+" MCP configuration is missing"))
		return report
	case rightMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[1], leftRoot, rightRoot, ids[1]+" MCP configuration is missing"))
		return report
	}

	leftEntry, leftFound := nestedMap(leftConfig, leftSpec.MCPTableKey, left.Server)
	rightEntry, rightFound := nestedMap(rightConfig, rightSpec.MCPTableKey, right.Server)
	// Goose extensions mix non-MCP types into the table; only stdio and
	// streamable_http entries are MCP servers.
	if leftFound && leftSpec.MCPTableKey == "extensions" && !gooseMCPServer(leftEntry) {
		leftFound = false
	}
	if rightFound && rightSpec.MCPTableKey == "extensions" && !gooseMCPServer(rightEntry) {
		rightFound = false
	}
	switch {
	case !leftFound && !rightFound:
		if !server.Optional {
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, "", leftRoot, rightRoot, "MCP server entry is missing from both tools"))
		}
		return report
	case !leftFound:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[0], leftRoot, rightRoot, ids[0]+" MCP server entry is missing"))
	case !rightFound:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[1], leftRoot, rightRoot, ids[1]+" MCP server entry is missing"))
	}
	if !leftFound || !rightFound {
		return report
	}

	if comparePublic(server) && !reflect.DeepEqual(publicMCP(leftEntry), publicMCP(rightEntry)) {
		report.Findings = append(report.Findings, mcpFinding(server, StateDifferent, "", leftRoot, rightRoot, "public command/args/transport/url fields differ (secret values were not compared)"))
	}
	leftEnv := envNames(leftEntry, leftSpec)
	rightEnv := envNames(rightEntry, rightSpec)
	for _, name := range server.RequiredEnv {
		_, leftHas := leftEnv[name]
		_, rightHas := rightEnv[name]
		switch {
		case !leftHas && !rightHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, "", leftRoot, rightRoot, fmt.Sprintf("required environment key %s is missing from both tools", name)))
		case !leftHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[0], leftRoot, rightRoot, fmt.Sprintf("required environment key %s is missing from %s", name, ids[0])))
		case !rightHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissing, ids[1], leftRoot, rightRoot, fmt.Sprintf("required environment key %s is missing from %s", name, ids[1])))
		}
	}
	return report
}

func gooseMCPServer(entry map[string]any) bool {
	extensionType, _ := entry["type"].(string)
	return extensionType == "stdio" || extensionType == "streamable_http"
}

// envNames collects the compared environment key NAMES (never values): the
// spec's env map field plus any env_vars passthrough list (codex).
func envNames(entry map[string]any, spec agent.Spec) map[string]struct{} {
	names := make(map[string]struct{})
	if spec.MCPEnvField != "" {
		if env, ok := asMap(entry[spec.MCPEnvField]); ok {
			for name := range env {
				names[name] = struct{}{}
			}
		}
	}
	if forwarded, ok := entry["env_vars"].([]any); ok {
		for _, value := range forwarded {
			if name, ok := value.(string); ok {
				names[name] = struct{}{}
			}
		}
	}
	return names
}

func readStructured(root *safefs.Root, filePath string, maxSize int64, format agent.MCPFormat) (map[string]any, error) {
	data, _, err := root.ReadFile(filePath, maxSize)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	switch format {
	case agent.MCPFormatJSON, agent.MCPFormatJSONC:
		if format == agent.MCPFormatJSONC {
			data = stripJSONC(data)
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&result); err != nil {
			return nil, errors.New("decode JSON: invalid configuration")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("decode JSON: multiple top-level values")
			}
			return nil, errors.New("decode JSON: invalid trailing content")
		}
	case agent.MCPFormatTOML:
		if _, err := toml.Decode(string(data), &result); err != nil {
			return nil, errors.New("decode TOML: invalid configuration")
		}
	case agent.MCPFormatYAML:
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&result); err != nil {
			return nil, errors.New("decode YAML: invalid configuration")
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, errors.New("decode YAML: multiple documents")
			}
			return nil, errors.New("decode YAML: invalid trailing content")
		}
	default:
		return nil, fmt.Errorf("unsupported configuration format %q for %s", format, filePath)
	}
	return result, nil
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

// nestedMap walks a dot-separated table key (e.g. "amp.mcpServers") and
// returns the named server entry.
func nestedMap(values map[string]any, table, name string) (map[string]any, bool) {
	current := values
	for _, segment := range strings.Split(table, ".") {
		next, ok := asMap(current[segment])
		if !ok {
			return nil, false
		}
		current = next
	}
	return asMap(current[name])
}

func asMap(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

// publicMCP reduces a server entry to the compared public fields, normalized
// across agent dialects: single-array commands split into command+args
// (opencode), transport aliases converge (local→stdio, streamable variants
// →http), and a missing transport is inferred from command/url presence.
func publicMCP(entry map[string]any) map[string]any {
	result := make(map[string]any, 4)
	command, args := commandParts(entry)
	if command != "" {
		result["command"] = command
	}
	if len(args) > 0 {
		result["args"] = args
	}
	if transport := transportName(entry); transport != "" {
		result["transport"] = transport
	}
	if url, ok := entry["url"]; ok {
		result["url"] = url
	}
	return result
}

// commandParts accepts both "command" + "args" pairs and opencode's
// single-array command form.
func commandParts(entry map[string]any) (string, []any) {
	if list, ok := entry["command"].([]any); ok {
		if len(list) == 0 {
			return "", nil
		}
		head, _ := list[0].(string)
		return head, list[1:]
	}
	command, _ := entry["command"].(string)
	args, _ := entry["args"].([]any)
	return command, args
}

// transportName normalizes the transport discriminator across agents.
func transportName(entry map[string]any) string {
	raw, _ := entry["transport"].(string)
	if raw == "" {
		raw, _ = entry["type"].(string)
	}
	switch raw {
	case "":
		// Infer: a command means stdio, a bare URL means http.
		if _, hasCommand := entry["command"]; hasCommand {
			return "stdio"
		}
		if _, hasURL := entry["url"]; hasURL {
			return "http"
		}
		return ""
	case "local":
		return "stdio"
	case "streamable-http", "streamable_http":
		return "http"
	}
	return raw
}

func comparePublic(server config.MCPServer) bool {
	return server.ComparePublic == nil || *server.ComparePublic
}

func mcpRelative(server config.MCPServer) string {
	ids := server.PeerIDs()
	return server.Peers[ids[0]].Server + " ↔ " + server.Peers[ids[1]].Server
}

func mcpFinding(server config.MCPServer, state State, peer string, leftRoot, rightRoot *safefs.Root, detail string) Finding {
	ids := server.PeerIDs()
	return Finding{
		Pair:     server.ID,
		Relative: mcpRelative(server),
		State:    state,
		Peer:     peer,
		Paths: map[string]string{
			ids[0]: leftRoot.Abs(server.Peers[ids[0]].Config.Path),
			ids[1]: rightRoot.Abs(server.Peers[ids[1]].Config.Path),
		},
		Detail: detail,
	}
}

func mcpErrorReport(server config.MCPServer, err error) PairReport {
	return PairReport{
		ID:   server.ID,
		Name: mcpName(server),
		Findings: []Finding{{
			Pair: server.ID, State: StateError, Relative: mcpRelative(server), Detail: err.Error(),
		}},
	}
}

func mcpName(server config.MCPServer) string {
	if server.Name != "" {
		return server.Name
	}
	return server.ID
}
