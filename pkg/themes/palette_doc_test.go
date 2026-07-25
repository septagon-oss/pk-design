// Validates: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package themes_test

// palette_doc_test.go generates docs/palette.md from the canonical theme and
// pins it as a golden, so the documented palette cannot drift from the code.
// pk-docs' design-system page defers to this artifact when they disagree.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/themes"
)

func TestPaletteDocMatchesDefaultTheme(t *testing.T) {
	t.Parallel()

	theme := themes.Default()
	keys := make([]string, 0, len(theme.Tokens.Values))
	for k := range theme.Tokens.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("# PlatformKit palette\n\n")
	b.WriteString("Generated from `themes.Default()` by TestPaletteDocMatchesDefaultTheme.\n")
	b.WriteString("Regenerate with UPDATE_GOLDEN=1; edit the theme, never this file.\n\n")
	b.WriteString("| Token | Value |\n|---|---|\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "| `%s` | `%v` |\n", k, theme.Tokens.Values[k])
	}
	want := b.String()

	const path = "../../docs/palette.md"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (run with UPDATE_GOLDEN=1 to create): %v", path, err)
	}
	if string(got) != want {
		t.Fatal("docs/palette.md does not match themes.Default(); regenerate with UPDATE_GOLDEN=1 and review the diff")
	}
}
