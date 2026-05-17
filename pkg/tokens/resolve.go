package tokens

// resolve.go owns group inheritance and token-reference resolution. Resolution
// returns a new normalized set and never mutates the caller's input.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"
)

func applyGroupExtends(set Set) (Set, error) {
	expanded, err := set.Normalize()
	if err != nil {
		return Set{}, err
	}
	for _, groupPath := range sortedGroupKeys(expanded.Groups) {
		if _, err := expandGroup(&expanded, groupPath, nil); err != nil {
			return Set{}, err
		}
	}
	return expanded.Normalize()
}

func expandGroup(set *Set, groupPath string, stack []string) (struct{}, error) {
	group, ok := set.Groups[groupPath]
	if !ok || group.Extends == "" {
		return struct{}{}, nil
	}
	if contains(stack, groupPath) {
		return struct{}{}, Report{Issues: []Issue{{
			Code:     IssueReferenceCycle,
			Severity: SeverityError,
			Path:     groupPath,
			Message:  "group extends cycle: " + strings.Join(append(stack, groupPath), " -> "),
		}}}
	}
	sourceGroupPath, ok := ParseReference(group.Extends)
	if !ok {
		return struct{}{}, Report{Issues: []Issue{{
			Code:     IssueInvalidReference,
			Severity: SeverityError,
			Path:     groupPath,
			Message:  fmt.Sprintf("group %q extends invalid reference %q", groupPath, group.Extends),
		}}}
	}
	if sourceGroup, ok := set.Groups[sourceGroupPath]; ok && sourceGroup.Extends != "" {
		if _, err := expandGroup(set, sourceGroupPath, append(stack, groupPath)); err != nil {
			return struct{}{}, err
		}
	}
	sourcePrefix := sourceGroupPath + "."
	targetPrefix := groupPath + "."
	copied := false
	for _, sourceToken := range sortedValueKeys(set.Values) {
		if !strings.HasPrefix(sourceToken, sourcePrefix) {
			continue
		}
		suffix, _ := strings.CutPrefix(sourceToken, sourcePrefix)
		targetToken := targetPrefix + suffix
		if _, exists := set.Values[targetToken]; exists {
			continue
		}
		copyTokenMetadata(set, sourceToken, targetToken)
		copied = true
	}
	if !copied && !hasGroup(set, sourceGroupPath) {
		return struct{}{}, Report{Issues: []Issue{{
			Code:     IssueMissingReference,
			Severity: SeverityError,
			Path:     groupPath,
			Message:  fmt.Sprintf("group %q extends unknown group %q", groupPath, sourceGroupPath),
		}}}
	}
	return struct{}{}, nil
}

func copyTokenMetadata(set *Set, sourceToken, targetToken string) {
	set.Values[targetToken] = deepCopyValue(set.Values[sourceToken])
	if value, ok := set.Types[sourceToken]; ok {
		if set.Types == nil {
			set.Types = map[string]Type{}
		}
		set.Types[targetToken] = value
	}
	if value, ok := set.Descriptions[sourceToken]; ok {
		if set.Descriptions == nil {
			set.Descriptions = map[string]string{}
		}
		set.Descriptions[targetToken] = value
	}
	if value, ok := set.Extensions[sourceToken]; ok {
		if set.Extensions == nil {
			set.Extensions = map[string]map[string]any{}
		}
		set.Extensions[targetToken] = normalizeAnyMap(value)
	}
	if value, ok := set.Deprecated[sourceToken]; ok {
		if set.Deprecated == nil {
			set.Deprecated = map[string]any{}
		}
		set.Deprecated[targetToken] = deepCopyValue(value)
	}
}

func hasGroup(set *Set, groupPath string) bool {
	if _, exists := set.Groups[groupPath]; exists {
		return true
	}
	prefix := groupPath + "."
	for path := range set.Values {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func resolveTokenValue(set Set, path string, stack []string) (Value, error) {
	if contains(stack, path) {
		return nil, Report{Issues: []Issue{{
			Code:     IssueReferenceCycle,
			Severity: SeverityError,
			Path:     path,
			Message:  "token reference cycle: " + strings.Join(append(stack, path), " -> "),
		}}}
	}
	value, ok := set.Values[path]
	if !ok {
		return nil, Report{Issues: []Issue{{
			Code:     IssueMissingReference,
			Severity: SeverityError,
			Path:     path,
			Message:  fmt.Sprintf("token %q does not exist", path),
		}}}
	}
	return resolveAnyValue(set, value, append(stack, path))
}

func resolveAnyValue(set Set, value Value, stack []string) (Value, error) {
	if ref, ok := value.(string); ok {
		if path, isReference := ParseReference(ref); isReference {
			resolved, err := resolveTokenValue(set, path, stack)
			if err != nil {
				return nil, err
			}
			return resolved, nil
		}
		return ref, nil
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			resolved, err := resolveAnyValue(set, child, stack)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			resolved, err := resolveAnyValue(set, child, stack)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return deepCopyValue(value), nil
	}
}

// ParseReference returns the token path from a DTCG alias such as
// "{color.brand.primary}".
func ParseReference(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 3 || !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return "", false
	}
	path, _ := strings.CutPrefix(value, "{")
	path, _ = strings.CutSuffix(path, "}")
	path = strings.TrimSpace(path)
	if err := validatePath(path, true); err != nil {
		return "", false
	}
	return path, true
}

func (s Set) tokenType(path string) Type {
	if tokenType := s.Types[path]; tokenType != "" {
		return tokenType
	}
	segments := strings.Split(path, ".")
	if len(segments) > 0 && segments[len(segments)-1] == rootSegment {
		segments = segments[:len(segments)-1]
	}
	for i := len(segments); i > 0; i-- {
		groupPath := strings.Join(segments[:i], ".")
		if group := s.Groups[groupPath]; group.Type != "" {
			return group.Type
		}
	}
	return ""
}
