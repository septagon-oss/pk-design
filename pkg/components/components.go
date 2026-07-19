// Package components defines provider-neutral design component descriptors.
package components

// components.go owns the OSS component-design contract used by modules and
// renderers to describe component APIs, slots, variants, anatomy, and token
// dependencies without importing frontend implementation code.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Category describes a component's design-system level.
type Category string

// Component categories following atomic-design levels plus surface and pattern.
const (
	// CategoryAtom is an indivisible primitive component.
	CategoryAtom Category = "atom"
	// CategoryMolecule is a small composition of atoms.
	CategoryMolecule Category = "molecule"
	// CategoryOrganism is a larger composition of molecules and atoms.
	CategoryOrganism Category = "organism"
	// CategoryTemplate is a page-level layout scaffold.
	CategoryTemplate Category = "template"
	// CategorySurface is a container or backdrop component.
	CategorySurface Category = "surface"
	// CategoryPattern is a reusable interaction or composition pattern.
	CategoryPattern Category = "pattern"
)

// SourceOfTruth identifies the visual authority for a component contract.
type SourceOfTruth string

// Visual authorities a component contract can be sourced from.
const (
	// SourceDefinition treats this descriptor as the authoritative definition.
	SourceDefinition SourceOfTruth = "definition"
	// SourceRuntime defers to the live runtime implementation.
	SourceRuntime SourceOfTruth = "runtime"
	// SourceStorybook defers to a Storybook story as the source of truth.
	SourceStorybook SourceOfTruth = "storybook"
)

// PropType identifies stable component prop shapes.
type PropType string

// Supported component prop shapes.
const (
	// PropString is a string-valued prop.
	PropString PropType = "string"
	// PropNumber is a numeric-valued prop.
	PropNumber PropType = "number"
	// PropBoolean is a boolean-valued prop.
	PropBoolean PropType = "boolean"
	// PropEnum is a prop constrained to declared enum values.
	PropEnum PropType = "enum"
	// PropObject is an object-valued prop.
	PropObject PropType = "object"
	// PropArray is an array-valued prop.
	PropArray PropType = "array"
	// PropNode is a renderable-node prop (e.g. children).
	PropNode PropType = "node"
	// PropToken is a prop whose value references a design token.
	PropToken PropType = "token"
)

// Prop describes one component property.
type Prop struct {
	Name        string
	Type        PropType
	Required    bool
	Description string
	EnumValues  []string
	Default     string
}

// Slot describes one named render slot.
type Slot struct {
	Name        string
	Required    bool
	Description string
}

// Variant describes one named visual or behavioral variant.
type Variant struct {
	Name        string
	Values      []string
	Default     string
	Description string
}

// AnatomyNode describes a renderer-neutral component part tree.
type AnatomyNode struct {
	Name     string
	Role     string
	Tokens   []string
	Children []AnatomyNode
	Metadata map[string]string
}

// Descriptor is the public component design contract.
type Descriptor struct {
	ID             string
	Name           string
	Category       Category
	SourceOfTruth  SourceOfTruth
	Description    string
	ModuleID       string
	Props          []Prop
	Slots          []Slot
	Variants       []Variant
	RequiredTokens []string
	Anatomy        []AnatomyNode
	Metadata       map[string]string
}

