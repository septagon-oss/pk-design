package handoff_test

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// handoff_test.go proves deterministic snapshots, layered profiles, minimal
// diffs, ownership enforcement, conflict detection, and atomic application.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func TestSnapshotDigestIsDeterministicAndIgnoresGenerationTime(t *testing.T) {
	t.Parallel()

	first := testSnapshot()
	second := testSnapshot()
	second.Provenance.GeneratedAt = first.Provenance.GeneratedAt.Add(24 * time.Hour)
	second.Tokens[0], second.Tokens[1] = second.Tokens[1], second.Tokens[0]

	firstDigest, err := first.Digest()
	if err != nil {
		t.Fatalf("first Digest() error = %v", err)
	}
	secondDigest, err := second.Digest()
	if err != nil {
		t.Fatalf("second Digest() error = %v", err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("digest changed for equivalent content: %s != %s", firstDigest, secondDigest)
	}
}

func TestLayeredProfilesFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile handoff.Profile
	}{
		{
			name: "oss with parent",
			profile: handoff.Profile{
				ID: "platformkit-oss", Kind: handoff.ProfileOSS, Version: "1.0.0",
				ParentID: "other", ParentDigest: digestA,
			},
		},
		{
			name: "pro without parent digest",
			profile: handoff.Profile{
				ID: "platformkit-pro", Kind: handoff.ProfilePro, Version: "1.0.0",
				ParentID: "platformkit-oss",
			},
		},
		{
			name: "client without client id",
			profile: handoff.Profile{
				ID: "client/collect", Kind: handoff.ProfileClient, Version: "1.0.0",
				ParentID: "platformkit-pro", ParentDigest: digestA,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			snapshot := testSnapshot()
			snapshot.Profile = test.profile
			if _, err := snapshot.Normalize(); err == nil {
				t.Fatalf("Normalize() accepted profile %#v", test.profile)
			}
		})
	}
}

func TestDiffProducesMinimalDeterministicTransaction(t *testing.T) {
	t.Parallel()

	base := testSnapshot()
	edited := testSnapshot()
	edited.Tokens[0].Value = "#2563eb"
	edited.Tokens = append(edited.Tokens[:1], handoff.Token{
		Path:  "/color/brand/soft",
		Mode:  "light",
		Type:  tokens.TypeColor,
		Value: "#dbeafe",
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "semantic",
			SourceURI: "src/semantic.tokens.json", Writable: true,
		},
	})

	changes, err := handoff.Diff(base, edited, provenance("figma"))
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if len(changes.Changes) != 3 {
		t.Fatalf("Diff() produced %d changes, want 3", len(changes.Changes))
	}
	gotOperations := []handoff.Operation{
		changes.Changes[0].Operation,
		changes.Changes[1].Operation,
		changes.Changes[2].Operation,
	}
	wantOperations := []handoff.Operation{
		handoff.OperationUpdate,
		handoff.OperationAdd,
		handoff.OperationRemove,
	}
	for index := range wantOperations {
		if gotOperations[index] != wantOperations[index] {
			t.Fatalf("operation[%d] = %q, want %q", index, gotOperations[index], wantOperations[index])
		}
	}

	applied, err := handoff.Apply(base, changes)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	appliedDigest, err := applied.Digest()
	if err != nil {
		t.Fatalf("applied Digest() error = %v", err)
	}
	edited.Provenance = changes.Provenance
	editedDigest, err := edited.Digest()
	if err != nil {
		t.Fatalf("edited Digest() error = %v", err)
	}
	if appliedDigest != editedDigest {
		t.Fatalf("applied digest %s != edited digest %s", appliedDigest, editedDigest)
	}
}

func TestDiffRejectsReadOnlyAndOriginMutation(t *testing.T) {
	t.Parallel()

	t.Run("read only", func(t *testing.T) {
		base := testSnapshot()
		base.Tokens[0].Origin.Writable = false
		edited := testSnapshot()
		edited.Tokens[0].Origin.Writable = false
		edited.Tokens[0].Value = "#000000"
		_, err := handoff.Diff(base, edited, provenance("figma"))
		requireConflict(t, err, handoff.ConflictOwnership)
	})

	t.Run("origin mutation", func(t *testing.T) {
		base := testSnapshot()
		edited := testSnapshot()
		edited.Tokens[0].Origin.SourceURI = "other.cue"
		_, err := handoff.Diff(base, edited, provenance("figma"))
		requireConflict(t, err, handoff.ConflictOwnership)
	})
}

func TestApplyDetectsBaseDriftAndExpectedValueMismatch(t *testing.T) {
	t.Parallel()

	base := testSnapshot()
	edited := testSnapshot()
	edited.Tokens[0].Value = "#2563eb"
	changes, err := handoff.Diff(base, edited, provenance("figma"))
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	drifted := testSnapshot()
	drifted.Tokens[1].Value = "#f9fafb"
	_, err = handoff.Apply(drifted, changes)
	requireConflict(t, err, handoff.ConflictBaseDigest)

	tampered := changes
	before := *tampered.Changes[0].Before
	before.Value = "#ffffff"
	tampered.Changes[0].Before = &before
	_, err = handoff.Apply(base, tampered)
	requireConflict(t, err, handoff.ConflictTokenMismatch)
}

