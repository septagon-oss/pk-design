// Package catalog builds deterministic design contribution catalogs.
package catalog

// catalog.go owns the pk-design extension point: modules, apps, and downstream
// distributions contribute token sets, themes, and component descriptors through
// one provider-neutral catalog without importing renderers or private code.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/septagon-oss/pk-design/pkg/components"
	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

// ManifestSchemaVersion is the current contribution-manifest schema.
const ManifestSchemaVersion = "pk.design.contribution.v1"

// ConflictPolicy controls duplicate contribution keys.
type ConflictPolicy int

const (
	ConflictReject ConflictPolicy = iota
	ConflictFirstWins
	ConflictLastWins
)

// Contribution is one design extension bundle from a module, app, or product.
type Contribution struct {
	Source     string
	Manifest   Manifest
	TokenSets  []tokens.Set
	Themes     []themes.Theme
	Components []components.Descriptor
}

// Manifest describes a contribution's compatibility and extension metadata.
type Manifest struct {
	SchemaVersion string
	Source        string
	Version       string
	Compatibility Compatibility
	Capabilities  []string
	Deprecations  []Deprecation
	Metadata      map[string]any
}

// Compatibility declares the pk-design version range a contribution expects.
type Compatibility struct {
	MinCoreVersion string
	MaxCoreVersion string
}

// Deprecation declares a token or component contract that downstream tooling
// should phase out.
type Deprecation struct {
	Path        string
	Replacement string
	Message     string
}

// Entry records the source that contributed a normalized value.
type Entry[T any] struct {
	Key    string
	Source string
	Value  T
}

// Catalog is an immutable view of normalized design contributions.
type Catalog struct {
	tokenSets  map[string]Entry[tokens.Set]
	themes     map[string]Entry[themes.Theme]
	components map[string]Entry[components.Descriptor]
	manifests  map[string]Entry[Manifest]
}

// Builder gathers contributions and validates them into a Catalog.
type Builder struct {
	contributions []Contribution
	conflict      ConflictPolicy
}

// New creates a design catalog builder.
func New() *Builder {
	return &Builder{conflict: ConflictReject}
}

// WithConflictPolicy sets duplicate-key behavior.
func (b *Builder) WithConflictPolicy(policy ConflictPolicy) *Builder {
	if b == nil {
		return b
	}
	b.conflict = policy
	return b
}

// Add appends a contribution.
func (b *Builder) Add(contribution Contribution) *Builder {
	if b == nil {
		return b
	}
	b.contributions = append(b.contributions, copyContribution(contribution))
	return b
}

// Build validates all contributions and returns an immutable catalog.
func (b *Builder) Build() (*Catalog, error) {
	catalog := &Catalog{
		tokenSets:  map[string]Entry[tokens.Set]{},
		themes:     map[string]Entry[themes.Theme]{},
		components: map[string]Entry[components.Descriptor]{},
		manifests:  map[string]Entry[Manifest]{},
	}
	if b == nil {
		return catalog, nil
	}
	if err := validateConflictPolicy(b.conflict); err != nil {
		return nil, err
	}
	for _, contribution := range b.contributions {
		manifest, err := normalizeManifest(contribution.Manifest, contribution.Source)
		if err != nil {
			return nil, err
		}
		source := manifest.Source
		if source == "" {
			return nil, fmt.Errorf("design contribution source is required")
		}
		if source != "" {
			if err := insert(catalog.manifests, source, source, manifest, b.conflict); err != nil {
				return nil, err
			}
		}
		for _, set := range contribution.TokenSets {
			normalized, err := set.Normalize()
			if err != nil {
				return nil, fmt.Errorf("design contribution %q token set: %w", source, err)
			}
			if err := insert(catalog.tokenSets, normalized.Name, source, normalized, b.conflict); err != nil {
				return nil, err
			}
		}
		for _, theme := range contribution.Themes {
			normalized, err := theme.Normalize()
			if err != nil {
				return nil, fmt.Errorf("design contribution %q theme: %w", source, err)
			}
			if err := insert(catalog.themes, normalized.ID, source, normalized, b.conflict); err != nil {
				return nil, err
			}
		}
		for _, descriptor := range contribution.Components {
			normalized, err := descriptor.Normalize()
			if err != nil {
				return nil, fmt.Errorf("design contribution %q component: %w", source, err)
			}
			if err := insert(catalog.components, normalized.ID, source, normalized, b.conflict); err != nil {
				return nil, err
			}
		}
	}
	return catalog, nil
}

