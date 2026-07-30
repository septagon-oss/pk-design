// Implements: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package themes

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

var defaultStatusContrastPairs = []struct {
	foreground string
	background string
}{
	{foreground: "color.status.ok", background: "color.status.okbg"},
	{foreground: "color.status.warning", background: "color.status.warningbg"},
	{foreground: "color.status.danger", background: "color.status.dangerbg"},
	{foreground: "color.status.info", background: "color.status.infobg"},
}

func validateDefaultAccessibility(theme Theme) error {
	for _, pair := range defaultStatusContrastPairs {
		foreground, err := colorToken(theme, pair.foreground)
		if err != nil {
			return err
		}
		background, err := colorToken(theme, pair.background)
		if err != nil {
			return err
		}
		ratio, err := tokens.ContrastRatio(foreground, background)
		if err != nil {
			return fmt.Errorf(
				"pk-design: measure %s against %s: %w",
				pair.foreground,
				pair.background,
				err,
			)
		}
		if ratio < tokens.WCAGAAContrast {
			return fmt.Errorf(
				"pk-design: %s against %s has %.2f:1 contrast; want at least %.1f:1",
				pair.foreground,
				pair.background,
				ratio,
				tokens.WCAGAAContrast,
			)
		}
	}
	return nil
}

func colorToken(theme Theme, path string) (string, error) {
	value, ok := theme.Tokens.Values[path]
	if !ok {
		return "", fmt.Errorf("pk-design: required accessible color token %q is missing", path)
	}
	color, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("pk-design: accessible color token %q is %T, want string", path, value)
	}
	return color, nil
}
