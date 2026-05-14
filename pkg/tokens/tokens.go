// Package tokens provides a minimal semantic design-token model.
package tokens

import (
	"fmt"
	"sort"
	"strings"
)

// Set is a named group of semantic design tokens.
type Set struct {
	Name   string
	Values map[string]string
}

// CSSVars renders tokens as CSS custom properties.
func CSSVars(set Set) (string, error) {
	name := strings.TrimSpace(set.Name)
	if name == "" {
		name = "platformkit"
	}
	keys := make([]string, 0, len(set.Values))
	for key := range set.Values {
		key = strings.TrimSpace(key)
		if key == "" {
			return "", fmt.Errorf("token key must not be empty")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var out strings.Builder
	out.WriteString(":root {\n")
	for _, key := range keys {
		value := strings.TrimSpace(set.Values[key])
		if value == "" {
			return "", fmt.Errorf("token %q has empty value", key)
		}
		fmt.Fprintf(&out, "  --%s-%s: %s;\n", name, cssName(key), value)
	}
	out.WriteString("}\n")
	return out.String(), nil
}

func cssName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
