package catalog_test

// Validates: REQ-011.
// Per: ADR-0031.
// Discipline: C-14.
// example_test.go documents the public catalog API with executable examples.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/catalog"
	"github.com/septagon-oss/pk-design/pkg/components"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func ExampleBuilder_Build() {
	designCatalog, err := catalog.New().
		Add(catalog.Contribution{
			Manifest: catalog.Manifest{
				Source:        "booking_management",
				SchemaVersion: catalog.ManifestSchemaVersion,
				Version:       "1.0.0",
			},
			TokenSets: []tokens.Set{
				{
					Name: "pk",
					Values: map[string]tokens.Value{
						"color.text.primary": "#111827",
					},
				},
			},
			Components: []components.Descriptor{
				{
					ID:       "booking.calendar",
					Category: components.CategoryOrganism,
				},
			},
		}).
		Build()
	if err != nil {
		panic(err)
	}

	tokenEntries := designCatalog.TokenSetEntries()
	component, ok := designCatalog.Component("booking.calendar")
	fmt.Println(tokenEntries[0].Key, tokenEntries[0].Source)
	fmt.Println(ok, component.Category)

	// Output:
	// pk booking_management
	// true organism
}
