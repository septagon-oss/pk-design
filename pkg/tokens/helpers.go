package tokens

// helpers.go owns deterministic copy, merge, and sorting helpers shared across
// the token package internals.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
)

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

func normalizeAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key != "" && value != nil {
			out[key] = deepCopyValue(value)
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
	for key, value := range normalizeStringMap(overlay) {
		out[key] = value
	}
	return out
}

func mergeAnyMaps(base, overlay map[string]any) map[string]any {
	out := normalizeAnyMap(base)
	if len(overlay) == 0 {
		return out
	}
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range normalizeAnyMap(overlay) {
		out[key] = value
	}
	return out
}

func mergeTypeMaps(base, overlay map[string]Type) map[string]Type {
	out := make(map[string]Type, len(base)+len(overlay))
	for key, value := range base {
		if value != "" {
			out[key] = value
		}
	}
	for key, value := range overlay {
		if value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeNestedAnyMaps(base, overlay map[string]map[string]any) map[string]map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := map[string]map[string]any{}
	for key, value := range base {
		if normalized := normalizeAnyMap(value); len(normalized) > 0 {
			out[key] = normalized
		}
	}
	for key, value := range overlay {
		if normalized := normalizeAnyMap(value); len(normalized) > 0 {
			out[key] = normalized
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeGroups(base, overlay map[string]Group) map[string]Group {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := map[string]Group{}
	for key, value := range base {
		out[key] = copyGroup(value)
	}
	for key, value := range overlay {
		out[key] = copyGroup(value)
	}
	return out
}

func copyGroup(value Group) Group {
	return Group{
		Type:        value.Type,
		Description: value.Description,
		Extends:     value.Extends,
		Extensions:  normalizeAnyMap(value.Extensions),
	}
}

func deepCopyValue(value Value) Value {
	switch typed := value.(type) {
	case nil:
		return nil
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = deepCopyValue(child)
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = child
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = deepCopyValue(child)
		}
		return out
	case []string:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = child
		}
		return out
	default:
		return typed
	}
}

func sortedValueKeys(values map[string]Value) []string {
	return slices.Sorted(maps.Keys(values))
}

func sortedGroupKeys(values map[string]Group) []string {
	return slices.Sorted(maps.Keys(values))
}

func sortedObjectKeys(values map[string]any) []string {
	return slices.Sorted(maps.Keys(values))
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}
