package tokens

// tokens_test.go validates token normalization, overlays, CSS export, DTCG
// export, and defensive-copy behavior for the OSS design-token contract.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
)

func validSet() Set {
	return Set{
		Name:    "pk",
		Version: "0.1.0",
		Values: map[string]Value{
			"color.surface.primary": "#ffffff",
			"color.text.primary":    "#111827",
			"space.2":               "0.5rem",
		},
		Types: map[string]Type{
			"color.surface.primary": TypeColor,
			"color.text.primary":    TypeColor,
			"space.2":               TypeDimension,
		},
		Descriptions: map[string]string{
			"color.surface.primary": "Default page surface",
		},
		Extensions: map[string]map[string]any{
			"color.surface.primary": {"mode": "light"},
		},
		Metadata: map[string]any{" owner ": "design"},
	}
}

func TestNormalizeAndKeys(t *testing.T) {
	t.Parallel()

	normalized, err := validSet().Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	keys, err := normalized.Keys()
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	want := []string{"color.surface.primary", "color.text.primary", "space.2"}
	if !slices.Equal(keys, want) {
		t.Fatalf("Keys() = %v; want %v", keys, want)
	}
	if normalized.Metadata["owner"] != "design" {
		t.Fatalf("Metadata = %#v; want normalized owner", normalized.Metadata)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []Set{
		{},
		{Name: "bad name", Values: map[string]Value{"color.surface": "#fff"}},
		{Name: "pk", Values: map[string]Value{"": "#fff"}},
		{Name: "pk", Values: map[string]Value{"color.": "#fff"}},
		{Name: "pk", Values: map[string]Value{"color surface": "#fff"}},
		{Name: "pk", Values: map[string]Value{"color.surface": " "}},
		{Name: "pk", Values: map[string]Value{"color": "#fff", "color.surface": "#fff"}},
		{Name: "pk", Values: map[string]Value{"color.surface": "#fff"}, Types: map[string]Type{"color.surface": "palette"}},
		{Name: "pk", Values: map[string]Value{"color.surface": "#fff"}, Types: map[string]Type{"color.missing": TypeColor}},
		{Name: "pk", Values: map[string]Value{"color.surface": "#fff"}, Descriptions: map[string]string{"color.missing": "Missing"}},
		{Name: "pk", Values: map[string]Value{"color.surface": "#fff"}, Extensions: map[string]map[string]any{"color.missing": {"owner": "design"}}},
	}
	for _, test := range tests {
		if _, err := test.Normalize(); err == nil {
			t.Fatalf("Normalize(%#v) should fail", test)
		}
	}
}

func TestNormalizeAcceptsCoreAndExtensionTypes(t *testing.T) {
	t.Parallel()

	set := Set{
		Name: "pk",
		Values: map[string]Value{
			"motion.ease.standard":     "cubic-bezier(0.2, 0, 0, 1)",
			"typography.weight.strong": "700",
			"vendor.asset.logo":        "platformkit",
		},
		Types: map[string]Type{
			"motion.ease.standard":     TypeCubicBezier,
			"typography.weight.strong": TypeFontWeight,
			"vendor.asset.logo":        "x.asset",
		},
	}
	if _, err := set.Normalize(); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
}

func TestLookupReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Values["shadow.card"] = map[string]any{"color": "{color.surface.primary}"}
	set.Types["shadow.card"] = TypeShadow
	token, ok, err := set.Lookup("color.surface.primary")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !ok {
		t.Fatal("Lookup() ok = false")
	}
	token.Extensions["mode"] = "dark"

	again, _, err := set.Lookup("color.surface.primary")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if again.Extensions["mode"] != "light" {
		t.Fatalf("Lookup returned aliased extensions: %#v", again.Extensions)
	}

	shadow, _, err := set.Lookup("shadow.card")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	shadow.Value.(map[string]any)["color"] = "mutated"
	shadowAgain, _, err := set.Lookup("shadow.card")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if shadowAgain.Value.(map[string]any)["color"] != "{color.surface.primary}" {
		t.Fatalf("Lookup returned aliased composite value: %#v", shadowAgain.Value)
	}
}

