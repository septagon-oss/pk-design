// Validates: REQ-011.
// Per: ADR-0029.
// Discipline: C-14.

package tokens_test

import (
	"math"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func TestContrastRatioUsesWCAGRelativeLuminance(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name       string
		foreground string
		background string
		want       float64
	}{
		{name: "black on white", foreground: "#000000", background: "#ffffff", want: 21},
		{name: "same color", foreground: "#fff", background: "#ffffff", want: 1},
		{
			name:       "canonical info pair",
			foreground: "#2455c4",
			background: "#e5ecfa",
			want:       5.584142951127753,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := tokens.ContrastRatio(test.foreground, test.background)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(got-test.want) > 0.000001 {
				t.Fatalf("ContrastRatio() = %.9f, want %.9f", got, test.want)
			}
		})
	}
}

func TestContrastRatioRejectsContextDependentOrMalformedColors(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"rgb(0 0 0)",
		"#0000",
		"#000000ff",
		"#12",
		"#xyz",
	} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if _, err := tokens.ContrastRatio(value, "#ffffff"); err == nil {
				t.Fatalf("ContrastRatio(%q) unexpectedly succeeded", value)
			}
		})
	}
}
