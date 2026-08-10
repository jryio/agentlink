package agent

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// designMatrixEntry mirrors one agent's record in
// docs/research/design-matrix.json. The "agentlink" block is the
// machine-checkable contract between the research and this registry: editing
// a Spec without updating the matrix (or vice versa) fails this test.
type designMatrixEntry struct {
	Agent     string `json:"agent"`
	ConfigDir string `json:"config_dir"`
	Hooks     struct {
		Supported bool     `json:"supported"`
		Events    []string `json:"events"`
	} `json:"hooks"`
	Contract struct {
		HooksFormat     string            `json:"hooks_format"`
		MCPFormat       string            `json:"mcp_format"`
		HookEventMap    map[string]string `json:"hook_event_map"`
		HookTimeoutUnit string            `json:"hook_timeout_unit"`
	} `json:"agentlink"`
}

func loadDesignMatrix(t *testing.T) map[string]designMatrixEntry {
	t.Helper()
	data, err := os.ReadFile("../../docs/research/design-matrix.json")
	if err != nil {
		t.Fatalf("read design matrix: %v", err)
	}
	var entries []designMatrixEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse design matrix: %v", err)
	}
	byAgent := make(map[string]designMatrixEntry, len(entries))
	for _, entry := range entries {
		byAgent[entry.Agent] = entry
	}
	return byAgent
}

// TestRegistryMatchesDesignMatrix locks the registry to the sourced research
// in docs/research: dialects, hook event maps, timeout units, and the event
// vocabulary must agree, so per-agent facts cannot drift between docs and
// code. The canonical "agents" hub is ours and has no matrix entry.
func TestRegistryMatchesDesignMatrix(t *testing.T) {
	matrix := loadDesignMatrix(t)
	for _, spec := range All() {
		if spec.ID == "agents" {
			continue
		}
		entry, ok := matrix[spec.ID]
		if !ok {
			t.Errorf("%s: no design-matrix entry", spec.ID)
			continue
		}
		t.Run(spec.ID, func(t *testing.T) {
			if got := string(spec.Hooks.Format); got != entry.Contract.HooksFormat {
				t.Errorf("Hooks.Format = %q, matrix contract says %q", got, entry.Contract.HooksFormat)
			}
			if got := string(spec.MCP.Format); got != entry.Contract.MCPFormat {
				t.Errorf("MCP.Format = %q, matrix contract says %q", got, entry.Contract.MCPFormat)
			}
			if got := string(spec.Hooks.TimeoutUnit); got != entry.Contract.HookTimeoutUnit {
				t.Errorf("Hooks.TimeoutUnit = %q, matrix contract says %q", got, entry.Contract.HookTimeoutUnit)
			}
			if !mapsEqual(spec.Hooks.EventMap, entry.Contract.HookEventMap) {
				t.Errorf("Hooks.EventMap = %v, matrix contract says %v", spec.Hooks.EventMap, entry.Contract.HookEventMap)
			}
			if !entry.Hooks.Supported {
				return
			}
			for _, event := range spec.Hooks.Events {
				if !slices.Contains(entry.Hooks.Events, event) {
					t.Errorf("Hooks.Events lists %q, absent from the matrix event vocabulary", event)
				}
			}
			for _, target := range spec.Hooks.EventMap {
				if !slices.Contains(entry.Hooks.Events, target) {
					t.Errorf("Hooks.EventMap target %q absent from the matrix event vocabulary", target)
				}
			}
		})
	}
}

// TestConfigDirMatchesDesignMatrix checks the project config dir where the
// matrix states one cleanly ("none" maps to an empty ConfigDir; prose
// annotations are skipped).
func TestConfigDirMatchesDesignMatrix(t *testing.T) {
	matrix := loadDesignMatrix(t)
	for _, spec := range All() {
		if spec.ID == "agents" {
			continue
		}
		entry, ok := matrix[spec.ID]
		if !ok {
			continue
		}
		dir := entry.ConfigDir
		if dir == "none" {
			dir = ""
		}
		if strings.ContainsAny(dir, " ()") {
			continue // prose annotation, not a single clean directory
		}
		if got := strings.TrimSuffix(spec.ConfigDir, "/"); got != strings.TrimSuffix(dir, "/") {
			t.Errorf("%s: ConfigDir = %q, matrix says %q", spec.ID, spec.ConfigDir, entry.ConfigDir)
		}
	}
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if other, ok := right[key]; !ok || other != value {
			return false
		}
	}
	return true
}
