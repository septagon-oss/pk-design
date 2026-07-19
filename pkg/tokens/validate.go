package tokens

// validate.go owns structural validation and canonical normalization for token
// sets before composition, resolution, or export.
//
// Implements: REQ-011.
// Per: ADR-0029 (file purpose declaration); C-10 (shared builders return errors).
// Discipline: C-14.

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalize(s Set) (Set, Report) {
	var report Report
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = "platformkit"
	}
	if !validIdentifier(name) {
		report.Add(IssueInvalidName, SeverityError, name, fmt.Sprintf("token set name %q must use letters, numbers, dot, underscore, or hyphen", name))
	}
	if len(s.Values) == 0 && len(s.Groups) == 0 {
		report.Add(IssueEmptySet, SeverityError, name, fmt.Sprintf("token set %q must declare at least one token", name))
	}

	out := Set{
		Name:         name,
		Version:      strings.TrimSpace(s.Version),
		Values:       make(map[string]Value, len(s.Values)),
		Types:        map[string]Type{},
		Descriptions: map[string]string{},
		Extensions:   map[string]map[string]any{},
		Groups:       map[string]Group{},
		Metadata:     normalizeAnyMap(s.Metadata),
	}
	for _, rawPath := range sortedMapKeys(s.Values) {
		rawValue := s.Values[rawPath]
		path := strings.TrimSpace(rawPath)
		if err := validatePath(path, true); err != nil {
			report.Add(IssueInvalidPath, SeverityError, path, err.Error())
			continue
		}
		if invalidEmptyValue(rawValue) {
			report.Add(IssueEmptyValue, SeverityError, path, fmt.Sprintf("token %q has empty value", path))
			continue
		}
		if _, exists := out.Values[path]; exists {
			report.Add(IssueDuplicatePath, SeverityError, path, fmt.Sprintf("duplicate token %q after normalization", path))
			continue
		}
		out.Values[path] = deepCopyValue(rawValue)
	}
	report.Issues = append(report.Issues, pathConflictIssues(out.Values)...)
	normalizeTypes(&report, out.Values, out.Types, s.Types, "token type")
	normalizeDescriptions(&report, out.Values, out.Descriptions, s.Descriptions, "token description")
	normalizeExtensions(&report, out.Values, out.Extensions, s.Extensions, "token extension")
	for _, rawPath := range sortedMapKeys(s.Groups) {
		rawGroup := s.Groups[rawPath]
		path := strings.TrimSpace(rawPath)
		if err := validatePath(path, false); err != nil {
			report.Add(IssueInvalidGroup, SeverityError, path, fmt.Sprintf("group path: %v", err))
			continue
		}
		if _, exists := out.Groups[path]; exists {
			report.Add(IssueInvalidGroup, SeverityError, path, fmt.Sprintf("duplicate group %q after normalization", path))
			continue
		}
		group := Group{
			Type:        Type(strings.TrimSpace(string(rawGroup.Type))),
			Description: strings.TrimSpace(rawGroup.Description),
			Extends:     strings.TrimSpace(rawGroup.Extends),
			Extensions:  normalizeAnyMap(rawGroup.Extensions),
		}
		if group.Type != "" && !validType(group.Type) {
			report.Add(IssueInvalidType, SeverityError, path, fmt.Sprintf("group %q type %q is not supported", path, group.Type))
		}
		if group.Extends != "" {
			if ref, ok := ParseReference(group.Extends); !ok {
				report.Add(IssueInvalidReference, SeverityError, path, fmt.Sprintf("group %q extends invalid reference %q", path, group.Extends))
			} else if ref == path {
				report.Add(IssueReferenceCycle, SeverityError, path, fmt.Sprintf("group %q cannot extend itself", path))
			}
		}
		out.Groups[path] = group
	}
	trimEmptyOptionalMaps(&out)
	return out, report
}

func normalizeTypes(report *Report, values map[string]Value, out map[string]Type, in map[string]Type, label string) {
	for _, rawPath := range sortedMapKeys(in) {
		tokenType := in[rawPath]
		path := strings.TrimSpace(rawPath)
		if err := validatePath(path, true); err != nil {
			report.Add(IssueInvalidPath, SeverityError, path, fmt.Sprintf("%s path: %v", label, err))
			continue
		}
		if _, exists := values[path]; !exists {
			report.Add(IssueUnknownMetadataPath, SeverityError, path, fmt.Sprintf("%s references unknown token %q", label, path))
			continue
		}
		if _, exists := out[path]; exists {
			report.Add(IssueDuplicatePath, SeverityError, path, fmt.Sprintf("duplicate %s for %q after normalization", label, path))
			continue
		}
		normalizedType := Type(strings.TrimSpace(string(tokenType)))
		if normalizedType != "" && !validType(normalizedType) {
			report.Add(IssueInvalidType, SeverityError, path, fmt.Sprintf("token %q type %q is not supported", path, normalizedType))
		}
		if normalizedType != "" {
			out[path] = normalizedType
		}
	}
}

