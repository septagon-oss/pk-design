package tokens

// Implements: REQ-011.
// Per: ADR-0004.
// Discipline: C-14.
// dtcg.go owns lossless DTCG document import/export. The flat Set model stays
// small while this file handles nested document shape at the boundary.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DTCG renders the set as a nested Design Tokens Community Group document.
func DTCG(set Set) (map[string]any, error) {
	normalized, err := set.Normalize()
	if err != nil {
		return nil, err
	}
	document := map[string]any{}
	for _, groupPath := range sortedGroupKeys(normalized.Groups) {
		group := normalized.Groups[groupPath]
		cursor, err := ensureGroup(document, strings.Split(groupPath, "."))
		if err != nil {
			return nil, err
		}
		if group.Type != "" {
			cursor["$type"] = string(group.Type)
		}
		if group.Description != "" {
			cursor["$description"] = group.Description
		}
		if group.Extends != "" {
			cursor["$extends"] = group.Extends
		}
		if len(group.Extensions) > 0 {
			cursor["$extensions"] = normalizeAnyMap(group.Extensions)
		}
	}
	for _, path := range sortedValueKeys(normalized.Values) {
		segments := strings.Split(path, ".")
		cursor, err := ensureGroup(document, segments[:len(segments)-1])
		if err != nil {
			return nil, err
		}
		leaf := map[string]any{"$value": deepCopyValue(normalized.Values[path])}
		if tokenType := normalized.Types[path]; tokenType != "" {
			leaf["$type"] = string(tokenType)
		}
		if description := normalized.Descriptions[path]; description != "" {
			leaf["$description"] = description
		}
		if ext := normalized.Extensions[path]; len(ext) > 0 {
			leaf["$extensions"] = normalizeAnyMap(ext)
		}
		cursor[segments[len(segments)-1]] = leaf
	}
	return document, nil
}

// DTCGJSON renders the DTCG document as stable, indented JSON.
func DTCGJSON(set Set) ([]byte, error) {
	document, err := DTCG(set)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(document, "", "  ")
}

// ParseDTCGJSON parses a DTCG document into a normalized flat token set.
func ParseDTCGJSON(name string, data []byte) (Set, error) {
	var document map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return Set{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Set{}, fmt.Errorf("DTCG JSON must contain exactly one document")
		}
		return Set{}, err
	}
	return ParseDTCG(name, document)
}

// ParseDTCG parses a DTCG document into a normalized flat token set.
func ParseDTCG(name string, document map[string]any) (Set, error) {
	set := Set{
		Name:         name,
		Values:       map[string]Value{},
		Types:        map[string]Type{},
		Descriptions: map[string]string{},
		Extensions:   map[string]map[string]any{},
		Groups:       map[string]Group{},
	}
	if err := parseDTCGGroup(&set, nil, document, Group{}); err != nil {
		return Set{}, err
	}
	return set.Normalize()
}

func parseDTCGGroup(set *Set, path []string, object map[string]any, inherited Group) error {
	effective := inherited
	current := Group{}
	groupPath := strings.Join(path, ".")
	if _, exists := object["$deprecated"]; exists {
		return Report{Issues: []Issue{{
			Code:     IssueInvalidDTCG,
			Severity: SeverityError,
			Path:     groupPath,
			Message:  "$deprecated metadata is not supported; remove obsolete tokens",
		}}}
	}
	if tokenValue, isToken := object["$value"]; isToken {
		for key := range object {
			if !strings.HasPrefix(key, "$") {
				return Report{Issues: []Issue{{
					Code:     IssueInvalidDTCG,
					Severity: SeverityError,
					Path:     groupPath,
					Message:  "DTCG node cannot contain both $value and child groups",
				}}}
			}
		}
		set.Values[groupPath] = deepCopyValue(tokenValue)
		if tokenType, ok := stringField(object, "$type"); ok {
			set.Types[groupPath] = Type(tokenType)
		}
		if description, ok := stringField(object, "$description"); ok {
			set.Descriptions[groupPath] = description
		}
		if extensions, ok := mapField(object, "$extensions"); ok {
			set.Extensions[groupPath] = extensions
		}
		return nil
	}
	if len(path) > 0 {
		if tokenType, ok := stringField(object, "$type"); ok {
			current.Type = Type(tokenType)
			effective.Type = current.Type
		}
		if description, ok := stringField(object, "$description"); ok {
			current.Description = description
			effective.Description = current.Description
		}
		if extends, ok := stringField(object, "$extends"); ok {
			current.Extends = extends
			effective.Extends = current.Extends
		}
		if extensions, ok := mapField(object, "$extensions"); ok {
			current.Extensions = extensions
			effective.Extensions = current.Extensions
		}
		if current.Type != "" || current.Description != "" || current.Extends != "" || len(current.Extensions) > 0 {
			set.Groups[groupPath] = current
		}
	}
	for _, key := range sortedObjectKeys(object) {
		if strings.HasPrefix(key, "$") && key != rootSegment {
			continue
		}
		childObject, ok := object[key].(map[string]any)
		if !ok {
			return Report{Issues: []Issue{{
				Code:     IssueInvalidDTCG,
				Severity: SeverityError,
				Path:     strings.Join(append(path, key), "."),
				Message:  "DTCG child must be an object",
			}}}
		}
		if err := parseDTCGGroup(set, append(path, key), childObject, effective); err != nil {
			return err
		}
	}
	return nil
}

func stringField(object map[string]any, key string) (string, bool) {
	value, ok := object[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

func mapField(object map[string]any, key string) (map[string]any, bool) {
	value, ok := object[key]
	if !ok {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	return normalizeAnyMap(typed), true
}

func ensureGroup(document map[string]any, segments []string) (map[string]any, error) {
	cursor := document
	for _, segment := range segments {
		if segment == "" {
			return nil, fmt.Errorf("empty DTCG path segment")
		}
		next, ok := cursor[segment].(map[string]any)
		if !ok {
			next = map[string]any{}
			cursor[segment] = next
		}
		if _, token := next["$value"]; token {
			return nil, Report{Issues: []Issue{{
				Code:     IssueInvalidDTCG,
				Severity: SeverityError,
				Path:     strings.Join(segments, "."),
				Message:  "DTCG group path collides with token",
			}}}
		}
		cursor = next
	}
	return cursor, nil
}
