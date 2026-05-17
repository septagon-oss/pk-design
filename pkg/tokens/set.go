package tokens

// set.go owns the public Set operations: normalization, lookup, merge, and
// reference resolution entry points.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"
)

// Validate returns all structural issues that can be found without resolving
// aliases.
func Validate(set Set) Report {
	_, report := normalize(set)
	return report
}

// Normalize trims, validates, and defensively copies a token set.
func (s Set) Normalize() (Set, error) {
	normalized, report := normalize(s)
	if report.HasErrors() {
		return Set{}, report
	}
	return normalized, nil
}

// Keys returns normalized token paths in deterministic order.
func (s Set) Keys() ([]string, error) {
	normalized, err := s.Normalize()
	if err != nil {
		return nil, err
	}
	return sortedValueKeys(normalized.Values), nil
}

// Lookup returns one normalized token by path. If the token does not declare an
// explicit type, Lookup returns the nearest inherited group type.
func (s Set) Lookup(path string) (Token, bool, error) {
	normalized, err := s.Normalize()
	if err != nil {
		return Token{}, false, err
	}
	path = strings.TrimSpace(path)
	value, ok := normalized.Values[path]
	if !ok {
		return Token{}, false, nil
	}
	return Token{
		Path:        path,
		Type:        normalized.tokenType(path),
		Value:       deepCopyValue(value),
		Description: normalized.Descriptions[path],
		Extensions:  normalizeAnyMap(normalized.Extensions[path]),
		Deprecated:  deepCopyValue(normalized.Deprecated[path]),
	}, true, nil
}

// Merge overlays token sets from left to right. Overlay names may be empty or
// equal to the base name; this prevents accidental CSS namespace changes.
func Merge(base Set, overlays ...Set) (Set, error) {
	merged, err := base.Normalize()
	if err != nil {
		return Set{}, err
	}
	for _, overlay := range overlays {
		if strings.TrimSpace(overlay.Name) == "" {
			overlay.Name = merged.Name
		}
		next, err := overlay.Normalize()
		if err != nil {
			return Set{}, err
		}
		if next.Name != merged.Name {
			return Set{}, fmt.Errorf("token overlay name %q does not match base %q", next.Name, merged.Name)
		}
		if next.Version != "" {
			merged.Version = next.Version
		}
		merged.Metadata = mergeAnyMaps(merged.Metadata, next.Metadata)
		merged.Groups = mergeGroups(merged.Groups, next.Groups)
		for path, value := range next.Values {
			merged.Values[path] = deepCopyValue(value)
		}
		merged.Types = mergeTypeMaps(merged.Types, next.Types)
		merged.Descriptions = mergeStringMaps(merged.Descriptions, next.Descriptions)
		merged.Extensions = mergeNestedAnyMaps(merged.Extensions, next.Extensions)
		merged.Deprecated = mergeAnyMaps(merged.Deprecated, next.Deprecated)
	}
	return merged.Normalize()
}

// Resolve returns a normalized set with group $extends applied and aliases
// resolved. It fails on missing references and reference cycles.
func Resolve(set Set) (Set, error) {
	normalized, err := set.Normalize()
	if err != nil {
		return Set{}, err
	}
	expanded, err := applyGroupExtends(normalized)
	if err != nil {
		return Set{}, err
	}
	resolvedValues := make(map[string]Value, len(expanded.Values))
	for _, path := range sortedValueKeys(expanded.Values) {
		value, err := resolveTokenValue(expanded, path, nil)
		if err != nil {
			return Set{}, err
		}
		resolvedValues[path] = value
	}
	expanded.Values = resolvedValues
	return expanded.Normalize()
}
