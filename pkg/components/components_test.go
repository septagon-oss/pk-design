package components

// Validates: REQ-011.
// Per: ADR-0031.
// Discipline: C-14.
// components_test.go validates component descriptor normalization, validation,
// deterministic ordering, and defensive-copy behavior.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"slices"
	"testing"
)

func validDescriptor() Descriptor {
	return Descriptor{
		ID:          " button.primary ",
		Name:        " Primary Button ",
		Category:    CategoryAtom,
		Description: " Action trigger ",
		ModuleID:    "design_core",
		Props: []Prop{
			{
				Name:        "tone",
				Type:        PropEnum,
				EnumValues:  []string{"neutral", "brand", "neutral"},
				Default:     "brand",
				Description: "Visual tone",
			},
			{Name: "disabled", Type: PropBoolean},
		},
		Slots: []Slot{
			{Name: "icon", Description: "Leading visual"},
			{Name: "label", Required: true},
		},
		Variants: []Variant{
			{Name: "size", Values: []string{"sm", "md"}, Default: "md"},
		},
		RequiredTokens: []string{"color.text.primary", "space.2", "space.2"},
		Anatomy: []AnatomyNode{
			{
				Name:   "root",
				Role:   "button",
				Tokens: []string{"color.surface.primary"},
				Children: []AnatomyNode{
					{Name: "label", Tokens: []string{"color.text.primary"}},
				},
				Metadata: map[string]string{" part ": " control "},
			},
		},
		Metadata: map[string]string{" owner ": " design "},
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	descriptor := validDescriptor()
	normalized, err := descriptor.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.ID != "button.primary" || normalized.Name != "Primary Button" {
		t.Fatalf("Normalize() identity = %#v", normalized)
	}
	if normalized.SourceOfTruth != SourceDefinition {
		t.Fatalf("SourceOfTruth = %q; want %q", normalized.SourceOfTruth, SourceDefinition)
	}
	if !slices.Equal([]string{normalized.Props[0].Name, normalized.Props[1].Name}, []string{"disabled", "tone"}) {
		t.Fatalf("Props not sorted: %#v", normalized.Props)
	}
	if !slices.Equal(normalized.Props[1].EnumValues, []string{"neutral", "brand"}) {
		t.Fatalf("EnumValues = %#v", normalized.Props[1].EnumValues)
	}
	if !slices.Equal(normalized.RequiredTokens, []string{"color.text.primary", "space.2"}) {
		t.Fatalf("RequiredTokens = %v", normalized.RequiredTokens)
	}
	if normalized.Anatomy[0].Metadata["part"] != "control" || normalized.Metadata["owner"] != "design" {
		t.Fatalf("Metadata not normalized: %#v %#v", normalized.Anatomy[0].Metadata, normalized.Metadata)
	}

	normalized.Props[1].EnumValues[0] = "mutated"
	normalized.Anatomy[0].Children[0].Tokens[0] = "mutated"
	again, err := descriptor.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if again.Props[1].EnumValues[0] != "neutral" || again.Anatomy[0].Children[0].Tokens[0] != "color.text.primary" {
		t.Fatalf("Normalize() returned aliased slices: %#v %#v", again.Props, again.Anatomy)
	}
}

func TestNormalizeRejectsInvalid(t *testing.T) {
	t.Parallel()

	base := validDescriptor()
	tests := []Descriptor{
		{},
		func() Descriptor { d := base; d.ID = "bad id"; return d }(),
		func() Descriptor { d := base; d.Category = "primitive"; return d }(),
		func() Descriptor { d := base; d.SourceOfTruth = "wiki"; return d }(),
		func() Descriptor { d := base; d.ModuleID = "bad module"; return d }(),
		func() Descriptor { d := base; d.Props = []Prop{{Type: PropString}}; return d }(),
		func() Descriptor { d := base; d.Props = []Prop{{Name: "label.part", Type: PropString}}; return d }(),
		func() Descriptor { d := base; d.Props = []Prop{{Name: "label", Type: "text"}}; return d }(),
		func() Descriptor { d := base; d.Props = []Prop{{Name: "tone", Type: PropEnum}}; return d }(),
		func() Descriptor {
			d := base
			d.Props = []Prop{{Name: "tone", Type: PropEnum, EnumValues: []string{"bad value"}}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Props = []Prop{{Name: "tone", Type: PropEnum, EnumValues: []string{"brand"}, Default: "neutral"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Props = []Prop{{Name: "label", Type: PropString, EnumValues: []string{"short"}}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Props = []Prop{{Name: "icon", Type: PropToken, Default: "icon default"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Props = []Prop{{Name: "label", Type: PropString}, {Name: "label", Type: PropString}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Slots = []Slot{{Name: "body"}, {Name: "body"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Slots = []Slot{{Name: "body.main"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Variants = []Variant{{Name: "size", Values: []string{"sm"}, Default: "md"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Variants = []Variant{{Name: "size", Values: []string{"bad value"}}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.RequiredTokens = []string{"color text"}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Anatomy = []AnatomyNode{{Name: "root"}, {Name: "root"}}
			return d
		}(),
		func() Descriptor {
			d := base
			d.Anatomy = []AnatomyNode{{Name: "root.part"}}
			return d
		}(),
	}
	for _, test := range tests {
		if _, err := test.Normalize(); err == nil {
			t.Fatalf("Normalize(%#v) should fail", test)
		}
	}
}
