package link

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/jryio/agentlink/internal/agent"
	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/format"
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
	leftConfig, leftErr := readStructured(leftRoot, left.Config.Path, e.doc.Config.MaxFileSize(), leftSpec.MCP.Format)
	rightConfig, rightErr := readStructured(rightRoot, right.Config.Path, e.doc.Config.MaxFileSize(), rightSpec.MCP.Format)
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

	leftEntry, leftFound := nestedMap(leftConfig, leftSpec.MCP.TableKey, left.Server)
	rightEntry, rightFound := nestedMap(rightConfig, rightSpec.MCP.TableKey, right.Server)
	// Goose extensions mix non-MCP types into the table; only stdio and
	// streamable_http entries are MCP servers.
	if leftFound && leftSpec.MCP.TableKey == "extensions" && !gooseMCPServer(leftEntry) {
		leftFound = false
	}
	if rightFound && rightSpec.MCP.TableKey == "extensions" && !gooseMCPServer(rightEntry) {
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
	if spec.MCP.EnvField != "" {
		if env, ok := asMap(entry[spec.MCP.EnvField]); ok {
			for name := range env {
				names[name] = struct{}{}
			}
		}
	}
	// Codex's env_vars passthrough list accepts plain names and an object
	// form ({name = "TOKEN", source = "local"}); both count as forwarded.
	// TOML decodes object arrays as []map[string]any.
	forwarded, _ := entry["env_vars"].([]any)
	if tables, ok := entry["env_vars"].([]map[string]any); ok {
		forwarded = make([]any, 0, len(tables))
		for _, table := range tables {
			forwarded = append(forwarded, table)
		}
	}
	for _, value := range forwarded {
		switch entry := value.(type) {
		case string:
			names[entry] = struct{}{}
		case map[string]any:
			if name, ok := entry["name"].(string); ok && name != "" {
				names[name] = struct{}{}
			}
		}
	}
	return names
}

// readStructured parses an MCP configuration file through the shared
// document codec. Decode errors are sanitized: decoder messages can echo
// file content (potentially secrets) into findings.
func readStructured(root *safefs.Root, filePath string, maxSize int64, dialect agent.Dialect) (map[string]any, error) {
	data, _, err := root.ReadFile(filePath, maxSize)
	if err != nil {
		return nil, err
	}
	result, err := format.DecodeDocument(dialect, data)
	if err != nil {
		label := map[agent.Dialect]string{
			agent.DialectJSON:  "JSON",
			agent.DialectJSONC: "JSON",
			agent.DialectTOML:  "TOML",
			agent.DialectYAML:  "YAML",
		}[dialect]
		if label == "" {
			return nil, err // dialect vocabulary error: carries no file content
		}
		return nil, errors.New("decode " + label + ": invalid configuration")
	}
	return result, nil
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
