package themes_test

// example_test.go documents the public theme API with executable examples.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func ExampleLayer() {
	layered, err := themes.Layer(
		themes.Theme{
			ID: "light",
			Tokens: tokens.Set{
				Name: "pk",
				Values: map[string]tokens.Value{
					"color.text.primary": "#111827",
				},
				Types: map[string]tokens.Type{
					"color.text.primary": tokens.TypeColor,
				},
			},
		},
		themes.Theme{
			Tokens: tokens.Set{
				Values: map[string]tokens.Value{
					"color.text.primary": "#0f172a",
				},
				Types: map[string]tokens.Type{
					"color.text.primary": tokens.TypeColor,
				},
			},
			Metadata: map[string]string{"source": "tenant"},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(layered.ID)
	fmt.Println(layered.Tokens.Values["color.text.primary"])
	fmt.Println(layered.Metadata["source"])

	// Output:
	// light
	// #0f172a
	// tenant
}

func ExampleNewStack() {
	stack, err := themes.NewStack(
		themes.TokenLayer{
			ID:   "client",
			Kind: themes.LayerClient,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#1d4ed8"},
			},
		},
		themes.TokenLayer{
			ID:   "base",
			Kind: themes.LayerBase,
			Tokens: tokens.Set{
				Name:   "pk",
				Values: map[string]tokens.Value{"color.brand.primary": "#2563eb"},
				Groups: map[string]tokens.Group{"color.brand": {Type: tokens.TypeColor}},
			},
		},
	)
	if err != nil {
		panic(err)
	}
	resolved, err := stack.Resolve()
	if err != nil {
		panic(err)
	}

	fmt.Println(resolved.Values["color.brand.primary"])

	// Output:
	// #1d4ed8
}