// TokenSet returns a token set by name.
func (c *Catalog) TokenSet(name string) (tokens.Set, bool) {
	if c == nil {
		return tokens.Set{}, false
	}
	entry, ok := lookup(c.tokenSets, name)
	if !ok {
		return tokens.Set{}, false
	}
	return copyTokenSet(entry.Value), true
}

// Theme returns a theme by ID.
func (c *Catalog) Theme(id string) (themes.Theme, bool) {
	if c == nil {
		return themes.Theme{}, false
	}
	entry, ok := lookup(c.themes, id)
	if !ok {
		return themes.Theme{}, false
	}
	return copyTheme(entry.Value), true
}

// Component returns a component descriptor by ID.
func (c *Catalog) Component(id string) (components.Descriptor, bool) {
	if c == nil {
		return components.Descriptor{}, false
	}
	entry, ok := lookup(c.components, id)
	if !ok {
		return components.Descriptor{}, false
	}
	return copyDescriptor(entry.Value), true
}

// TokenSetEntries returns token-set entries in deterministic key order.
func (c *Catalog) TokenSetEntries() []Entry[tokens.Set] {
	if c == nil {
		return nil
	}
	return tokenSetEntries(c.tokenSets)
}

// ThemeEntries returns theme entries in deterministic key order.
func (c *Catalog) ThemeEntries() []Entry[themes.Theme] {
	if c == nil {
		return nil
	}
	return themeEntries(c.themes)
}

// ComponentEntries returns component entries in deterministic key order.
func (c *Catalog) ComponentEntries() []Entry[components.Descriptor] {
	if c == nil {
		return nil
	}
	return componentEntries(c.components)
}

// Manifest returns a contribution manifest by source.
func (c *Catalog) Manifest(source string) (Manifest, bool) {
	if c == nil {
		return Manifest{}, false
	}
	entry, ok := lookup(c.manifests, source)
	if !ok {
		return Manifest{}, false
	}
	return copyManifest(entry.Value), true
}

// ManifestEntries returns contribution manifests in deterministic source order.
func (c *Catalog) ManifestEntries() []Entry[Manifest] {
	if c == nil {
		return nil
	}
	keys := sortedKeys(c.manifests)
	out := make([]Entry[Manifest], 0, len(keys))
	for _, key := range keys {
		entry := c.manifests[key]
		entry.Value = copyManifest(entry.Value)
		out = append(out, entry)
	}
	return out
}

func insert[T any](entries map[string]Entry[T], key, source string, value T, policy ConflictPolicy) error {
	key = strings.TrimSpace(key)
	if existing, exists := entries[key]; exists {
		switch policy {
		case ConflictReject:
			return fmt.Errorf("design catalog: duplicate key %q from %q already contributed by %q", key, source, existing.Source)
		case ConflictFirstWins:
			return nil
		case ConflictLastWins:
		default:
			return fmt.Errorf("design catalog: unknown conflict policy %d", policy)
		}
	}
	entries[key] = Entry[T]{Key: key, Source: source, Value: value}
	return nil
}

func lookup[T any](entries map[string]Entry[T], key string) (Entry[T], bool) {
	entry, ok := entries[strings.TrimSpace(key)]
	return entry, ok
}

func sortedKeys[T any](in map[string]Entry[T]) []string {
	return slices.Sorted(maps.Keys(in))
}

func tokenSetEntries(in map[string]Entry[tokens.Set]) []Entry[tokens.Set] {
	keys := sortedKeys(in)
	out := make([]Entry[tokens.Set], 0, len(keys))
	for _, key := range keys {
		entry := in[key]
		entry.Value = copyTokenSet(entry.Value)
		out = append(out, entry)
	}
	return out
}

func themeEntries(in map[string]Entry[themes.Theme]) []Entry[themes.Theme] {
	keys := sortedKeys(in)
	out := make([]Entry[themes.Theme], 0, len(keys))
	for _, key := range keys {
		entry := in[key]
		entry.Value = copyTheme(entry.Value)
		out = append(out, entry)
	}
	return out
}