func TestSnapshotJSONRoundTripPreservesCompositeValues(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	snapshot.Tokens = append(snapshot.Tokens, handoff.Token{
		Path: "/shadow/card", Mode: handoff.DefaultMode, Type: tokens.TypeShadow,
		Value: map[string]any{
			"color":   "#00000033",
			"offsetX": map[string]any{"value": json.Number("0"), "unit": "px"},
			"offsetY": map[string]any{"value": json.Number("2"), "unit": "px"},
		},
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "primitive",
			SourceURI: "src/primitives.tokens.json", Writable: true,
		},
	})
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded handoff.Snapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, err := decoded.Normalize(); err != nil {
		t.Fatalf("decoded Normalize() error = %v", err)
	}
}

func TestFromSetPreservesDTCGMetadataAndBuildsPointers(t *testing.T) {
	t.Parallel()

	snapshot, err := handoff.FromSet(tokens.Set{
		Name: "pk",
		Values: map[string]tokens.Value{
			"color.brand.primary": "#0f5d4e",
			"space.1":             "4px",
		},
		Types: map[string]tokens.Type{
			"color.brand.primary": tokens.TypeColor,
			"space.1":             tokens.TypeDimension,
		},
		Descriptions: map[string]string{
			"color.brand.primary": "Primary action color.",
		},
	}, handoff.SetSnapshotOptions{
		Profile:    testSnapshot().Profile,
		Provenance: provenance("pk-design"),
		Mode:       "Light",
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "semantic",
			SourceURI: "themes/default.go", Writable: true,
		},
	})
	if err != nil {
		t.Fatalf("FromSet() error = %v", err)
	}
	if len(snapshot.Tokens) != 2 {
		t.Fatalf("FromSet() tokens = %d, want 2", len(snapshot.Tokens))
	}
	first := snapshot.Tokens[0]
	if first.Path != "/color/brand/primary" ||
		first.Type != tokens.TypeColor ||
		first.Description != "Primary action color." ||
		first.Origin.Pointer != "/tokens/color/brand/primary/$value" {
		t.Fatalf("FromSet() first token = %#v", first)
	}
}

func TestJSONPointerPathsPreserveLiteralDotsAndSlashes(t *testing.T) {
	t.Parallel()

	pointer, err := handoff.PathFromSegments("spacing", "0.5", "mobile/tablet", "~internal")
	if err != nil {
		t.Fatalf("PathFromSegments() error = %v", err)
	}
	if pointer != "/spacing/0.5/mobile~1tablet/~0internal" {
		t.Fatalf("PathFromSegments() = %q", pointer)
	}
	segments, err := handoff.PathSegments(pointer)
	if err != nil {
		t.Fatalf("PathSegments() error = %v", err)
	}
	want := []string{"spacing", "0.5", "mobile/tablet", "~internal"}
	for index := range want {
		if segments[index] != want[index] {
			t.Fatalf("PathSegments()[%d] = %q, want %q", index, segments[index], want[index])
		}
	}

	snapshot := testSnapshot()
	snapshot.Tokens = append(snapshot.Tokens, handoff.Token{
		Path: "/spacing/0.5", Mode: "light", Type: tokens.TypeDimension, Value: "2px",
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "primitive",
			SourceURI: "src/spacing.tokens.json", Pointer: "/tokens/spacing/0.5/$value",
			Writable: true,
		},
	})
	if _, err := snapshot.Normalize(); err != nil {
		t.Fatalf("Normalize() rejected literal-dot segment: %v", err)
	}
}

func TestSnapshotRejectsPointerAncestorCollision(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	snapshot.Tokens = append(snapshot.Tokens, handoff.Token{
		Path: "/color/brand", Mode: "light", Type: tokens.TypeColor, Value: "#000000",
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "semantic",
			SourceURI: "src/semantic.tokens.json", Pointer: "/tokens/color/brand/$value",
			Writable: true,
		},
	})
	if _, err := snapshot.Normalize(); err == nil {
		t.Fatal("Normalize() accepted a token/group path collision")
	}
}

func requireConflict(t *testing.T, err error, code handoff.ConflictCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected conflict %s", code)
	}
	var conflict handoff.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("error %T is not ConflictError: %v", err, err)
	}
	if conflict.Code != code {
		t.Fatalf("conflict code = %s, want %s", conflict.Code, code)
	}
}

func testSnapshot() handoff.Snapshot {
	return handoff.Snapshot{
		SchemaVersion: handoff.SnapshotSchemaVersion,
		Profile: handoff.Profile{
			ID: "platformkit-oss", Kind: handoff.ProfileOSS, Version: "1.0.0",
		},
		Provenance: provenance("pk-design"),
		Metadata:   map[string]string{"catalogDigest": digestA},
		Tokens: []handoff.Token{
			{
				Path: "/color/brand/primary", Mode: "light", Type: tokens.TypeColor, Value: "#3b82f6",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "semantic",
					SourceURI: "src/semantic.tokens.json", Pointer: "/color/brand/primary",
					Writable: true,
				},
			},
			{
				Path: "/color/surface/primary", Mode: "light", Type: tokens.TypeColor, Value: "#ffffff",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "semantic",
					SourceURI: "src/semantic.tokens.json", Pointer: "/color/surface/primary",
					Writable: true,
				},
			},
		},
	}
}

func provenance(source string) handoff.Provenance {
	return handoff.Provenance{
		Source: source, GeneratedAt: time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
	}
}

const digestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
