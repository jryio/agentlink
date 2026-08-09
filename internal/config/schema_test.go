package config

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/jryio/agentlink/internal/agent"
)

// TestSchemaMatchesRegistry keeps the embedded editor schema's peer-key enums
// in lockstep with the compiled agent registry so the schema cannot rot when
// agents are added or removed.
func TestSchemaMatchesRegistry(t *testing.T) {
	t.Parallel()

	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				PropertyNames struct {
					Enum []string `json:"enum"`
				} `json:"propertyNames"`
			} `json:"properties"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(Schema(), &schema); err != nil {
		t.Fatalf("decode embedded schema: %v", err)
	}
	specs := agent.All()
	want := make([]string, 0, len(specs))
	for _, spec := range specs {
		want = append(want, spec.ID)
	}
	for _, def := range []string{"pair", "mcpServer"} {
		got := schema.Defs[def].Properties["peers"].PropertyNames.Enum
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("$defs.%s.peers enum mismatch (-want +got):\n%s", def, diff)
		}
	}
}
