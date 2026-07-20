package themes

// Validates: REQ-011.
// Per: ADR-0031.
// Discipline: C-14.
// themes_test.go validates theme normalization, layering, CSS export, and
// defensive-copy behavior for the OSS theme contract.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"slices"
	"strings"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func validTheme() Theme {
	return Theme{
		ID:      " light ",
		Name:    " Light ",
		Version: " 0.1.0 ",
		Extends: []string{"base", "brand", "base"},
		Tokens: tokens.Set{
			Name: " pk ",
			Values: map[string]tokens.Value{
				"color.surface.primary": "#ffffff",
				"color.text.primary":    "#111827",
			},
			Types: map[string]tokens.Type{
				"color.surface.primary": tokens.TypeColor,
				"color.text.primary":    tokens.TypeColor,
			},
		},
		Metadata: map[string]string{" owner ": " design "},
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	theme := validTheme()
	normalized, err := theme.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.ID != "light" || normalized.Name != "Light" || normalized.Version != "0.1.0" {
		t.Fatalf("Normalize() identity = %#v", normalized)
	}
	if !slices.Equal(normalized.Extends, []string{"base", "brand"}) {
		t.Fatalf("Normalize() Extends = %v", normalized.Extends)
	}
	if normalized.Tokens.Name != "pk" || normalized.Tokens.Values["color.text.primary"] != "#111827" {
		t.Fatalf("Normalize() Tokens = %#v", normalized.Tokens)
	}
	if normalized.Metadata["owner"] != "design" {
		t.Fatalf("Normalize() Metadata = %#v", normalized.Metadata)
	}

	normalized.Tokens.Values["color.text.primary"] = "#000000"
	again, err := theme.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if again.Tokens.Values["color.text.primary"] != "#111827" {
		t.Fatalf("Normalize() returned aliased token values: %#v", again.Tokens.Values)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	t.Parallel()

	tests := []Theme{
		{},
		{ID: "bad id", Tokens: validTheme().Tokens},
		{ID: "light", Extends: []string{"light"}, Tokens: validTheme().Tokens},
		{ID: "light", Extends: []string{"bad id"}, Tokens: validTheme().Tokens},
		{ID: "light", Tokens: tokens.Set{Name: "pk"}},
	}
	for _, test := range tests {
		if _, err := test.Normalize(); err == nil {
			t.Fatalf("Normalize(%#v) should fail", test)
		}
	}
}

func TestLayer(t *testing.T) {
	t.Parallel()

	base := validTheme()
	overlay := Theme{
		Tokens: tokens.Set{
			Values: map[string]tokens.Value{
				"color.text.primary": "#0f172a",
				"radius.sm":          "2px",
			},
			Types: map[string]tokens.Type{
				"color.text.primary": tokens.TypeColor,
				"radius.sm":          tokens.TypeDimension,
			},
		},
		Extends:  []string{"tenant"},
		Metadata: map[string]string{"tier": "client"},
	}

	layered, err := Layer(base, overlay)
	if err != nil {
		t.Fatalf("Layer() error = %v", err)
	}
	if layered.ID != "light" || layered.Tokens.Values["color.text.primary"] != "#0f172a" {
		t.Fatalf("Layer() = %#v", layered)
	}
	if layered.Tokens.Values["radius.sm"] != "2px" || layered.Metadata["tier"] != "client" {
		t.Fatalf("Layer() did not apply overlay: %#v", layered)
	}
	if !slices.Equal(layered.Extends, []string{"base", "brand", "tenant"}) {
		t.Fatalf("Layer() Extends = %v", layered.Extends)
	}

	if _, err := Layer(base, Theme{ID: "dark", Tokens: overlay.Tokens}); err == nil {
		t.Fatal("Layer() should reject mismatched overlay IDs")
	}
}

func TestCSSVars(t *testing.T) {
	t.Parallel()

	css, err := CSSVars(validTheme())
	if err != nil {
		t.Fatalf("CSSVars() error = %v", err)
	}
	if !strings.Contains(css, "--pk-color-text-primary: #111827;") {
		t.Fatalf("CSSVars() =\n%s", css)
	}
}

func TestResolveModeAppliesGlobalAndMatchingLayers(t *testing.T) {
	t.Parallel()

	resolved, err := ResolveMode(
		"dark",
		TokenLayer{
			ID:   "base",
			Kind: LayerBase,
			Tokens: tokens.Set{
				Name: "pk",
				Values: map[string]tokens.Value{
					"color.brand.primary": "#2563eb",
				},
				Groups: map[string]tokens.Group{"color.brand": {Type: tokens.TypeColor}},
			},
		},
		TokenLayer{
			ID:   "semantic",
			Kind: LayerSemantic,
			Tokens: tokens.Set{
				Name: "pk",
				Values: map[string]tokens.Value{
					"color.action.primary": "{color.brand.primary}",
				},
				Types: map[string]tokens.Type{"color.action.primary": tokens.TypeColor},
			},
		},
		TokenLayer{
			ID:   "client-light",
			Kind: LayerClient,
			Mode: "light",
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#60a5fa"},
			},
		},
		TokenLayer{
			ID:   "client-dark",
			Kind: LayerClient,
			Mode: "dark",
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#1d4ed8"},
			},
		},
	)
	if err != nil {
		t.Fatalf("ResolveMode() error = %v", err)
	}
	if resolved.Values["color.action.primary"] != "#1d4ed8" {
		t.Fatalf("ResolveMode() values = %#v", resolved.Values)
	}
}

