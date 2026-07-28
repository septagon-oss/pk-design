// Package blueprint defines the provider-neutral visual component model shared
// by design libraries, executable previews, and native design-tool adapters.
package blueprint

// Implements: REQ-011.
// Per: ADR-0022, ADR-0076.
// Discipline: C-14.
// model.go owns the portable visual blueprint. It intentionally contains no
// Figma, Storybook, browser, renderer, or private PlatformKit dependency.
//
// ADR: ADR-0029 (file purpose declaration), ADR-0076 (one layered delivery graph).
// Convention: C-14 (every Go file declares its purpose).

// SourceOfTruth identifies which executable or authored surface is the visual
// authority for a component.
type SourceOfTruth string

const (
	// SourceDefinition means the native blueprint is authoritative.
	SourceDefinition SourceOfTruth = "definition"
	// SourceRuntime means a renderer is authoritative and the blueprint is a
	// documented projection of that renderer.
	SourceRuntime SourceOfTruth = "runtime"
	// SourceStorybook means an identity-matched Storybook fixture is the
	// authority for runtime behavior.
	SourceStorybook SourceOfTruth = "storybook"
)

// NodeKind describes one semantic layer in a target-neutral visual tree.
type NodeKind string

const (
	NodeFrame    NodeKind = "frame"
	NodeText     NodeKind = "text"
	NodeImage    NodeKind = "image"
	NodeSVG      NodeKind = "svg"
	NodeSlot     NodeKind = "slot"
	NodeInstance NodeKind = "instance"
)

// AssetKind identifies which external asset pipeline materializes a node.
type AssetKind string

const (
	AssetImage AssetKind = "image"
	AssetSVG   AssetKind = "svg"
)

// Definition is the component-owned visual contract consumed by native design
// tools and executable-preview adapters.
type Definition struct {
	SourceOfTruth     SourceOfTruth      `json:"source_of_truth,omitempty"`
	CanonicalExample  string             `json:"canonical_example,omitempty"`
	ExampleVariants   bool               `json:"example_variants,omitempty"`
	VariantExamples   []string           `json:"variant_examples,omitempty"`
	VariantMatrix     []string           `json:"variant_matrix,omitempty"`
	ExampleDefaults   *ExampleContract   `json:"example_defaults,omitempty"`
	Taxonomy          *Taxonomy          `json:"taxonomy,omitempty"`
	Root              *Node              `json:"root,omitempty"`
	Assets            []Asset            `json:"assets,omitempty"`
	InteractiveStates []InteractiveState `json:"interactive_states,omitempty"`
}

// InteractiveState documents a browser pseudo-state without multiplying
// native component variants.
type InteractiveState struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Classes     string `json:"classes"`
}

// Taxonomy controls deterministic library and flow placement.
type Taxonomy struct {
	LibraryPage    string `json:"library_page,omitempty"`
	LibrarySection string `json:"library_section,omitempty"`
	FlowGroup      string `json:"flow_group,omitempty"`
	FlowOrder      int    `json:"flow_order,omitempty"`
	ExcludeLibrary bool   `json:"exclude_library,omitempty"`
}

// Asset declares an image or vector owned by a component blueprint.
type Asset struct {
	Name        string    `json:"name"`
	Kind        AssetKind `json:"kind"`
	Source      string    `json:"source,omitempty"`
	Description string    `json:"description,omitempty"`
}

// Node is a semantic visual tree. Classes carry the renderer-neutral utility
// vocabulary resolved by adapters; Props carry explicit conditional and
// binding instructions. Nodes never contain screenshots.
type Node struct {
	Kind          NodeKind                     `json:"kind"`
	Name          string                       `json:"name"`
	Classes       string                       `json:"classes,omitempty"`
	ClassBindings map[string]map[string]string `json:"class_bindings,omitempty"`
	Text          string                       `json:"text,omitempty"`
	Slot          string                       `json:"slot,omitempty"`
	AssetRef      string                       `json:"asset_ref,omitempty"`
	Props         map[string]any               `json:"props,omitempty"`
	Children      []Node                       `json:"children,omitempty"`
}

// ExampleContract declares the exact viewport and mode in which one authored
// fixture participates in design delivery.
type ExampleContract struct {
	Canonical bool     `json:"canonical,omitempty"`
	Viewport  string   `json:"viewport,omitempty"`
	Viewports []string `json:"viewports,omitempty"`
	ColorMode string   `json:"color_mode,omitempty"`
	Density   string   `json:"density,omitempty"`
}
