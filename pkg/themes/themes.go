// Package themes defines renderer-neutral PlatformKit theme overlays.
package themes

// themes.go owns the OSS theme contract that layers semantic token sets without
// coupling design packages to a frontend renderer, Figma adapter, or build tool.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// Theme is a named token layer. Themes are intentionally token-first; concrete
// renderers decide whether to emit CSS, Tailwind config, native variables, or
// another target format.
type Theme struct {
	ID          string
	Name        string
	Version     string
	Description string
	Extends     []string
	Tokens      tokens.Set
	Metadata    map[string]string
}

// LayerKind identifies the semantic position of a token layer in a PlatformKit
// design stack. Adapters can add richer behavior outside core, but these names
// describe the stable composition contract.
type LayerKind string

// Layer kinds ordered from the most foundational to the most specific override
// in a PlatformKit design stack.
const (
	// LayerBase is the foundational reset/base layer.
	LayerBase LayerKind = "base"
	// LayerPrimitive holds raw primitive tokens (e.g. color ramps, scales).
	LayerPrimitive LayerKind = "primitive"
	// LayerSemantic maps primitives to semantic roles.
	LayerSemantic LayerKind = "semantic"
	// LayerModule contributes module-scoped token overrides.
	LayerModule LayerKind = "module"
	// LayerApp contributes application-level overrides.
	LayerApp LayerKind = "app"
	// LayerClient contributes client/brand-level overrides.
	LayerClient LayerKind = "client"
	// LayerTenant contributes tenant-level overrides.
	LayerTenant LayerKind = "tenant"
	// LayerAccessibility contributes accessibility-driven overrides.
	LayerAccessibility LayerKind = "accessibility"
	// LayerPlatform contributes platform-wide overrides applied last.
	LayerPlatform LayerKind = "platform"
)

// TokenLayer is one token contribution in an ordered theme stack. Mode is optional;
// layers without a mode apply to every mode.
type TokenLayer struct {
	ID       string
	Kind     LayerKind
	Mode     string
	Tokens   tokens.Set
	Metadata map[string]string
}

// Normalize trims, validates, and defensively copies a theme.
func (t Theme) Normalize() (Theme, error) {
	t.ID = strings.TrimSpace(t.ID)
	if t.ID == "" {
		return Theme{}, fmt.Errorf("theme ID is required")
	}
	if !validIdentifier(t.ID) {
		return Theme{}, fmt.Errorf("theme ID %q must use letters, numbers, dot, underscore, or hyphen", t.ID)
	}
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		t.Name = t.ID
	}
	t.Version = strings.TrimSpace(t.Version)
	t.Description = strings.TrimSpace(t.Description)
	var err error
	if t.Extends, err = normalizeExtends(t.ID, t.Extends); err != nil {
		return Theme{}, fmt.Errorf("theme %q extends: %w", t.ID, err)
	}
	t.Metadata = normalizeStringMap(t.Metadata)
	if strings.TrimSpace(t.Tokens.Name) == "" {
		t.Tokens.Name = "pk"
	}
	normalizedTokens, err := t.Tokens.Normalize()
	if err != nil {
		return Theme{}, fmt.Errorf("theme %q tokens: %w", t.ID, err)
	}
	t.Tokens = normalizedTokens
	return t, nil
}

// Layer applies overlays to base and returns a normalized theme. Overlay IDs may
// be empty or equal to the base ID; cross-theme layering should be represented
// through Extends and resolved by the caller's catalog.
func Layer(base Theme, overlays ...Theme) (Theme, error) {
	merged, err := base.Normalize()
	if err != nil {
		return Theme{}, err
	}
	for _, overlay := range overlays {
		if strings.TrimSpace(overlay.ID) == "" {
			overlay.ID = merged.ID
		}
		if strings.TrimSpace(overlay.Name) == "" {
			overlay.Name = merged.Name
		}
		if strings.TrimSpace(overlay.Tokens.Name) == "" {
			overlay.Tokens.Name = merged.Tokens.Name
		}
		next, err := overlay.Normalize()
		if err != nil {
			return Theme{}, err
		}
		if next.ID != merged.ID {
			return Theme{}, fmt.Errorf("theme overlay ID %q does not match base %q", next.ID, merged.ID)
		}
		if next.Version != "" {
			merged.Version = next.Version
		}
		if next.Description != "" {
			merged.Description = next.Description
		}
		merged.Extends, err = normalizeExtends(merged.ID, append(merged.Extends, next.Extends...))
		if err != nil {
			return Theme{}, fmt.Errorf("theme %q extends: %w", merged.ID, err)
		}
		merged.Metadata = mergeStringMaps(merged.Metadata, next.Metadata)
		merged.Tokens, err = tokens.Merge(merged.Tokens, next.Tokens)
		if err != nil {
			return Theme{}, fmt.Errorf("theme %q overlay tokens: %w", merged.ID, err)
		}
	}
	return merged, nil
}

