package themes

// stack.go owns canonical token-layer stack composition. ResolveLayers remains
// available for explicit caller order; Stack gives modules and apps a stable
// default order for extension-heavy PlatformKit surfaces.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// Stack is a canonical PlatformKit token-layer stack. Layers are ordered by
// LayerKind while preserving contribution order within the same kind.
type Stack struct {
	layers []TokenLayer
}

// NewStack validates and canonicalizes layers for deterministic design
// composition.
func NewStack(layers ...TokenLayer) (Stack, error) {
	var stack Stack
	for _, layer := range layers {
		if err := stack.Add(layer); err != nil {
			return Stack{}, err
		}
	}
	return stack, nil
}

// Add validates a layer, snapshots it, and places it in canonical stack order.
func (s *Stack) Add(layer TokenLayer) error {
	if s == nil {
		return fmt.Errorf("theme stack is nil")
	}
	normalized, err := normalizeLayer(layer)
	if err != nil {
		return err
	}
	s.layers = append(s.layers, copyTokenLayer(normalized))
	sortTokenLayers(s.layers)
	return nil
}

// Layers returns defensive copies of the stack layers in canonical order.
func (s Stack) Layers() []TokenLayer {
	if len(s.layers) == 0 {
		return nil
	}
	out := make([]TokenLayer, len(s.layers))
	for i, layer := range s.layers {
		out[i] = copyTokenLayer(layer)
	}
	return out
}

// Resolve applies all stack layers and resolves aliases.
func (s Stack) Resolve() (tokens.Set, error) {
	return ResolveLayers(s.layers...)
}

// ResolveMode applies global layers plus layers matching the requested mode,
// then resolves aliases.
func (s Stack) ResolveMode(mode string) (tokens.Set, error) {
	return ResolveMode(mode, s.layers...)
}

func sortTokenLayers(layers []TokenLayer) {
	slices.SortStableFunc(layers, func(a, b TokenLayer) int {
		return cmp.Compare(layerKindRank(a.Kind), layerKindRank(b.Kind))
	})
}

func layerKindRank(kind LayerKind) int {
	switch kind {
	case LayerBase:
		return 10
	case LayerPrimitive:
		return 20
	case LayerSemantic:
		return 30
	case LayerModule:
		return 40
	case LayerApp:
		return 50
	case LayerClient:
		return 60
	case LayerTenant:
		return 70
	case LayerPlatform:
		return 80
	case LayerAccessibility:
		return 90
	default:
		return 100
	}
}

func copyTokenLayer(layer TokenLayer) TokenLayer {
	return TokenLayer{
		ID:       layer.ID,
		Kind:     layer.Kind,
		Mode:     layer.Mode,
		Tokens:   copyTokenSet(layer.Tokens),
		Metadata: maps.Clone(layer.Metadata),
	}
}

func copyTokenSet(set tokens.Set) tokens.Set {
	return tokens.Set{
		Name:         set.Name,
		Version:      set.Version,
		Values:       copyTokenValues(set.Values),
		Types:        maps.Clone(set.Types),
		Descriptions: maps.Clone(set.Descriptions),
		Extensions:   copyNestedAnyMap(set.Extensions),
		Deprecated:   copyAnyMap(set.Deprecated),
		Groups:       copyGroups(set.Groups),
		Metadata:     copyAnyMap(set.Metadata),
	}
}

func copyTokenValues(values map[string]tokens.Value) map[string]tokens.Value {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]tokens.Value, len(values))
	for key, value := range values {
		out[key] = tokens.CopyValue(value)
	}
	return out
}

func copyAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = tokens.CopyValue(value)
	}
	return out
}

func copyNestedAnyMap(values map[string]map[string]any) map[string]map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(values))
	for key, value := range values {
		out[key] = copyAnyMap(value)
	}
	return out
}

func copyGroups(values map[string]tokens.Group) map[string]tokens.Group {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]tokens.Group, len(values))
	for key, value := range values {
		out[key] = tokens.Group{
			Type:        value.Type,
			Description: value.Description,
			Extends:     value.Extends,
			Extensions:  copyAnyMap(value.Extensions),
		}
	}
	return out
}
