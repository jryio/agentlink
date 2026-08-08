package link

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"

	"github.com/jryio/agentlink/internal/config"
	"github.com/jryio/agentlink/internal/safefs"
)

func (e *Engine) checkMCP(server config.MCPServer) PairReport {
	report := PairReport{ID: server.ID, Name: mcpName(server), Files: 2}
	if !e.fs.Available(server.Claude.Config.Source) || !e.fs.Available(server.Codex.Config.Source) {
		report.Skipped = true
		report.Reason = "optional source unavailable"
		return report
	}
	claudeRoot, err := e.fs.Root(server.Claude.Config.Source)
	if err != nil {
		return mcpErrorReport(server, err)
	}
	codexRoot, err := e.fs.Root(server.Codex.Config.Source)
	if err != nil {
		return mcpErrorReport(server, err)
	}
	claudeConfig, claudeErr := readStructured(claudeRoot, server.Claude.Config.Path, e.doc.Config.MaxFileSize())
	codexConfig, codexErr := readStructured(codexRoot, server.Codex.Config.Path, e.doc.Config.MaxFileSize())
	claudeMissing := errors.Is(claudeErr, os.ErrNotExist)
	codexMissing := errors.Is(codexErr, os.ErrNotExist)
	switch {
	case claudeErr != nil && !claudeMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateError, claudeRoot, codexRoot, "read Claude MCP configuration: "+claudeErr.Error()))
		return report
	case codexErr != nil && !codexMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateError, claudeRoot, codexRoot, "read Codex MCP configuration: "+codexErr.Error()))
		return report
	case claudeMissing && codexMissing:
		if !server.Optional {
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, claudeRoot, codexRoot, "both MCP configuration files are missing"))
		}
		return report
	case claudeMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissingClaude, claudeRoot, codexRoot, "Claude MCP configuration is missing"))
		return report
	case codexMissing:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissingCodex, claudeRoot, codexRoot, "Codex MCP configuration is missing"))
		return report
	}

	claudeEntry, claudeFound := nestedMap(claudeConfig, "mcpServers", server.Claude.Server)
	codexEntry, codexFound := nestedMap(codexConfig, "mcp_servers", server.Codex.Server)
	switch {
	case !claudeFound && !codexFound:
		if !server.Optional {
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, claudeRoot, codexRoot, "MCP server entry is missing from both tools"))
		}
		return report
	case !claudeFound:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissingClaude, claudeRoot, codexRoot, "Claude MCP server entry is missing"))
	case !codexFound:
		report.Findings = append(report.Findings, mcpFinding(server, StateMissingCodex, claudeRoot, codexRoot, "Codex MCP server entry is missing"))
	}
	if !claudeFound || !codexFound {
		return report
	}

	if comparePublic(server) && !reflect.DeepEqual(publicMCP(claudeEntry), publicMCP(codexEntry)) {
		report.Findings = append(report.Findings, mcpFinding(server, StateDifferent, claudeRoot, codexRoot, "public command/args/transport/url fields differ (secret values were not compared)"))
	}
	claudeEnv, _ := asMap(claudeEntry["env"])
	codexEnv, _ := asMap(codexEntry["env"])
	for _, name := range server.RequiredEnv {
		_, claudeHas := claudeEnv[name]
		_, codexHas := codexEnv[name]
		switch {
		case !claudeHas && !codexHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingBoth, claudeRoot, codexRoot, fmt.Sprintf("required environment key %s is missing from both tools", name)))
		case !claudeHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingClaude, claudeRoot, codexRoot, fmt.Sprintf("required environment key %s is missing from Claude", name)))
		case !codexHas:
			report.Findings = append(report.Findings, mcpFinding(server, StateMissingCodex, claudeRoot, codexRoot, fmt.Sprintf("required environment key %s is missing from Codex", name)))
		}
	}
	return report
}

func readStructured(root *safefs.Root, filePath string, maxSize int64) (map[string]any, error) {
	data, _, err := root.ReadFile(filePath, maxSize)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	switch strings.ToLower(path.Ext(filePath)) {
	case ".json":
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
	case ".toml":
		if _, err := toml.Decode(string(data), &result); err != nil {
			return nil, errors.New("decode TOML: invalid configuration")
		}
	case ".yaml", ".yml":
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
		return nil, fmt.Errorf("unsupported configuration extension %q", path.Ext(filePath))
	}
	return result, nil
}

func nestedMap(values map[string]any, table, name string) (map[string]any, bool) {
	parent, ok := asMap(values[table])
	if !ok {
		return nil, false
	}
	return asMap(parent[name])
}

func asMap(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

func publicMCP(entry map[string]any) map[string]any {
	result := make(map[string]any, 4)
	for _, key := range []string{"command", "args", "transport", "url"} {
		if value, ok := entry[key]; ok {
			result[key] = value
		}
	}
	return result
}

func comparePublic(server config.MCPServer) bool {
	return server.ComparePublic == nil || *server.ComparePublic
}

func mcpFinding(server config.MCPServer, state State, claudeRoot, codexRoot *safefs.Root, detail string) Finding {
	return Finding{
		Pair:     server.ID,
		Relative: server.Claude.Server + " ↔ " + server.Codex.Server,
		State:    state,
		Claude:   claudeRoot.Abs(server.Claude.Config.Path),
		Codex:    codexRoot.Abs(server.Codex.Config.Path),
		Detail:   detail,
	}
}

func mcpErrorReport(server config.MCPServer, err error) PairReport {
	return PairReport{
		ID:   server.ID,
		Name: mcpName(server),
		Findings: []Finding{{
			Pair: server.ID, State: StateError, Relative: server.Claude.Server + " ↔ " + server.Codex.Server, Detail: err.Error(),
		}},
	}
}

func mcpName(server config.MCPServer) string {
	if server.Name != "" {
		return server.Name
	}
	return server.ID
}
