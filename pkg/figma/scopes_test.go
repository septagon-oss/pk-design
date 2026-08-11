package figma

// Validates: REQ-011.
// Per: ADR-0076.
// Discipline: C-14.
// scopes_test.go pins the token taxonomy a variable's Figma scopes are derived
// from. The rules are shared by every profile, so they are stated against the
// canonical groups rather than any client's token names.

import (
	"slices"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func TestVariableScopesFollowTheTokenTaxonomy(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		path  string
		kind  tokens.Type
		scope []string
	}{
		{"/color/border/brand", tokens.TypeColor, []string{"STROKE_COLOR"}},
		{"/color/foreground/primary", tokens.TypeColor, []string{"TEXT_FILL"}},
		{"/color/surface/primary", tokens.TypeColor, []string{"FRAME_FILL", "SHAPE_FILL"}},
		{"/borderRadius/lg", tokens.TypeDimension, []string{"CORNER_RADIUS"}},
		{"/spacing/4", tokens.TypeDimension, []string{"GAP", "WIDTH_HEIGHT"}},
		{"/typography/fontSize-lg", tokens.TypeDimension, []string{"FONT_SIZE"}},
		{"/typography/lineHeight-lg", tokens.TypeDimension, []string{"LINE_HEIGHT"}},
		{"/density/comfortable/element-gap", tokens.TypeDimension, []string{"GAP", "WIDTH_HEIGHT"}},
		{"/opacity/disabled", tokens.TypeNumber, []string{"OPACITY"}},
		{"/client/brand/500", tokens.TypeColor, []string{"ALL_FILLS", "STROKE_COLOR"}},
	} {
		got := variableScopes(testCase.path, testCase.kind)
		if !slices.Equal(got, testCase.scope) {
			t.Errorf("%s: got %v, want %v", testCase.path, got, testCase.scope)
		}
	}
}

// TestUnclassifiedTokensKeepEveryScope states the deliberate fallback: a token
// whose group we do not recognise stays offered everywhere rather than
// vanishing from the picker.
func TestUnclassifiedTokensKeepEveryScope(t *testing.T) {
	t.Parallel()

	if got := variableScopes("/motion/duration/fast", tokens.TypeDuration); got != nil {
		t.Fatalf("unclassified token was narrowed to %v", got)
	}
}

// TestPaletteColorsStayUsableAsFillAndStroke keeps primitives broad: a brand
// scale step is a legitimate fill and a legitimate stroke.
func TestPaletteColorsStayUsableAsFillAndStroke(t *testing.T) {
	t.Parallel()

	got := variableScopes("/color/primitive/orange/500", tokens.TypeColor)
	if !slices.Equal(got, []string{"ALL_FILLS", "STROKE_COLOR"}) {
		t.Fatalf("palette colour narrowed to %v", got)
	}
}
