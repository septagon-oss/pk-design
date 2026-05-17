package components_test

// example_test.go documents the public component descriptor API with
// executable examples.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/components"
)

func ExampleDescriptor_Normalize() {
	descriptor, err := (components.Descriptor{
		ID:       "button.primary",
		Category: components.CategoryAtom,
		Props: []components.Prop{
			{
				Name:       "tone",
				Type:       components.PropEnum,
				EnumValues: []string{"brand", "neutral"},
				Default:    "brand",
			},
			{Name: "disabled", Type: components.PropBoolean},
		},
		RequiredTokens: []string{"color.text.primary", "space.2", "space.2"},
	}).Normalize()
	if err != nil {
		panic(err)
	}

	fmt.Println(descriptor.ID)
	fmt.Println(descriptor.SourceOfTruth)
	fmt.Println(descriptor.Props[0].Name)
	fmt.Println(descriptor.RequiredTokens)

	// Output:
	// button.primary
	// definition
	// disabled
	// [color.text.primary space.2]
}
