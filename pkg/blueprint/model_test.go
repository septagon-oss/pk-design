package blueprint

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// model_test.go proves the OSS visual contract remains provider-neutral and
// serializes the stable shape consumed by downstream delivery adapters.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefinitionSerializesNativeSemanticNodesWithoutRasterPayloads(t *testing.T) {
	t.Parallel()

	definition := Definition{
		SourceOfTruth:    SourceDefinition,
		CanonicalExample: "default",
		Root: &Node{
			Kind:              NodeFrame,
			Name:              "Button",
			Classes:           "inline-flex bg-surface-brand",
			ClassBindings:     map[string]map[string]string{"tone": {"danger": "bg-surface-danger"}},
			ClassBindingOrder: []string{"tone"},
			Children: []Node{{
				Kind: NodeText,
				Name: "Label",
				Text: "Continue",
			}},
		},
	}
	data, err := json.Marshal(definition)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	payload := string(data)
	for _, forbidden := range []string{"data:image", "png", "jpeg", "snapshot"} {
		if strings.Contains(strings.ToLower(payload), forbidden) {
			t.Fatalf("blueprint contains raster/snapshot payload %q: %s", forbidden, payload)
		}
	}
	if !strings.Contains(payload, `"kind":"frame"`) ||
		!strings.Contains(payload, `"kind":"text"`) {
		t.Fatalf("blueprint lost semantic native nodes: %s", payload)
	}
	if !strings.Contains(payload, `"class_binding_order":["tone"]`) {
		t.Fatalf("blueprint lost authored class-binding precedence: %s", payload)
	}
}