func componentEntries(in map[string]Entry[components.Descriptor]) []Entry[components.Descriptor] {
	keys := sortedKeys(in)
	out := make([]Entry[components.Descriptor], 0, len(keys))
	for _, key := range keys {
		entry := in[key]
		entry.Value = copyDescriptor(entry.Value)
		out = append(out, entry)
	}
	return out
}

func copyTokenSet(value tokens.Set) tokens.Set {
	return tokens.Set{
		Name:         value.Name,
		Version:      value.Version,
		Values:       copyValueMap(value.Values),
		Types:        maps.Clone(value.Types),
		Descriptions: maps.Clone(value.Descriptions),
		Extensions:   copyNestedAnyMap(value.Extensions),
		Deprecated:   copyAnyMap(value.Deprecated),
		Groups:       copyGroups(value.Groups),
		Metadata:     copyAnyMap(value.Metadata),
	}
}

func copyContribution(value Contribution) Contribution {
	return Contribution{
		Source:     value.Source,
		Manifest:   copyManifest(value.Manifest),
		TokenSets:  copyTokenSets(value.TokenSets),
		Themes:     copyThemes(value.Themes),
		Components: copyDescriptors(value.Components),
	}
}

func copyManifest(value Manifest) Manifest {
	return Manifest{
		SchemaVersion: value.SchemaVersion,
		Source:        value.Source,
		Version:       value.Version,
		Compatibility: value.Compatibility,
		Capabilities:  slices.Clone(value.Capabilities),
		Deprecations:  copyDeprecations(value.Deprecations),
		Metadata:      copyAnyMap(value.Metadata),
	}
}

func copyDeprecations(values []Deprecation) []Deprecation {
	if len(values) == 0 {
		return nil
	}
	return slices.Clone(values)
}

func copyTokenSets(values []tokens.Set) []tokens.Set {
	if len(values) == 0 {
		return nil
	}
	out := make([]tokens.Set, len(values))
	for i, value := range values {
		out[i] = copyTokenSet(value)
	}
	return out
}

func copyThemes(values []themes.Theme) []themes.Theme {
	if len(values) == 0 {
		return nil
	}
	out := make([]themes.Theme, len(values))
	for i, value := range values {
		out[i] = copyTheme(value)
	}
	return out
}

func copyDescriptors(values []components.Descriptor) []components.Descriptor {
	if len(values) == 0 {
		return nil
	}
	out := make([]components.Descriptor, len(values))
	for i, value := range values {
		out[i] = copyDescriptor(value)
	}
	return out
}

func copyTheme(value themes.Theme) themes.Theme {
	return themes.Theme{
		ID:          value.ID,
		Name:        value.Name,
		Version:     value.Version,
		Description: value.Description,
		Extends:     slices.Clone(value.Extends),
		Tokens:      copyTokenSet(value.Tokens),
		Metadata:    maps.Clone(value.Metadata),
	}
}

func copyDescriptor(value components.Descriptor) components.Descriptor {
	out := components.Descriptor{
		ID:             value.ID,
		Name:           value.Name,
		Category:       value.Category,
		SourceOfTruth:  value.SourceOfTruth,
		Description:    value.Description,
		ModuleID:       value.ModuleID,
		Props:          slices.Clone(value.Props),
		Slots:          slices.Clone(value.Slots),
		Variants:       slices.Clone(value.Variants),
		RequiredTokens: slices.Clone(value.RequiredTokens),
		Anatomy:        copyAnatomy(value.Anatomy),
		Metadata:       maps.Clone(value.Metadata),
	}
	for i := range out.Props {
		out.Props[i].EnumValues = slices.Clone(out.Props[i].EnumValues)
	}
	for i := range out.Variants {
		out.Variants[i].Values = slices.Clone(out.Variants[i].Values)
	}
	return out
}

func copyAnatomy(values []components.AnatomyNode) []components.AnatomyNode {
	if len(values) == 0 {
		return nil
	}
	out := make([]components.AnatomyNode, len(values))
	for i, value := range values {
		out[i] = components.AnatomyNode{
			Name:     value.Name,
			Role:     value.Role,
			Tokens:   slices.Clone(value.Tokens),
			Children: copyAnatomy(value.Children),
			Metadata: maps.Clone(value.Metadata),
		}
	}
	return out
}