// Normalize trims, validates, sorts deterministic lists, and defensively copies
// a component descriptor.
func (d Descriptor) Normalize() (Descriptor, error) {
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" {
		return Descriptor{}, fmt.Errorf("component ID is required")
	}
	if !validIdentifier(d.ID) {
		return Descriptor{}, fmt.Errorf("component ID %q must use letters, numbers, dot, underscore, or hyphen", d.ID)
	}
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" {
		d.Name = d.ID
	}
	d.Category = Category(strings.TrimSpace(string(d.Category)))
	if !validCategory(d.Category) {
		return Descriptor{}, fmt.Errorf("component %q category %q is not supported", d.ID, d.Category)
	}
	d.SourceOfTruth = SourceOfTruth(strings.TrimSpace(string(d.SourceOfTruth)))
	if d.SourceOfTruth == "" {
		d.SourceOfTruth = SourceDefinition
	}
	if !validSource(d.SourceOfTruth) {
		return Descriptor{}, fmt.Errorf("component %q source of truth %q is not supported", d.ID, d.SourceOfTruth)
	}
	d.Description = strings.TrimSpace(d.Description)
	d.ModuleID = strings.TrimSpace(d.ModuleID)
	if d.ModuleID != "" && !validIdentifier(d.ModuleID) {
		return Descriptor{}, fmt.Errorf("component %q module ID %q is invalid", d.ID, d.ModuleID)
	}
	var err error
	if d.Props, err = normalizeProps(d.Props); err != nil {
		return Descriptor{}, fmt.Errorf("component %q props: %w", d.ID, err)
	}
	if d.Slots, err = normalizeSlots(d.Slots); err != nil {
		return Descriptor{}, fmt.Errorf("component %q slots: %w", d.ID, err)
	}
	if d.Variants, err = normalizeVariants(d.Variants); err != nil {
		return Descriptor{}, fmt.Errorf("component %q variants: %w", d.ID, err)
	}
	if d.RequiredTokens, err = normalizeTokenRefs(d.RequiredTokens); err != nil {
		return Descriptor{}, fmt.Errorf("component %q required tokens: %w", d.ID, err)
	}
	if d.Anatomy, err = normalizeAnatomy(d.Anatomy); err != nil {
		return Descriptor{}, fmt.Errorf("component %q anatomy: %w", d.ID, err)
	}
	d.Metadata = normalizeStringMap(d.Metadata)
	return d, nil
}

