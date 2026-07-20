package tokens

// Implements: REQ-011.
// Per: ADR-0004.
// Discipline: C-14.
// css.go owns renderer-neutral CSS custom-property export for resolved token
// sets. It intentionally emits plain CSS data only; framework adapters belong
// outside pk-design core.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// CSSVars renders resolved tokens as CSS custom properties in a :root block.
func CSSVars(set Set) (string, error) {
	resolved, err := Resolve(set)
	if err != nil {
		return "", err
	}
	keys := sortedValueKeys(resolved.Values)

	var out strings.Builder
	namespace := cssName(resolved.Name)
	out.WriteString(":root {\n")
	for _, key := range keys {
		value, ok := cssValue(resolved.Values[key])
		if !ok {
			return "", Report{Issues: []Issue{{
				Code:     IssueUnrenderableCSS,
				Severity: SeverityError,
				Path:     key,
				Message:  "token value cannot be rendered as a CSS custom property",
			}}}
		}
		fmt.Fprintf(&out, "  --%s-%s: %s;\n", namespace, cssTokenName(key), value)
	}
	out.WriteString("}\n")
	return out.String(), nil
}

// CSSMap returns CSS custom-property names to resolved values.
func CSSMap(set Set) (map[string]string, error) {
	resolved, err := Resolve(set)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(resolved.Values))
	namespace := cssName(resolved.Name)
	for path, rawValue := range resolved.Values {
		value, ok := cssValue(rawValue)
		if !ok {
			return nil, Report{Issues: []Issue{{
				Code:     IssueUnrenderableCSS,
				Severity: SeverityError,
				Path:     path,
				Message:  "token value cannot be rendered as a CSS custom property",
			}}}
		}
		out["--"+namespace+"-"+cssTokenName(path)] = value
	}
	return out, nil
}

func cssValue(value Value) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, strings.TrimSpace(typed) != ""
	case json.RawMessage:
		return cssValueFromJSON(typed)
	case map[string]any:
		if hex, ok := typed["hex"].(string); ok && strings.TrimSpace(hex) != "" {
			return strings.TrimSpace(hex), true
		}
		if rawValue, hasValue := typed["value"]; hasValue {
			if unit, ok := typed["unit"].(string); ok {
				if number, ok := cssNumeric(rawValue); ok {
					return number + strings.TrimSpace(unit), true
				}
			}
		}
		return "", false
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case bool:
		return strconv.FormatBool(typed), true
	default:
		return "", false
	}
}

func cssValueFromJSON(value json.RawMessage) (string, bool) {
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return "", false
	}
	return cssValue(decoded)
}

func cssNumeric(value any) (string, bool) {
	switch typed := value.(type) {
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), true
	case json.Number:
		return typed.String(), true
	default:
		return "", false
	}
}

func cssTokenName(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, "."+rootSegment)
	return cssName(path)
}

func cssName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "$", "")
	return value
}