func normalizeDescriptions(report *Report, values map[string]Value, out map[string]string, in map[string]string, label string) {
	for _, rawPath := range sortedMapKeys(in) {
		description := in[rawPath]
		path := strings.TrimSpace(rawPath)
		if err := validatePath(path, true); err != nil {
			report.Add(IssueInvalidPath, SeverityError, path, fmt.Sprintf("%s path: %v", label, err))
			continue
		}
		if _, exists := values[path]; !exists {
			report.Add(IssueUnknownMetadataPath, SeverityError, path, fmt.Sprintf("%s references unknown token %q", label, path))
			continue
		}
		if _, exists := out[path]; exists {
			report.Add(IssueDuplicatePath, SeverityError, path, fmt.Sprintf("duplicate %s for %q after normalization", label, path))
			continue
		}
		if description = strings.TrimSpace(description); description != "" {
			out[path] = description
		}
	}
}

func normalizeExtensions(report *Report, values map[string]Value, out map[string]map[string]any, in map[string]map[string]any, label string) {
	for _, rawPath := range sortedMapKeys(in) {
		extensions := in[rawPath]
		path := strings.TrimSpace(rawPath)
		if err := validatePath(path, true); err != nil {
			report.Add(IssueInvalidPath, SeverityError, path, fmt.Sprintf("%s path: %v", label, err))
			continue
		}
		if _, exists := values[path]; !exists {
			report.Add(IssueUnknownMetadataPath, SeverityError, path, fmt.Sprintf("%s references unknown token %q", label, path))
			continue
		}
		if _, exists := out[path]; exists {
			report.Add(IssueDuplicatePath, SeverityError, path, fmt.Sprintf("duplicate %s for %q after normalization", label, path))
			continue
		}
		if ext := normalizeAnyMap(extensions); len(ext) > 0 {
			out[path] = ext
		}
	}
}

func trimEmptyOptionalMaps(out *Set) {
	if len(out.Types) == 0 {
		out.Types = nil
	}
	if len(out.Descriptions) == 0 {
		out.Descriptions = nil
	}
	if len(out.Extensions) == 0 {
		out.Extensions = nil
	}
	if len(out.Groups) == 0 {
		out.Groups = nil
	}
	if len(out.Metadata) == 0 {
		out.Metadata = nil
	}
}

func validatePath(path string, allowRoot bool) error {
	if path == "" {
		return fmt.Errorf("token path is required")
	}
	if strings.ContainsAny(path, " \t\n\r") {
		return fmt.Errorf("token path %q must not contain whitespace", path)
	}
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		if segment == rootSegment {
			if !allowRoot {
				return fmt.Errorf("path %q must not contain %s", path, rootSegment)
			}
			if i != len(segments)-1 {
				return fmt.Errorf("path %q can only use %s as the final segment", path, rootSegment)
			}
			if len(segments) == 1 {
				return fmt.Errorf("path %q cannot be only %s", path, rootSegment)
			}
			continue
		}
		if !validIdentifier(segment) {
			return fmt.Errorf("token path %q contains invalid segment %q", path, segment)
		}
	}
	return nil
}

func pathConflictIssues(values map[string]Value) []Issue {
	keys := sortedValueKeys(values)
	var issues []Issue
	for i := 0; i < len(keys)-1; i++ {
		if strings.HasPrefix(keys[i+1], keys[i]+".") {
			issues = append(issues, Issue{
				Code:     IssuePathConflict,
				Severity: SeverityError,
				Path:     keys[i],
				Message:  fmt.Sprintf("token path %q conflicts with descendant token %q", keys[i], keys[i+1]),
			})
		}
	}
	return issues
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

func validType(tokenType Type) bool {
	switch tokenType {
	case TypeBorder, TypeColor, TypeCubicBezier, TypeDimension, TypeDuration, TypeFontFamily, TypeFontWeight, TypeGradient, TypeNumber, TypeShadow, TypeString, TypeStrokeStyle, TypeTransition, TypeTypography:
		return true
	default:
		return isExtensionToken(string(tokenType))
	}
}

func isExtensionToken(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "x.") || value == "x." {
		return false
	}
	return !strings.ContainsAny(value, " \t\n\r")
}

func invalidEmptyValue(value Value) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	if raw, ok := value.(json.RawMessage); ok {
		return len(raw) == 0
	}
	return false
}