// CSSVars renders the theme tokens as CSS custom properties.
func CSSVars(theme Theme) (string, error) {
	normalized, err := theme.Normalize()
	if err != nil {
		return "", err
	}
	return tokens.CSSVars(normalized.Tokens)
}

// ResolveLayers applies layers in caller-provided order and resolves token
// aliases and group inheritance.
func ResolveLayers(layers ...TokenLayer) (tokens.Set, error) {
	return ResolveMode("", layers...)
}

// ResolveMode applies global layers and layers matching the requested mode.
func ResolveMode(mode string, layers ...TokenLayer) (tokens.Set, error) {
	mode = strings.TrimSpace(mode)
	var merged tokens.Set
	applied := 0
	for _, layer := range layers {
		normalized, err := normalizeLayer(layer)
		if err != nil {
			return tokens.Set{}, err
		}
		if mode != "" && normalized.Mode != "" && normalized.Mode != mode {
			continue
		}
		if applied == 0 {
			merged = normalized.Tokens
		} else {
			merged, err = tokens.Merge(merged, normalized.Tokens)
			if err != nil {
				return tokens.Set{}, fmt.Errorf("theme layer %q: %w", normalized.ID, err)
			}
		}
		applied++
	}
	if applied == 0 {
		return tokens.Set{}, fmt.Errorf("at least one theme layer is required")
	}
	resolved, err := tokens.Resolve(merged)
	if err != nil {
		return tokens.Set{}, err
	}
	return resolved, nil
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func validLayerKind(kind LayerKind) bool {
	switch kind {
	case LayerBase, LayerPrimitive, LayerSemantic, LayerModule, LayerApp, LayerClient, LayerTenant, LayerAccessibility, LayerPlatform:
		return true
	default:
		return false
	}
}

func normalizeLayer(layer TokenLayer) (TokenLayer, error) {
	layer.ID = strings.TrimSpace(layer.ID)
	if layer.ID == "" {
		return TokenLayer{}, fmt.Errorf("theme layer ID is required")
	}
	if !validIdentifier(layer.ID) {
		return TokenLayer{}, fmt.Errorf("theme layer ID %q is invalid", layer.ID)
	}
	layer.Kind = LayerKind(strings.TrimSpace(string(layer.Kind)))
	if layer.Kind == "" {
		return TokenLayer{}, fmt.Errorf("theme layer %q kind is required", layer.ID)
	}
	if !validLayerKind(layer.Kind) {
		return TokenLayer{}, fmt.Errorf("theme layer %q kind %q is not supported", layer.ID, layer.Kind)
	}
	layer.Mode = strings.TrimSpace(layer.Mode)
	if layer.Mode != "" && !validIdentifier(layer.Mode) {
		return TokenLayer{}, fmt.Errorf("theme layer %q mode %q is invalid", layer.ID, layer.Mode)
	}
	layer.Metadata = normalizeStringMap(layer.Metadata)
	normalizedTokens, err := layer.Tokens.Normalize()
	if err != nil {
		return TokenLayer{}, fmt.Errorf("theme layer %q tokens: %w", layer.ID, err)
	}
	layer.Tokens = normalizedTokens
	return layer, nil
}

func normalizeExtends(themeID string, values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !validIdentifier(value) {
			return nil, fmt.Errorf("theme ID %q is invalid", value)
		}
		if value == themeID {
			return nil, fmt.Errorf("theme cannot extend itself")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out, nil
}

func normalizeStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeStringMaps(base, overlay map[string]string) map[string]string {
	out := normalizeStringMap(base)
	if len(overlay) == 0 {
		return out
	}
	if out == nil {
		out = map[string]string{}
	}
	maps.Copy(out, normalizeStringMap(overlay))
	return out
}