func normalizeProps(props []Prop) ([]Prop, error) {
	out := make([]Prop, 0, len(props))
	seen := map[string]struct{}{}
	for _, prop := range props {
		prop.Name = strings.TrimSpace(prop.Name)
		if prop.Name == "" {
			return nil, fmt.Errorf("prop name is required")
		}
		if !validLocalIdentifier(prop.Name) {
			return nil, fmt.Errorf("prop name %q is invalid", prop.Name)
		}
		prop.Type = PropType(strings.TrimSpace(string(prop.Type)))
		if !validPropType(prop.Type) {
			return nil, fmt.Errorf("prop %q type %q is not supported", prop.Name, prop.Type)
		}
		prop.Description = strings.TrimSpace(prop.Description)
		prop.Default = strings.TrimSpace(prop.Default)
		var err error
		prop.EnumValues, err = normalizeChoices(prop.EnumValues)
		if err != nil {
			return nil, fmt.Errorf("prop %q enum values: %w", prop.Name, err)
		}
		if prop.Type == PropEnum {
			if len(prop.EnumValues) == 0 {
				return nil, fmt.Errorf("prop %q enum values are required", prop.Name)
			}
			if prop.Default != "" && !contains(prop.EnumValues, prop.Default) {
				return nil, fmt.Errorf("prop %q default %q is not declared", prop.Name, prop.Default)
			}
		}
		if prop.Type != PropEnum && len(prop.EnumValues) > 0 {
			return nil, fmt.Errorf("prop %q enum values require enum type", prop.Name)
		}
		if prop.Type == PropToken && prop.Default != "" {
			if err := validateTokenRef(prop.Default); err != nil {
				return nil, fmt.Errorf("prop %q default token: %w", prop.Name, err)
			}
		}
		if _, exists := seen[prop.Name]; exists {
			return nil, fmt.Errorf("duplicate prop %q", prop.Name)
		}
		seen[prop.Name] = struct{}{}
		out = append(out, prop)
	}
	slices.SortStableFunc(out, func(a, b Prop) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

func normalizeSlots(slots []Slot) ([]Slot, error) {
	out := make([]Slot, 0, len(slots))
	seen := map[string]struct{}{}
	for _, slot := range slots {
		slot.Name = strings.TrimSpace(slot.Name)
		if slot.Name == "" {
			return nil, fmt.Errorf("slot name is required")
		}
		if !validLocalIdentifier(slot.Name) {
			return nil, fmt.Errorf("slot name %q is invalid", slot.Name)
		}
		slot.Description = strings.TrimSpace(slot.Description)
		if _, exists := seen[slot.Name]; exists {
			return nil, fmt.Errorf("duplicate slot %q", slot.Name)
		}
		seen[slot.Name] = struct{}{}
		out = append(out, slot)
	}
	slices.SortStableFunc(out, func(a, b Slot) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

func normalizeVariants(variants []Variant) ([]Variant, error) {
	out := make([]Variant, 0, len(variants))
	seen := map[string]struct{}{}
	for _, variant := range variants {
		variant.Name = strings.TrimSpace(variant.Name)
		if variant.Name == "" {
			return nil, fmt.Errorf("variant name is required")
		}
		if !validLocalIdentifier(variant.Name) {
			return nil, fmt.Errorf("variant name %q is invalid", variant.Name)
		}
		var err error
		variant.Values, err = normalizeChoices(variant.Values)
		if err != nil {
			return nil, fmt.Errorf("variant %q values: %w", variant.Name, err)
		}
		if len(variant.Values) == 0 {
			return nil, fmt.Errorf("variant %q values are required", variant.Name)
		}
		variant.Default = strings.TrimSpace(variant.Default)
		if variant.Default != "" && !contains(variant.Values, variant.Default) {
			return nil, fmt.Errorf("variant %q default %q is not declared", variant.Name, variant.Default)
		}
		variant.Description = strings.TrimSpace(variant.Description)
		if _, exists := seen[variant.Name]; exists {
			return nil, fmt.Errorf("duplicate variant %q", variant.Name)
		}
		seen[variant.Name] = struct{}{}
		out = append(out, variant)
	}
	slices.SortStableFunc(out, func(a, b Variant) int { return cmp.Compare(a.Name, b.Name) })
	return out, nil
}

func normalizeAnatomy(nodes []AnatomyNode) ([]AnatomyNode, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	out := make([]AnatomyNode, len(nodes))
	seen := map[string]struct{}{}
	for i, node := range nodes {
		normalized, err := normalizeAnatomyNode(node)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized.Name]; exists {
			return nil, fmt.Errorf("duplicate anatomy node %q", normalized.Name)
		}
		seen[normalized.Name] = struct{}{}
		out[i] = normalized
	}
	return out, nil
}

func normalizeAnatomyNode(node AnatomyNode) (AnatomyNode, error) {
	node.Name = strings.TrimSpace(node.Name)
	if node.Name == "" {
		return AnatomyNode{}, fmt.Errorf("anatomy node name is required")
	}
	if !validLocalIdentifier(node.Name) {
		return AnatomyNode{}, fmt.Errorf("anatomy node name %q is invalid", node.Name)
	}
	node.Role = strings.TrimSpace(node.Role)
	var err error
	if node.Tokens, err = normalizeTokenRefs(node.Tokens); err != nil {
		return AnatomyNode{}, fmt.Errorf("anatomy node %q tokens: %w", node.Name, err)
	}
	if node.Children, err = normalizeAnatomy(node.Children); err != nil {
		return AnatomyNode{}, fmt.Errorf("anatomy node %q children: %w", node.Name, err)
	}
	node.Metadata = normalizeStringMap(node.Metadata)
	return node, nil
}

func normalizeChoices(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !validLocalIdentifier(value) {
			return nil, fmt.Errorf("value %q is invalid", value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func normalizeTokenRefs(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if err := validateTokenRef(value); err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out, nil
}

func validateTokenRef(value string) error {
	if strings.ContainsAny(value, " \t\n\r") {
		return fmt.Errorf("token ref %q must not contain whitespace", value)
	}
	for segment := range strings.SplitSeq(value, ".") {
		if !validIdentifier(segment) {
			return fmt.Errorf("token ref %q contains invalid segment %q", value, segment)
		}
	}
	return nil
}

func validCategory(category Category) bool {
	switch category {
	case CategoryAtom, CategoryMolecule, CategoryOrganism, CategoryTemplate, CategorySurface, CategoryPattern:
		return true
	default:
		return false
	}
}

func validSource(source SourceOfTruth) bool {
	switch source {
	case SourceDefinition, SourceRuntime, SourceStorybook:
		return true
	default:
		return false
	}
}

func validPropType(propType PropType) bool {
	switch propType {
	case PropString, PropNumber, PropBoolean, PropEnum, PropObject, PropArray, PropNode, PropToken:
		return true
	default:
		return false
	}
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func validLocalIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func normalizeStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}