func TestMergeOverlays(t *testing.T) {
	t.Parallel()

	merged, err := Merge(validSet(), Set{
		Values: map[string]Value{
			"color.text.primary": "#0f172a",
			"color.brand":        "#2563eb",
		},
		Types: map[string]Type{"color.brand": TypeColor},
	})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if merged.Values["color.text.primary"] != "#0f172a" || merged.Values["color.brand"] != "#2563eb" {
		t.Fatalf("merged values = %#v", merged.Values)
	}

	if _, err := Merge(validSet(), Set{Name: "other", Values: map[string]Value{"color.brand": "#000"}}); err == nil {
		t.Fatal("Merge() should reject mismatched overlay names")
	}

	if _, err := Merge(
		Set{Name: "pk", Values: map[string]Value{"color": "#fff"}},
		Set{Name: "pk", Values: map[string]Value{"color.surface": "#fff"}},
	); err == nil {
		t.Fatal("Merge() should reject path conflicts introduced by overlays")
	}
}

func TestCSSVarsAndCSSMap(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Name = "PK.Brand"
	css, err := CSSVars(set)
	if err != nil {
		t.Fatalf("CSSVars() error = %v", err)
	}
	for _, want := range []string{
		"--pk-brand-color-surface-primary: #ffffff;",
		"--pk-brand-color-text-primary: #111827;",
		"--pk-brand-space-2: 0.5rem;",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("CSSVars() missing %q in:\n%s", want, css)
		}
	}

	cssMap, err := CSSMap(set)
	if err != nil {
		t.Fatalf("CSSMap() error = %v", err)
	}
	if cssMap["--pk-brand-color-text-primary"] != "#111827" {
		t.Fatalf("CSSMap() = %#v", cssMap)
	}
}

func TestCSSMapRendersRootTokensAsGroupVariables(t *testing.T) {
	t.Parallel()

	cssMap, err := CSSMap(Set{
		Name: "pk",
		Values: map[string]Value{
			"color.brand.$root": "#2563eb",
		},
		Types: map[string]Type{
			"color.brand.$root": TypeColor,
		},
	})
	if err != nil {
		t.Fatalf("CSSMap() error = %v", err)
	}
	if cssMap["--pk-color-brand"] != "#2563eb" {
		t.Fatalf("CSSMap() = %#v; want root token exported as group variable", cssMap)
	}
	if _, exists := cssMap["--pk-color-brand-root"]; exists {
		t.Fatalf("CSSMap() leaked DTCG root sentinel: %#v", cssMap)
	}
}