func TestResolveLayersRejectsInvalidLayer(t *testing.T) {
	t.Parallel()

	if _, err := ResolveLayers(TokenLayer{}); err == nil {
		t.Fatal("ResolveLayers() should reject invalid layer")
	}
	if _, err := ResolveMode("missing"); err == nil {
		t.Fatal("ResolveMode() should require at least one applicable layer")
	}
}

func TestStackCanonicalOrderAndDefensiveCopies(t *testing.T) {
	t.Parallel()

	stack, err := NewStack(
		TokenLayer{
			ID:   "client",
			Kind: LayerClient,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#1d4ed8"},
			},
		},
		TokenLayer{
			ID:   "semantic",
			Kind: LayerSemantic,
			Tokens: tokens.Set{
				Name: "pk",
				Values: map[string]tokens.Value{
					"color.action.primary": "{color.brand.primary}",
				},
				Types: map[string]tokens.Type{"color.action.primary": tokens.TypeColor},
			},
		},
		TokenLayer{
			ID:   "base",
			Kind: LayerBase,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#2563eb"},
				Groups: map[string]tokens.Group{"color.brand": {Type: tokens.TypeColor}},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewStack() error = %v", err)
	}
	layers := stack.Layers()
	if got := []string{layers[0].ID, layers[1].ID, layers[2].ID}; !slices.Equal(got, []string{"base", "semantic", "client"}) {
		t.Fatalf("Layers() order = %v", got)
	}

	resolved, err := stack.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Values["color.action.primary"] != "#1d4ed8" {
		t.Fatalf("Resolve() values = %#v", resolved.Values)
	}

	layers[2].Tokens.Values["color.brand.primary"] = "#000000"
	again, err := stack.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if again.Values["color.action.primary"] != "#1d4ed8" {
		t.Fatalf("Layers() returned aliased token data: %#v", again.Values)
	}
}

func TestStackUsesFullCanonicalLayerOrder(t *testing.T) {
	t.Parallel()

	stack, err := NewStack(
		stackOrderLayer("accessibility", LayerAccessibility),
		stackOrderLayer("tenant", LayerTenant),
		stackOrderLayer("base", LayerBase),
		stackOrderLayer("platform", LayerPlatform),
		stackOrderLayer("client", LayerClient),
		stackOrderLayer("app", LayerApp),
		stackOrderLayer("module", LayerModule),
		stackOrderLayer("semantic", LayerSemantic),
		stackOrderLayer("primitive", LayerPrimitive),
	)
	if err != nil {
		t.Fatalf("NewStack() error = %v", err)
	}
	layers := stack.Layers()
	got := make([]string, len(layers))
	for i, layer := range layers {
		got[i] = layer.ID
	}
	want := []string{"base", "primitive", "semantic", "module", "app", "client", "tenant", "platform", "accessibility"}
	if !slices.Equal(got, want) {
		t.Fatalf("Layers() order = %v; want %v", got, want)
	}
}

func TestStackResolveMode(t *testing.T) {
	t.Parallel()

	stack, err := NewStack(
		TokenLayer{
			ID:   "base",
			Kind: LayerBase,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#2563eb"},
				Groups: map[string]tokens.Group{"color.brand": {Type: tokens.TypeColor}},
			},
		},
		TokenLayer{
			ID:   "tenant-light",
			Kind: LayerTenant,
			Mode: "light",
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#60a5fa"},
			},
		},
		TokenLayer{
			ID:   "tenant-dark",
			Kind: LayerTenant,
			Mode: "dark",
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#1d4ed8"},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewStack() error = %v", err)
	}
	resolved, err := stack.ResolveMode("dark")
	if err != nil {
		t.Fatalf("ResolveMode() error = %v", err)
	}
	if resolved.Values["color.brand.primary"] != "#1d4ed8" {
		t.Fatalf("ResolveMode() values = %#v", resolved.Values)
	}
}

func TestStackPreservesSameKindContributionOrder(t *testing.T) {
	t.Parallel()

	stack, err := NewStack(
		TokenLayer{
			ID:   "base",
			Kind: LayerBase,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"space.2": "0.5rem"},
				Types:  map[string]tokens.Type{"space.2": tokens.TypeDimension},
			},
		},
		TokenLayer{
			ID:   "module-a",
			Kind: LayerModule,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"space.2": "0.75rem"},
			},
		},
		TokenLayer{
			ID:   "module-b",
			Kind: LayerModule,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"space.2": "1rem"},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewStack() error = %v", err)
	}
	resolved, err := stack.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Values["space.2"] != "1rem" {
		t.Fatalf("same-kind layer order was not preserved: %#v", resolved.Values)
	}

	var nilStack *Stack
	if err := nilStack.Add(TokenLayer{}); err == nil {
		t.Fatal("nil Stack.Add() should fail")
	}
}

func stackOrderLayer(id string, kind LayerKind) TokenLayer {
	return TokenLayer{
		ID:   id,
		Kind: kind,
		Tokens: tokens.Set{
			Name:   "pk",
			Values: map[string]tokens.Value{"layer." + id: "1"},
		},
	}
}
