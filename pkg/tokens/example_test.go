package tokens_test

// example_test.go documents the public token API with executable examples.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func ExampleCSSVars() {
	css, err := tokens.CSSVars(tokens.Set{
		Name: "pk",
		Values: map[string]tokens.Value{
			"color.text.primary": "#111827",
		},
		Types: map[string]tokens.Type{
			"color.text.primary": tokens.TypeColor,
		},
	})
	if err != nil {
		panic(err)
	}

	lines := strings.Split(strings.TrimSpace(css), "\n")
	fmt.Println(lines[0])
	fmt.Println(strings.TrimSpace(lines[1]))
	fmt.Println(lines[2])

	// Output:
	// :root {
	// --pk-color-text-primary: #111827;
	// }
}

func ExampleMerge() {
	merged, err := tokens.Merge(
		tokens.Set{
			Name: "pk",
			Values: map[string]tokens.Value{
				"color.text.primary": "#111827",
			},
			Types: map[string]tokens.Type{
				"color.text.primary": tokens.TypeColor,
			},
		},
		tokens.Set{
			Values: map[string]tokens.Value{
				"color.text.primary": "#0f172a",
				"space.2":            "0.5rem",
			},
			Types: map[string]tokens.Type{
				"space.2": tokens.TypeDimension,
			},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(merged.Values["color.text.primary"])
	fmt.Println(merged.Types["space.2"])

	// Output:
	// #0f172a
	// dimension
}

func ExampleReference() {
	ref, err := tokens.Reference("color.brand.primary")
	if err != nil {
		panic(err)
	}

	fmt.Println(ref)

	// Output:
	// {color.brand.primary}
}