func TestDTCGJSON(t *testing.T) {
	t.Parallel()

	data, err := DTCGJSON(validSet())
	if err != nil {
		t.Fatalf("DTCGJSON() error = %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("DTCGJSON emitted invalid JSON: %v", err)
	}
	color := doc["color"].(map[string]any)
	surface := color["surface"].(map[string]any)
	primary := surface["primary"].(map[string]any)
	if primary["$value"] != "#ffffff" || primary["$type"] != string(TypeColor) {
		t.Fatalf("DTCG primary token = %#v", primary)
	}
}

func TestDTCGRoundTripPreservesRootGroupsAndObjectValues(t *testing.T) {
	t.Parallel()

	set := Set{
		Name: "pk",
		Values: map[string]Value{
			"color.brand.$root": "#2563eb",
			"shadow.card": map[string]any{
				"color": "{color.brand.$root}",
				"x":     0,
				"y":     8,
				"blur":  24,
			},
		},
		Types: map[string]Type{
			"shadow.card": TypeShadow,
		},
		Groups: map[string]Group{
			"color.brand": {
				Type:        TypeColor,
				Description: "Brand color group",
				Extensions:  map[string]any{"scope": "brand"},
			},
		},
	}

	data, err := DTCGJSON(set)
	if err != nil {
		t.Fatalf("DTCGJSON() error = %v", err)
	}
	roundTripped, err := ParseDTCGJSON("pk", data)
	if err != nil {
		t.Fatalf("ParseDTCGJSON() error = %v", err)
	}
	token, ok, err := roundTripped.Lookup("color.brand.$root")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !ok || token.Value != "#2563eb" || token.Type != TypeColor {
		t.Fatalf("round-tripped root token = %#v", token)
	}
	shadow, ok, err := roundTripped.Lookup("shadow.card")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !ok || shadow.Type != TypeShadow {
		t.Fatalf("round-tripped shadow token = %#v", shadow)
	}
	shadowValue := shadow.Value.(map[string]any)
	if shadowValue["color"] != "{color.brand.$root}" || shadowValue["blur"].(json.Number).String() != "24" {
		t.Fatalf("shadow value = %#v", shadow.Value)
	}
}

func TestParseDTCGDoesNotMaterializeInheritedGroupMetadata(t *testing.T) {
	t.Parallel()

	set, err := ParseDTCG("pk", map[string]any{
		"color": map[string]any{
			"$type": "color",
			"brand": map[string]any{
				"primary": map[string]any{"$value": "#2563eb"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseDTCG() error = %v", err)
	}
	if _, exists := set.Groups["color.brand"]; exists {
		t.Fatalf("ParseDTCG() materialized inherited group metadata: %#v", set.Groups)
	}
	token, ok, err := set.Lookup("color.brand.primary")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !ok || token.Type != TypeColor {
		t.Fatalf("Lookup() inherited token type = %#v", token)
	}
}

func TestResolveAliasesAndCompositeValues(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(Set{
		Name: "pk",
		Values: map[string]Value{
			"color.brand.$root":      "#2563eb",
			"color.action.primary":   "{color.brand.$root}",
			"shadow.focus":           map[string]any{"color": "{color.action.primary}", "blur": 6},
			"typography.weight.bold": 700,
		},
		Types: map[string]Type{
			"color.brand.$root":      TypeColor,
			"color.action.primary":   TypeColor,
			"shadow.focus":           TypeShadow,
			"typography.weight.bold": TypeFontWeight,
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Values["color.action.primary"] != "#2563eb" {
		t.Fatalf("resolved alias = %#v", resolved.Values["color.action.primary"])
	}
	shadow := resolved.Values["shadow.focus"].(map[string]any)
	if shadow["color"] != "#2563eb" {
		t.Fatalf("resolved composite value = %#v", shadow)
	}
}

func TestResolveRejectsMissingReferenceAndCycles(t *testing.T) {
	t.Parallel()

	if _, err := Resolve(Set{
		Name:   "pk",
		Values: map[string]Value{"color.action.primary": "{color.missing}"},
	}); err == nil {
		t.Fatal("Resolve() should reject missing token references")
	}
	if _, err := Resolve(Set{
		Name: "pk",
		Values: map[string]Value{
			"color.a": "{color.b}",
			"color.b": "{color.a}",
		},
	}); err == nil {
		t.Fatal("Resolve() should reject token reference cycles")
	}
}

func TestResolveAppliesGroupExtends(t *testing.T) {
	t.Parallel()

	merged, err := Merge(
		Set{
			Name: "pk",
			Values: map[string]Value{
				"color.brand.primary":   "#2563eb",
				"color.brand.secondary": "#60a5fa",
			},
			Groups: map[string]Group{"color.brand": {Type: TypeColor}},
		},
		Set{
			Name: "pk",
			Values: map[string]Value{
				"color.accent.primary": "#7c3aed",
			},
			Groups: map[string]Group{
				"color.accent": {
					Type:    TypeColor,
					Extends: "{color.brand}",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	resolved, err := Resolve(merged)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Values["color.accent.primary"] != "#7c3aed" {
		t.Fatalf("explicit target override was not preserved: %#v", resolved.Values)
	}
	if resolved.Values["color.accent.secondary"] != "#60a5fa" {
		t.Fatalf("group $extends did not inherit missing child token: %#v", resolved.Values)
	}
}

func TestValidateReturnsStableIssueCodes(t *testing.T) {
	t.Parallel()

	report := Validate(Set{
		Name: "bad name",
		Values: map[string]Value{
			"color":         "#fff",
			"color.surface": "#fff",
		},
	})
	if !report.HasErrors() {
		t.Fatal("Validate() HasErrors() = false")
	}
	codes := map[IssueCode]bool{}
	for _, issue := range report.Issues {
		codes[issue.Code] = true
	}
	if !codes[IssueInvalidName] || !codes[IssuePathConflict] {
		t.Fatalf("Validate() issue codes = %#v", report.Issues)
	}
}

func TestCSSVarsRejectsUnrenderableCompositeValues(t *testing.T) {
	t.Parallel()

	_, err := CSSVars(Set{
		Name: "pk",
		Values: map[string]Value{
			"shadow.card": map[string]any{"x": 0, "y": 1},
		},
		Types: map[string]Type{"shadow.card": TypeShadow},
	})
	if err == nil {
		t.Fatal("CSSVars() should reject unrenderable composite values")
	}
}

func TestCSSVarsRendersDTCGDimensionObject(t *testing.T) {
	t.Parallel()

	cssMap, err := CSSMap(Set{
		Name: "pk",
		Values: map[string]Value{
			"color.brand":   map[string]any{"hex": "#2563eb"},
			"duration.fast": json.RawMessage(`"120ms"`),
			"opacity.full":  1,
			"space.2":       map[string]any{"value": json.Number("0.5"), "unit": "rem"},
			"state.enabled": true,
		},
		Types: map[string]Type{
			"color.brand":   TypeColor,
			"duration.fast": TypeDuration,
			"opacity.full":  TypeNumber,
			"space.2":       TypeDimension,
			"state.enabled": TypeString,
		},
	})
	if err != nil {
		t.Fatalf("CSSMap() error = %v", err)
	}
	if cssMap["--pk-space-2"] != "0.5rem" {
		t.Fatalf("CSSMap() = %#v", cssMap)
	}
	if cssMap["--pk-color-brand"] != "#2563eb" || cssMap["--pk-duration-fast"] != "120ms" || cssMap["--pk-opacity-full"] != "1" || cssMap["--pk-state-enabled"] != "true" {
		t.Fatalf("CSSMap() = %#v", cssMap)
	}
}

func TestParseReference(t *testing.T) {
	t.Parallel()

	path, ok := ParseReference(" {color.brand.primary} ")
	if !ok || path != "color.brand.primary" {
		t.Fatalf("ParseReference() = %q, %v", path, ok)
	}
	for _, invalid := range []string{"color.brand.primary", "{color brand}", "{$root}", "{color.$root.child}"} {
		if path, ok := ParseReference(invalid); ok {
			t.Fatalf("ParseReference(%q) = %q, true; want false", invalid, path)
		}
	}
}

func TestValueHelpers(t *testing.T) {
	t.Parallel()

	ref, err := Reference(" color.brand.primary ")
	if err != nil {
		t.Fatalf("Reference() error = %v", err)
	}
	if ref != "{color.brand.primary}" {
		t.Fatalf("Reference() = %q", ref)
	}
	if _, err := Reference("color brand"); err == nil {
		t.Fatal("Reference() should reject invalid token paths")
	}

	stringToken := Token{Value: json.RawMessage(`"compact"`)}
	if value, ok := stringToken.StringValue(); !ok || value != "compact" {
		t.Fatalf("StringValue() = %q, %v", value, ok)
	}
	numberToken := Token{Value: json.RawMessage(`0.5`)}
	if value, ok := numberToken.NumberValue(); !ok || value.String() != "0.5" {
		t.Fatalf("NumberValue() = %q, %v", value, ok)
	}
	mapToken := Token{Value: map[string]any{"value": json.Number("0.5"), "unit": "rem"}}
	value, ok := mapToken.MapValue()
	if !ok {
		t.Fatal("MapValue() ok = false")
	}
	value["unit"] = "px"
	again, ok := mapToken.MapValue()
	if !ok || again["unit"] != "rem" {
		t.Fatalf("MapValue() returned aliased map: %#v", again)
	}

	originalValue := map[string]any{"items": []any{"a"}}
	copied := CopyValue(originalValue).(map[string]any)
	copied["items"].([]any)[0] = "b"
	if originalValue["items"].([]any)[0] != "a" {
		t.Fatalf("CopyValue() returned aliased value: %#v", originalValue)
	}
	stringMapToken := Token{Value: map[string]string{"unit": "rem"}}
	stringMap, ok := stringMapToken.MapValue()
	if !ok || stringMap["unit"] != "rem" {
		t.Fatalf("MapValue() string map = %#v, %v", stringMap, ok)
	}
	if _, ok := (Token{Value: math.NaN()}).NumberValue(); ok {
		t.Fatal("NumberValue() should reject non-finite floats")
	}
}

func TestParseDTCGRejectsInvalidDocuments(t *testing.T) {
	t.Parallel()

	if _, err := ParseDTCG("pk", map[string]any{
		"color": map[string]any{
			"$value": "#fff",
			"brand":  map[string]any{"$value": "#000"},
		},
	}); err == nil {
		t.Fatal("ParseDTCG() should reject token nodes with children")
	}
	if _, err := ParseDTCG("pk", map[string]any{
		"color": "not an object",
	}); err == nil {
		t.Fatal("ParseDTCG() should reject non-object children")
	}
	if _, err := ParseDTCGJSON("pk", []byte("{")); err == nil {
		t.Fatal("ParseDTCGJSON() should reject invalid JSON")
	}
	if _, err := ParseDTCGJSON("pk", []byte(`{"color":{"brand":{"$value":"#2563eb"}}} {}`)); err == nil {
		t.Fatal("ParseDTCGJSON() should reject trailing JSON documents")
	}
}

func TestResolveRejectsInvalidGroupExtends(t *testing.T) {
	t.Parallel()

	if _, err := Resolve(Set{
		Name:   "pk",
		Values: map[string]Value{"color.brand.primary": "#2563eb"},
		Groups: map[string]Group{
			"color.accent": {Extends: "{color.missing}"},
		},
	}); err == nil {
		t.Fatal("Resolve() should reject missing group extends target")
	}
	if _, err := Resolve(Set{
		Name:   "pk",
		Values: map[string]Value{"color.brand.primary": "#2563eb"},
		Groups: map[string]Group{
			"color.a": {Extends: "{color.b}"},
			"color.b": {Extends: "{color.a}"},
		},
	}); err == nil {
		t.Fatal("Resolve() should reject group extends cycles")
	}
}

func TestNormalizeAllowsGroupOnlyOverlay(t *testing.T) {
	t.Parallel()

	set, err := (Set{
		Name: "pk",
		Groups: map[string]Group{
			"color.accent": {Extends: "{color.brand}"},
		},
	}).Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if set.Groups["color.accent"].Extends != "{color.brand}" {
		t.Fatalf("Normalize() groups = %#v", set.Groups)
	}
}

func TestMergePreservesDeprecatedAndMetadata(t *testing.T) {
	t.Parallel()

	merged, err := Merge(
		Set{
			Name: "pk",
			Values: map[string]Value{
				"color.old": "#111111",
			},
			Deprecated: map[string]any{
				"color.old": "Use color.new",
			},
			Metadata: map[string]any{"owner": "core"},
		},
		Set{
			Name: "pk",
			Values: map[string]Value{
				"color.new": "#222222",
			},
			Metadata: map[string]any{"stage": "module"},
		},
	)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if merged.Deprecated["color.old"] != "Use color.new" {
		t.Fatalf("Merge() deprecated = %#v", merged.Deprecated)
	}
	if merged.Metadata["owner"] != "core" || merged.Metadata["stage"] != "module" {
		t.Fatalf("Merge() metadata = %#v", merged.Metadata)
	}
}

func TestReportError(t *testing.T) {
	t.Parallel()

	report := Validate(Set{})
	if report.Error() == "" {
		t.Fatal("Report.Error() returned empty string")
	}
	var empty Report
	if empty.Error() == "" {
		t.Fatal("empty Report.Error() returned empty string")
	}
}
