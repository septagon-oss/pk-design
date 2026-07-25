// Implements: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package themes

// default.go owns the canonical PlatformKit theme values. Until now this
// package shipped layering machinery with no values, while the palette the
// binary actually renders lived inside pk-modules' admin shell — so "the
// design system" was real but unfindable. Default is the single source of
// truth: the admin consumes it, tw/emission renders utility CSS from it, and
// pk-docs generates its palette table from it, so none of them can drift.

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// Default returns the canonical PlatformKit theme: warm editorial surfaces,
// ink text, a deep green accent, and a lime signal, with a serif display face
// over IBM Plex body and mono stacks.
//
// The value is rebuilt on every call and normalized, so callers receive
// defensive copies and cannot mutate shared state. Overriding is layering:
//
//	theme, err := themes.Layer(themes.Default(), brandOverlay)
func Default() Theme {
	color := func(v string) tokens.Value { return tokens.Value(v) }
	theme := Theme{
		ID:          "pk-default",
		Name:        "PlatformKit Default",
		Version:     "1.0.0",
		Description: "Warm editorial surfaces, ink text, deep green accent, lime signal.",
		Tokens: tokens.Set{
			Name: "pk",
			Values: map[string]tokens.Value{
				// Warm editorial surfaces.
				"color.surface.canvas":  color("#f2efe7"),
				"color.surface.primary": color("#fffdf7"),
				"color.surface.muted":   color("#e9e4d8"),
				// Ink and supporting copy.
				"color.text.primary": color("#15221f"),
				"color.text.muted":   color("#5f6b65"),
				// Lines.
				"color.border.default": color("#cbc5b8"),
				"color.border.strong":  color("#8f988f"),
				// Brand/action palette.
				"color.accent.default": color("#0f5d4e"),
				"color.accent.hover":   color("#0a493e"),
				"color.accent.on":      color("#f9fff9"),
				"color.signal":         color("#d8f35d"),
				"color.focus":          color("#326de6"),
				// Status palette.
				"color.status.ok":        color("#12715d"),
				"color.status.okbg":      color("#dcf3e8"),
				"color.status.warning":   color("#9a5318"),
				"color.status.warningbg": color("#fff0d2"),
				"color.status.danger":    color("#9e3833"),
				"color.status.dangerbg":  color("#fbe5e2"),
				// Navigation.
				"color.sidebar.bg":    color("#12201d"),
				"color.sidebar.text":  color("#eff4e9"),
				"color.sidebar.muted": color("#aebbb2"),
				// Type stacks.
				"font.display": color(`"Iowan Old Style", "Palatino Linotype", Palatino, Georgia, serif`),
				"font.body":    color(`"IBM Plex Sans", Aptos, "Helvetica Neue", sans-serif`),
				"font.mono":    color(`"IBM Plex Mono", "SFMono-Regular", Consolas, monospace`),
				// Spacing scale (4px base).
				"space.1": color("4px"),
				"space.2": color("8px"),
				"space.3": color("12px"),
				"space.4": color("16px"),
				"space.5": color("24px"),
				"space.6": color("32px"),
				// Radii.
				"radius.sm":   color("4px"),
				"radius.md":   color("8px"),
				"radius.pill": color("999px"),
			},
			Types: map[string]tokens.Type{
				"color.surface.canvas":   tokens.TypeColor,
				"color.surface.primary":  tokens.TypeColor,
				"color.surface.muted":    tokens.TypeColor,
				"color.text.primary":     tokens.TypeColor,
				"color.text.muted":       tokens.TypeColor,
				"color.border.default":   tokens.TypeColor,
				"color.border.strong":    tokens.TypeColor,
				"color.accent.default":   tokens.TypeColor,
				"color.accent.hover":     tokens.TypeColor,
				"color.accent.on":        tokens.TypeColor,
				"color.signal":           tokens.TypeColor,
				"color.focus":            tokens.TypeColor,
				"color.status.ok":        tokens.TypeColor,
				"color.status.okbg":      tokens.TypeColor,
				"color.status.warning":   tokens.TypeColor,
				"color.status.warningbg": tokens.TypeColor,
				"color.status.danger":    tokens.TypeColor,
				"color.status.dangerbg":  tokens.TypeColor,
				"color.sidebar.bg":       tokens.TypeColor,
				"color.sidebar.text":     tokens.TypeColor,
				"color.sidebar.muted":    tokens.TypeColor,
				"font.display":           tokens.TypeFontFamily,
				"font.body":              tokens.TypeFontFamily,
				"font.mono":              tokens.TypeFontFamily,
				"space.1":                tokens.TypeDimension,
				"space.2":                tokens.TypeDimension,
				"space.3":                tokens.TypeDimension,
				"space.4":                tokens.TypeDimension,
				"space.5":                tokens.TypeDimension,
				"space.6":                tokens.TypeDimension,
				"radius.sm":              tokens.TypeDimension,
				"radius.md":              tokens.TypeDimension,
				"radius.pill":            tokens.TypeDimension,
			},
		},
	}
	normalized, err := theme.Normalize()
	if err != nil {
		// The literal above is a compile-time constant proven valid by
		// TestDefaultThemeIsReleaseGrade; reaching this is a programmer error
		// in this file, not a runtime condition a caller could handle.
		panic(fmt.Sprintf("pk-design: default theme invariant broken: %v", err))
	}
	return normalized
}
