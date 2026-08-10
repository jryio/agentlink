package format

import (
	"strings"
	"testing"

	"github.com/jryio/agentlink/internal/agent"
)

func TestDecodeDocumentJSONC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string // key expected in the decoded map
	}{
		{"line comment", "{// c\n\"a\": 1}", "a"},
		{"block comment", "{/* c */ \"a\": 1}", "a"},
		{"trailing comma", "{\"a\": 1,}", "a"},
		{"comment inside string stays", "{\"a\": \"x//y\"}", "a"},
		{"escaped quote inside string", "{\"a\": \"x\\\"//y\"}", "a"},
		{"trailing comma before bracket", "{\"a\": [1, 2,]}", "a"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			doc, err := DecodeDocument(agent.DialectJSONC, []byte(test.in))
			if err != nil {
				t.Fatalf("DecodeDocument(%s): %v", test.in, err)
			}
			if _, ok := doc[test.want]; !ok {
				t.Fatalf("DecodeDocument(%s) = %v, want key %q", test.in, doc, test.want)
			}
		})
	}

	t.Run("strict JSON still rejects comments", func(t *testing.T) {
		t.Parallel()
		if _, err := DecodeDocument(agent.DialectJSON, []byte("{// c\n\"a\": 1}")); err == nil {
			t.Fatal("DecodeDocument(strict JSON) accepted a comment")
		}
	})

	t.Run("string with slashes survives stripping", func(t *testing.T) {
		t.Parallel()
		doc, err := DecodeDocument(agent.DialectJSONC, []byte("{\"url\": \"https://example.com//path\"}"))
		if err != nil {
			t.Fatalf("DecodeDocument(): %v", err)
		}
		if doc["url"] != "https://example.com//path" {
			t.Fatalf("url = %v, want comment-like content preserved", doc["url"])
		}
	})
}

func TestEncodeDocumentRoundTripsDialects(t *testing.T) {
	t.Parallel()

	doc := map[string]any{"a": []any{"x", "y"}, "b": map[string]any{"c": "d"}}
	for _, dialect := range []agent.Dialect{agent.DialectJSON, agent.DialectTOML, agent.DialectYAML} {
		t.Run(string(dialect), func(t *testing.T) {
			t.Parallel()
			encoded, err := EncodeDocument(dialect, doc)
			if err != nil {
				t.Fatalf("EncodeDocument(): %v", err)
			}
			decoded, err := DecodeDocument(dialect, encoded)
			if err != nil {
				t.Fatalf("DecodeDocument(): %v\n%s", err, encoded)
			}
			first, _ := decoded["a"].([]any)
			if len(first) != 2 || first[0] != "x" {
				t.Fatalf("round trip lost array: %v", decoded)
			}
			nested, _ := decoded["b"].(map[string]any)
			if nested["c"] != "d" {
				t.Fatalf("round trip lost nested map: %v", decoded)
			}
			if dialect == agent.DialectJSON && !strings.HasSuffix(string(encoded), "}\n") {
				t.Fatalf("JSON output must end with a newline: %q", encoded)
			}
		})
	}
}
