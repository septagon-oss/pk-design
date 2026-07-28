package figma_test

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// render_test.go proves exact path bindings, native aliases, codecs,
// read-only degradation, and deterministic OSS Figma deliveries.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (layered design delivery).
// Convention: C-14 (every Go file declares its purpose).

import (
	"strings"
	"testing"
	"time"

	"github.com/septagon-oss/pk-design/pkg/figma"
	"github.com/septagon-oss/pk-design/pkg/handoff"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func TestRenderBuildsNativeAliasesAndExactBindings(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	bundle, err := figma.Render(snapshot, figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if bundle.Schema != figma.BundleSchemaVersion || bundle.SnapshotDigest == "" {
		t.Fatalf("Render() identity = %#v", bundle)
	}
	if len(bundle.Collections) != 2 {
		t.Fatalf("Render() collections = %d, want 2", len(bundle.Collections))
	}
	source := findVariable(t, bundle, "/color/primitive/brand/500")
	if !source.Binding.Writable || source.Type != "COLOR" {
		t.Fatalf("source variable = %#v", source)
	}
	if source.Binding.TokenPath != "/color/primitive/brand/500" ||
		source.Binding.ModeMap["Light"] != "light" {
		t.Fatalf("source binding = %#v", source.Binding)
	}
	resolved := findVariable(t, bundle, "/color/surface/brand")
	if resolved.Binding.Writable {
		t.Fatalf("resolved variable is writable: %#v", resolved)
	}
	if resolved.Aliases["Light"] != "/color/primitive/brand/500" {
		t.Fatalf("resolved alias = %#v", resolved.Aliases)
	}
}

func TestRenderAllowsModeSpecificSemanticAliasToModeIndependentSource(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	for index := range snapshot.Tokens {
		if snapshot.Tokens[index].Path == "/color/primitive/brand/500" {
			snapshot.Tokens[index].Mode = handoff.DefaultMode
		}
	}
	snapshot.Tokens = append(snapshot.Tokens[:1], snapshot.Tokens[2:]...)

	bundle, err := figma.Render(snapshot, figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	resolved := findVariable(t, bundle, "/color/surface/brand")
	if resolved.Aliases["Light"] != "/color/primitive/brand/500" ||
		resolved.Aliases["Dark"] != "/color/primitive/brand/500" {
		t.Fatalf("resolved aliases = %#v", resolved.Aliases)
	}
}

func TestRenderKeepsInheritedReadOnlyPrimitivesInSourceCollection(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	for index := range snapshot.Tokens {
		if snapshot.Tokens[index].Path == "/color/primitive/brand/500" {
			snapshot.Tokens[index].Origin.Writable = false
		}
	}
	bundle, err := figma.Render(snapshot, figma.Options{Product: "Client"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, collection := range bundle.Collections {
		for _, variable := range collection.Variables {
			if variable.Key != "/color/primitive/brand/500" {
				continue
			}
			if collection.Name != "PlatformKit Sources" {
				t.Fatalf("read-only inherited primitive collection = %q", collection.Name)
			}
			return
		}
	}
	t.Fatal("read-only inherited primitive was not rendered")
}

func TestRenderPreservesLiteralDotPathAndDimensionCodec(t *testing.T) {
	t.Parallel()

	bundle, err := figma.Render(testSnapshot(), figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatal(err)
	}
	variable := findVariable(t, bundle, "/spacing/0.5")
	if variable.Name != "spacing/0.5" || variable.Type != "FLOAT" {
		t.Fatalf("spacing variable = %#v", variable)
	}
	if variable.Values["Default"] != float64(2) {
		t.Fatalf("spacing native value = %#v", variable.Values["Default"])
	}
	codec := variable.Binding.Codecs["Default"]
	if codec.Kind != "dimension" || codec.Unit != "rem" || codec.Scale != 16 || codec.Precision != 3 {
		t.Fatalf("spacing codec = %#v", codec)
	}
}

func TestRenderDegradesCompositeWritableTokenToReadOnly(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	snapshot.Tokens = append(snapshot.Tokens, handoff.Token{
		Path: "/shadow/card", Mode: handoff.DefaultMode, Type: tokens.TypeShadow,
		Value: map[string]any{"color": "#00000033"},
		Origin: handoff.Origin{
			Owner: "septagon-oss/pk-design", Layer: "primitive",
			SourceURI: "tokens.json", Pointer: "/shadow/card/$value", Writable: true,
		},
	})
	bundle, err := figma.Render(snapshot, figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatal(err)
	}
	variable := findVariable(t, bundle, "/shadow/card")
	if variable.Binding.Writable {
		t.Fatal("composite shadow remained writable")
	}
	if len(bundle.Diagnostics) != 1 ||
		bundle.Diagnostics[0].Code != "PKF001_READ_ONLY_CODEC" {
		t.Fatalf("diagnostics = %#v", bundle.Diagnostics)
	}
}

func TestRenderKeepsNonColorSemanticSourcesOutOfColorModes(t *testing.T) {
	t.Parallel()

	snapshot := testSnapshot()
	snapshot.Tokens = append(snapshot.Tokens, handoff.Token{
		Path: "/borderRadius/md", Mode: handoff.DefaultMode,
		Type: tokens.TypeDimension, Value: "0.375rem",
		Origin: handoff.Origin{
			Owner: "septagon-dev/platformkit-design-system", Layer: "semantic",
			SourceURI: "semantic.cue", Pointer: "/tokens/borderRadius/md/$value",
			Writable: true,
		},
	})
	bundle, err := figma.Render(snapshot, figma.Options{Product: "PlatformKit Pro"})
	if err != nil {
		t.Fatal(err)
	}
	for _, collection := range bundle.Collections {
		if collection.Name == "PlatformKit Colors" {
			for _, mode := range collection.Modes {
				if mode == "Default" {
					t.Fatalf("non-color semantic source polluted color modes: %#v", collection.Modes)
				}
			}
		}
		for _, variable := range collection.Variables {
			if variable.Key == "/borderRadius/md" &&
				collection.Name != "PlatformKit Sources" {
				t.Fatalf("semantic dimension collection = %q", collection.Name)
			}
		}
	}
}

func TestRenderJSONIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := figma.RenderJSON(testSnapshot(), figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatal(err)
	}
	secondSnapshot := testSnapshot()
	secondSnapshot.Tokens[0], secondSnapshot.Tokens[1] =
		secondSnapshot.Tokens[1], secondSnapshot.Tokens[0]
	second, err := figma.RenderJSON(secondSnapshot, figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("RenderJSON() is not deterministic:\n%s\n---\n%s", first, second)
	}
	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("RenderJSON() is not newline terminated")
	}
}

func TestValidateRejectsTamperedSnapshotDigest(t *testing.T) {
	t.Parallel()

	bundle, err := figma.Render(testSnapshot(), figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	bundle.SnapshotDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := figma.Validate(bundle); err == nil {
		t.Fatal("Validate() accepted a tampered snapshot digest")
	}
}

func TestValidateRejectsMissingNativeSnapshotPath(t *testing.T) {
	t.Parallel()

	bundle, err := figma.Render(testSnapshot(), figma.Options{Product: "PlatformKit OSS"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(bundle.Collections) == 0 || len(bundle.Collections[0].Variables) == 0 {
		t.Fatal("test bundle has no variables")
	}
	bundle.Collections[0].Variables = bundle.Collections[0].Variables[1:]
	if err := figma.Validate(bundle); err == nil {
		t.Fatal("Validate() accepted a snapshot path with no native variable")
	}
}

func findVariable(t *testing.T, bundle figma.Bundle, key string) figma.Variable {
	t.Helper()
	for _, collection := range bundle.Collections {
		for _, variable := range collection.Variables {
			if variable.Key == key {
				return variable
			}
		}
	}
	t.Fatalf("variable %q not found", key)
	return figma.Variable{}
}

func testSnapshot() handoff.Snapshot {
	generatedAt := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	return handoff.Snapshot{
		SchemaVersion: handoff.SnapshotSchemaVersion,
		Profile: handoff.Profile{
			ID: "platformkit-oss", Kind: handoff.ProfileOSS, Version: "1.0.0",
		},
		Provenance: handoff.Provenance{Source: "pk-design", GeneratedAt: generatedAt},
		Tokens: []handoff.Token{
			{
				Path: "/color/primitive/brand/500", Mode: "light",
				Type: tokens.TypeColor, Value: "#3B82F6",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "primitive",
					SourceURI: "tokens.json", Pointer: "/color/primitive/brand/500/$value",
					Writable: true,
				},
			},
			{
				Path: "/color/primitive/brand/500", Mode: "dark",
				Type: tokens.TypeColor, Value: "37 99 235",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "primitive",
					SourceURI: "tokens.json", Pointer: "/color/primitive/brand/500/$value",
					Writable: true,
				},
			},
			{
				Path: "/color/surface/brand", Mode: "light",
				Type: tokens.TypeColor, Value: "{color.primitive.brand.500}",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "semantic",
					SourceURI: "tokens.json", Pointer: "/color/surface/brand/$value",
					Writable: false,
				},
			},
			{
				Path: "/color/surface/brand", Mode: "dark",
				Type: tokens.TypeColor, Value: "{color.primitive.brand.500}",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "semantic",
					SourceURI: "tokens.json", Pointer: "/color/surface/brand/$value",
					Writable: false,
				},
			},
			{
				Path: "/spacing/0.5", Mode: handoff.DefaultMode,
				Type: tokens.TypeDimension, Value: "0.125rem",
				Origin: handoff.Origin{
					Owner: "septagon-oss/pk-design", Layer: "primitive",
					SourceURI: "tokens.json", Pointer: "/spacing/0.5/$value",
					Writable: true,
				},
			},
		},
	}
}
