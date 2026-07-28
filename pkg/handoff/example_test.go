package handoff_test

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// example_test.go demonstrates the portable external-editor round trip.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-14 (every Go file declares its purpose).

import (
	"fmt"
	"time"

	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func ExampleDiff() {
	profile := handoff.Profile{
		ID: "platformkit-oss", Kind: handoff.ProfileOSS, Version: "1.0.0",
	}
	origin := handoff.Origin{
		Owner: "example/design", Layer: "semantic",
		SourceURI: "tokens/semantic.json", Writable: true,
	}
	base := handoff.Snapshot{
		SchemaVersion: handoff.SnapshotSchemaVersion,
		Profile:       profile,
		Provenance: handoff.Provenance{
			Source: "compiler", GeneratedAt: time.Unix(1, 0).UTC(),
		},
		Tokens: []handoff.Token{{
			Path: "/color/brand/primary", Mode: "light",
			Type: tokens.TypeColor, Value: "#3b82f6", Origin: origin,
		}},
	}
	edited := base
	edited.Tokens = append([]handoff.Token(nil), base.Tokens...)
	edited.Tokens[0].Value = "#2563eb"

	changes, err := handoff.Diff(base, edited, handoff.Provenance{
		Source: "figma", GeneratedAt: time.Unix(2, 0).UTC(),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(changes.SchemaVersion)
	fmt.Println(changes.Changes[0].Operation)
	// Output:
	// pk.design.token-changes.v1
	// update
}