func copyGroups(values map[string]tokens.Group) map[string]tokens.Group {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]tokens.Group, len(values))
	for key, value := range values {
		out[key] = tokens.Group{
			Type:        value.Type,
			Description: value.Description,
			Extends:     value.Extends,
			Extensions:  copyAnyMap(value.Extensions),
		}
	}
	return out
}

func copyValueMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = copyAny(value)
	}
	return out
}

func copyAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = copyAny(value)
	}
	return out
}

func copyNestedAnyMap(values map[string]map[string]any) map[string]map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]map[string]any, len(values))
	for key, value := range values {
		out[key] = copyAnyMap(value)
	}
	return out
}

func copyAny(value any) any {
	switch typed := value.(type) {
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...)
	case map[string]any:
		return copyAnyMap(typed)
	case map[string]string:
		return maps.Clone(typed)
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = copyAny(child)
		}
		return out
	case []string:
		return slices.Clone(typed)
	default:
		return typed
	}
}

func validateConflictPolicy(policy ConflictPolicy) error {
	switch policy {
	case ConflictReject, ConflictFirstWins, ConflictLastWins:
		return nil
	default:
		return fmt.Errorf("design catalog: unknown conflict policy %d", policy)
	}
}

func normalizeManifest(manifest Manifest, contributionSource string) (Manifest, error) {
	source := strings.TrimSpace(contributionSource)
	manifestSource := strings.TrimSpace(manifest.Source)
	if source != "" && manifestSource != "" && source != manifestSource {
		return Manifest{}, fmt.Errorf("design contribution source %q does not match manifest source %q", source, manifestSource)
	}
	if source == "" {
		source = manifestSource
	}
	if source != "" && !validIdentifier(source) {
		return Manifest{}, fmt.Errorf("design contribution source %q is invalid", source)
	}
	schemaVersion := strings.TrimSpace(manifest.SchemaVersion)
	if schemaVersion == "" {
		schemaVersion = ManifestSchemaVersion
	}
	if schemaVersion != ManifestSchemaVersion {
		return Manifest{}, fmt.Errorf("design contribution %q manifest schema %q is not supported", source, schemaVersion)
	}
	version, _, _, err := normalizeSemverField(source, "version", manifest.Version)
	if err != nil {
		return Manifest{}, err
	}
	minCoreVersion, minCore, hasMinCore, err := normalizeSemverField(source, "minCoreVersion", manifest.Compatibility.MinCoreVersion)
	if err != nil {
		return Manifest{}, err
	}
	maxCoreVersion, maxCore, hasMaxCore, err := normalizeSemverField(source, "maxCoreVersion", manifest.Compatibility.MaxCoreVersion)
	if err != nil {
		return Manifest{}, err
	}
	if hasMinCore && hasMaxCore && compareSemanticVersions(minCore, maxCore) > 0 {
		return Manifest{}, fmt.Errorf("design contribution %q manifest minCoreVersion %q is greater than maxCoreVersion %q", source, minCoreVersion, maxCoreVersion)
	}
	out := Manifest{
		SchemaVersion: schemaVersion,
		Source:        source,
		Version:       version,
		Compatibility: Compatibility{
			MinCoreVersion: minCoreVersion,
			MaxCoreVersion: maxCoreVersion,
		},
		Capabilities: normalizeList(manifest.Capabilities),
		Deprecations: normalizeDeprecations(manifest.Deprecations),
		Metadata:     normalizeAnyMap(manifest.Metadata),
	}
	return out, nil
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func normalizeDeprecations(values []Deprecation) []Deprecation {
	out := make([]Deprecation, 0, len(values))
	for _, value := range values {
		deprecation := Deprecation{
			Path:        strings.TrimSpace(value.Path),
			Replacement: strings.TrimSpace(value.Replacement),
			Message:     strings.TrimSpace(value.Message),
		}
		if deprecation.Path != "" {
			out = append(out, deprecation)
		}
	}
	return out
}

func normalizeAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key != "" && value != nil {
			out[key] = copyAny(value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
