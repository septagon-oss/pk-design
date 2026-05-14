package tokens

import (
	"strings"
	"testing"
)

func TestCSSVars(t *testing.T) {
	t.Parallel()

	css, err := CSSVars(Set{
		Name: "pk",
		Values: map[string]string{
			"color.surface": "#ffffff",
			"color.text":    "#111827",
		},
	})
	if err != nil {
		t.Fatalf("CSSVars() error = %v", err)
	}
	for _, want := range []string{"--pk-color-surface: #ffffff;", "--pk-color-text: #111827;"} {
		if !strings.Contains(css, want) {
			t.Fatalf("CSSVars() missing %q in:\n%s", want, css)
		}
	}
}
