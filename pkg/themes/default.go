// Implements: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package themes

// default.go loads the canonical PlatformKit OSS theme from its embedded DTCG
// source. The JSON file is the editable source consumed by code, Figma
// delivery, Storybook, docs, and round-trip receivers; Go does not mirror its
// values.

import (
	_ "embed"
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

const (
	// DefaultThemeID is the stable identity of the canonical OSS theme.
	DefaultThemeID = "pk-default"
	// DefaultThemeVersion is bumped when the public token contract changes.
	DefaultThemeVersion = "1.1.0"
	// DefaultSourceURI is the repository-relative writable token source.
	DefaultSourceURI = "pkg/themes/default.tokens.json"
)

//go:embed default.tokens.json
var defaultSource []byte

// DefaultSource returns a defensive copy of the canonical DTCG source.
func DefaultSource() []byte {
	return append([]byte(nil), defaultSource...)
}

// ParseDefaultSource validates source as the canonical OSS theme. Receivers
// use this before planning a write so malformed or semantically invalid edits
// fail before touching the repository.
func ParseDefaultSource(source []byte) (Theme, error) {
	set, err := tokens.ParseDTCGJSON("pk", source)
	if err != nil {
		return Theme{}, fmt.Errorf("pk-design: parse default DTCG source: %w", err)
	}
	normalized, err := (Theme{
		ID:          DefaultThemeID,
		Name:        "PlatformKit Default",
		Version:     DefaultThemeVersion,
		Description: "Warm editorial surfaces, ink text, deep green accent, lime signal.",
		Tokens:      set,
	}).Normalize()
	if err != nil {
		return Theme{}, fmt.Errorf("pk-design: normalize default DTCG source: %w", err)
	}
	if err := validateDefaultAccessibility(normalized); err != nil {
		return Theme{}, err
	}
	return normalized, nil
}

// Default returns the canonical PlatformKit theme: warm editorial surfaces,
// ink text, a deep green accent, and a lime signal, with a serif display face
// over IBM Plex body and mono stacks.
//
// The value is rebuilt on every call and normalized, so callers receive
// defensive copies and cannot mutate shared state. Overriding is layering:
//
//	theme, err := themes.Layer(themes.Default(), brandOverlay)
func Default() Theme {
	normalized, err := ParseDefaultSource(DefaultSource())
	if err != nil {
		// The embedded source is a checked-in artifact proven valid by
		// TestDefaultThemeIsReleaseGrade; reaching this is a programmer error
		// in this file, not a runtime condition a caller could handle.
		panic(fmt.Sprintf("pk-design: default theme invariant broken: %v", err))
	}
	return normalized
}
