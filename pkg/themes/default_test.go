// Validates: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package themes_test

// default_test.go proves the canonical theme is release-grade: it normalizes,
// resolves, renders to CSS custom properties, and carries every token path the
// admin shell and tw/emission depend on. If a path disappears or a value stops
// rendering, this fails before any consumer does.

import (
	"strings"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func TestDefaultThemeIsReleaseGrade(t *testing.T) {
	t.Parallel()

	theme := themes.Default()
	if theme.ID != "pk-default" {
		t.Fatalf("ID = %q", theme.ID)
	}
	if theme.Tokens.Name != "pk" {
		t.Fatalf("Tokens.Name = %q, want pk (CSS vars must render as --pk-*)", theme.Tokens.Name)
	}

	css, err := tokens.CSSVars(theme.Tokens)
	if err != nil {
		t.Fatalf("CSSVars: %v", err)
	}

	// The paths every consumer depends on. The admin shell's stylesheet
	// references these custom properties by name; renaming one here silently
	// un-themes the console.
	for _, path := range []string{
		"color.surface.canvas", "color.surface.primary", "color.surface.muted",
		"color.text.primary", "color.text.muted",
		"color.border.default", "color.border.strong",
		"color.accent.default", "color.accent.hover", "color.accent.on",
		"color.signal", "color.focus",
		"color.status.ok", "color.status.okbg",
		"color.status.warning", "color.status.warningbg",
		"color.status.danger", "color.status.dangerbg",
		"color.sidebar.bg", "color.sidebar.text", "color.sidebar.muted",
		"font.display", "font.body", "font.mono",
		"space.1", "space.2", "space.3", "space.4", "space.5", "space.6",
		"radius.sm", "radius.md", "radius.pill",
	} {
		if _, ok := theme.Tokens.Values[path]; !ok {
			t.Errorf("default theme missing %q", path)
		}
		cssVar := "--pk-" + strings.ReplaceAll(path, ".", "-")
		if !strings.Contains(css, cssVar+":") {
			t.Errorf("CSSVars output missing %s", cssVar)
		}
	}

	// The identity values of the palette. These are the brand; changing one is
	// a deliberate design decision that must show up in a diff of this file,
	// not only in a rendered screenshot.
	for path, want := range map[string]string{
		"color.surface.canvas": "#f2efe7",
		"color.text.primary":   "#15221f",
		"color.accent.default": "#0f5d4e",
		"color.signal":         "#d8f35d",
		"color.sidebar.bg":     "#12201d",
	} {
		if got := theme.Tokens.Values[path]; got != want {
			t.Errorf("%s = %v, want %q", path, got, want)
		}
	}
}

func TestDefaultThemeReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	first := themes.Default()
	first.Tokens.Values["color.signal"] = "#ff0000"
	first.Tokens.Types["color.signal"] = tokens.TypeNumber

	second := themes.Default()
	if got := second.Tokens.Values["color.signal"]; got != "#d8f35d" {
		t.Fatalf("mutating one copy leaked into the next: color.signal = %v", got)
	}
}

func TestDefaultThemeLayersAsABase(t *testing.T) {
	t.Parallel()

	// Per Layer's contract, an overlay ID is empty or equal to the base ID;
	// cross-theme composition goes through Extends and a catalog.
	overlay := themes.Theme{
		Tokens: tokens.Set{
			Name:   "pk",
			Values: map[string]tokens.Value{"color.accent.default": "#123456"},
			Types:  map[string]tokens.Type{"color.accent.default": tokens.TypeColor},
		},
	}
	layered, err := themes.Layer(themes.Default(), overlay)
	if err != nil {
		t.Fatalf("Layer: %v", err)
	}
	if got := layered.Tokens.Values["color.accent.default"]; got != "#123456" {
		t.Fatalf("overlay did not win: %v", got)
	}
	if got := layered.Tokens.Values["color.signal"]; got != "#d8f35d" {
		t.Fatalf("base value lost in layering: %v", got)
	}
}
