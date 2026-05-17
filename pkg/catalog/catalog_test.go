package catalog

// catalog_test.go validates contribution aggregation, duplicate-key policies,
// deterministic entries, and immutable catalog reads.
//
// ADR: ADR-0029 (file purpose declaration).
// Convention: C-10 (shared builders return errors), C-14 (every Go file declares its purpose).

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/septagon-oss/pk-design/pkg/components"
	"github.com/septagon-oss/pk-design/pkg/themes"
	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func testSet(name, value string) tokens.Set {
	return tokens.Set{
		Name: name,
		Values: map[string]tokens.Value{
			"color.text.primary": value,
		},
		Types: map[string]tokens.Type{
			"color.text.primary": tokens.TypeColor,
		},
		Extensions: map[string]map[string]any{
			"color.text.primary": {"source": name},
		},
		Groups: map[string]tokens.Group{
			"color.text": {
				Type:       tokens.TypeColor,
				Extensions: map[string]any{"scope": "content"},
			},
		},
		Metadata: map[string]any{"owner": "design"},
	}
}

func testTheme(id string) themes.Theme {
	return themes.Theme{
		ID: id,
		Tokens: tokens.Set{
			Name:   "pk",
			Values: map[string]tokens.Value{"color.surface.primary": "#ffffff"},
			Types:  map[string]tokens.Type{"color.surface.primary": tokens.TypeColor},
		},
		Metadata: map[string]string{"owner": "design"},
	}
}

func testDescriptor(id string) components.Descriptor {
	return components.Descriptor{
		ID:       id,
		Category: components.CategoryAtom,
		Props: []components.Prop{
			{Name: "tone", Type: components.PropEnum, EnumValues: []string{"brand", "neutral"}, Default: "brand"},
		},
		Anatomy: []components.AnatomyNode{
			{Name: "root", Tokens: []string{"color.surface.primary"}},
		},
		Metadata: map[string]string{"owner": "design"},
	}
}

func TestBuildAggregatesDeterministicCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := New().
		Add(Contribution{
			Source:     " base ",
			TokenSets:  []tokens.Set{testSet("zeta", "#111111"), testSet("alpha", "#222222")},
			Themes:     []themes.Theme{testTheme("light")},
			Components: []components.Descriptor{testDescriptor("button.primary")},
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	tokenEntries := catalog.TokenSetEntries()
	if got := []string{tokenEntries[0].Key, tokenEntries[1].Key}; !slices.Equal(got, []string{"alpha", "zeta"}) {
		t.Fatalf("TokenSetEntries() keys = %v", got)
	}
	if tokenEntries[0].Source != "base" {
		t.Fatalf("TokenSetEntries() source = %q", tokenEntries[0].Source)
	}

	themeEntries := catalog.ThemeEntries()
	if len(themeEntries) != 1 || themeEntries[0].Key != "light" {
		t.Fatalf("ThemeEntries() = %#v", themeEntries)
	}
	componentEntries := catalog.ComponentEntries()
	if len(componentEntries) != 1 || componentEntries[0].Key != "button.primary" {
		t.Fatalf("ComponentEntries() = %#v", componentEntries)
	}
	manifest, ok := catalog.Manifest("base")
	if !ok {
		t.Fatal("Manifest() ok = false")
	}
	if manifest.SchemaVersion != ManifestSchemaVersion || manifest.Source != "base" {
		t.Fatalf("Manifest() = %#v", manifest)
	}
}

func TestCatalogReadsAreDefensiveCopies(t *testing.T) {
	t.Parallel()

	catalog, err := New().
		Add(Contribution{
			Source:     "base",
			TokenSets:  []tokens.Set{testSet("pk", "#111111")},
			Themes:     []themes.Theme{testTheme("light")},
			Components: []components.Descriptor{testDescriptor("button.primary")},
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	set, ok := catalog.TokenSet(" pk ")
	if !ok {
		t.Fatal("TokenSet() ok = false")
	}
	set.Values["color.text.primary"] = "#000000"
	set.Extensions["color.text.primary"]["source"] = "mutated"
	again, _ := catalog.TokenSet("pk")
	if again.Values["color.text.primary"] != "#111111" || again.Extensions["color.text.primary"]["source"] != "pk" {
		t.Fatalf("TokenSet() returned aliased maps: %#v", again)
	}
	set.Groups["color.text"].Extensions["scope"] = "mutated"
	again, _ = catalog.TokenSet("pk")
	if again.Groups["color.text"].Extensions["scope"] != "content" {
		t.Fatalf("TokenSet() returned aliased groups: %#v", again.Groups)
	}

	theme, ok := catalog.Theme("light")
	if !ok {
		t.Fatal("Theme() ok = false")
	}
	theme.Tokens.Values["color.surface.primary"] = "#000000"
	theme.Metadata["owner"] = "mutated"
	themeAgain, _ := catalog.Theme("light")
	if themeAgain.Tokens.Values["color.surface.primary"] != "#ffffff" || themeAgain.Metadata["owner"] != "design" {
		t.Fatalf("Theme() returned aliased data: %#v", themeAgain)
	}

	descriptor, ok := catalog.Component("button.primary")
	if !ok {
		t.Fatal("Component() ok = false")
	}
	descriptor.Props[0].EnumValues[0] = "mutated"
	descriptor.Anatomy[0].Tokens[0] = "mutated"
	componentAgain, _ := catalog.Component("button.primary")
	if componentAgain.Props[0].EnumValues[0] != "brand" || componentAgain.Anatomy[0].Tokens[0] != "color.surface.primary" {
		t.Fatalf("Component() returned aliased data: %#v", componentAgain)
	}
}

func TestBuilderAddSnapshotsContribution(t *testing.T) {
	t.Parallel()

	set := testSet("pk", "#111111")
	contribution := Contribution{
		Source:    "base",
		TokenSets: []tokens.Set{set},
	}
	builder := New().Add(contribution)
	contribution.Source = "mutated"
	contribution.TokenSets[0].Values["color.text.primary"] = "#000000"

	catalog, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	entries := catalog.TokenSetEntries()
	if len(entries) != 1 || entries[0].Source != "base" {
		t.Fatalf("TokenSetEntries() = %#v", entries)
	}
	catalogSet, _ := catalog.TokenSet("pk")
	if catalogSet.Values["color.text.primary"] != "#111111" {
		t.Fatalf("Add() did not snapshot contribution: %#v", catalogSet.Values)
	}
}

func TestConflictPolicies(t *testing.T) {
	t.Parallel()

	first := Contribution{Source: "first", TokenSets: []tokens.Set{testSet("pk", "#111111")}}
	second := Contribution{Source: "second", TokenSets: []tokens.Set{testSet("pk", "#222222")}}

	if _, err := New().Add(first).Add(second).Build(); err == nil {
		t.Fatal("Build() should reject duplicate keys by default")
	}

	catalog, err := New().WithConflictPolicy(ConflictFirstWins).Add(first).Add(second).Build()
	if err != nil {
		t.Fatalf("Build() first-wins error = %v", err)
	}
	set, _ := catalog.TokenSet("pk")
	if set.Values["color.text.primary"] != "#111111" {
		t.Fatalf("first-wins set = %#v", set.Values)
	}

	catalog, err = New().WithConflictPolicy(ConflictLastWins).Add(first).Add(second).Build()
	if err != nil {
		t.Fatalf("Build() last-wins error = %v", err)
	}
	set, _ = catalog.TokenSet("pk")
	if set.Values["color.text.primary"] != "#222222" {
		t.Fatalf("last-wins set = %#v", set.Values)
	}
}

func TestManifestNormalization(t *testing.T) {
	t.Parallel()

	catalog, err := New().
		Add(Contribution{
			Manifest: Manifest{
				Source:  "module.booking",
				Version: " 1.2.3 ",
				Compatibility: Compatibility{
					MinCoreVersion: " 0.1.0 ",
					MaxCoreVersion: " v2.0.0 ",
				},
				Capabilities: []string{"tokens", "components", "tokens"},
				Deprecations: []Deprecation{
					{Path: "color.old", Replacement: "color.new", Message: "Use the semantic token"},
					{Path: " "},
				},
				Metadata: map[string]any{" owner ": "design"},
			},
			TokenSets: []tokens.Set{testSet("pk", "#111111")},
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	manifest, ok := catalog.Manifest("module.booking")
	if !ok {
		t.Fatal("Manifest() ok = false")
	}
	if manifest.Version != "1.2.3" || manifest.Compatibility.MinCoreVersion != "0.1.0" {
		t.Fatalf("Manifest() version fields = %#v", manifest)
	}
	if manifest.Compatibility.MaxCoreVersion != "v2.0.0" {
		t.Fatalf("Manifest() max core version = %#v", manifest.Compatibility)
	}
	if !slices.Equal(manifest.Capabilities, []string{"components", "tokens"}) {
		t.Fatalf("Manifest() capabilities = %#v", manifest.Capabilities)
	}
	if len(manifest.Deprecations) != 1 || manifest.Deprecations[0].Path != "color.old" {
		t.Fatalf("Manifest() deprecations = %#v", manifest.Deprecations)
	}

	manifest.Metadata["owner"] = "mutated"
	again, _ := catalog.Manifest("module.booking")
	if again.Metadata["owner"] != "design" {
		t.Fatalf("Manifest() returned aliased metadata: %#v", again.Metadata)
	}
}

func TestManifestEntriesAreDeterministicDefensiveCopies(t *testing.T) {
	t.Parallel()

	rawMetadata := json.RawMessage(`{"owner":"design"}`)
	catalog, err := New().
		Add(Contribution{
			Manifest: Manifest{
				Source:   "module.zeta",
				Metadata: map[string]any{"raw": rawMetadata, "labels": map[string]string{"owner": "design"}, "tags": []string{"homepage"}},
			},
			TokenSets: []tokens.Set{testSet("zeta", "#111111")},
		}).
		Add(Contribution{
			Manifest: Manifest{
				Source: "module.alpha",
			},
			TokenSets: []tokens.Set{testSet("alpha", "#222222")},
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	entries := catalog.ManifestEntries()
	if got := []string{entries[0].Key, entries[1].Key}; !slices.Equal(got, []string{"module.alpha", "module.zeta"}) {
		t.Fatalf("ManifestEntries() keys = %v", got)
	}
	raw := entries[1].Value.Metadata["raw"].(json.RawMessage)
	raw[0] = '['
	entries[1].Value.Metadata["labels"].(map[string]string)["owner"] = "mutated"
	entries[1].Value.Metadata["tags"].([]string)[0] = "mutated"
	again := catalog.ManifestEntries()
	if string(again[1].Value.Metadata["raw"].(json.RawMessage)) != `{"owner":"design"}` {
		t.Fatalf("ManifestEntries() returned aliased raw metadata: %#v", again[1].Value.Metadata)
	}
	if again[1].Value.Metadata["labels"].(map[string]string)["owner"] != "design" {
		t.Fatalf("ManifestEntries() returned aliased string-map metadata: %#v", again[1].Value.Metadata)
	}
	if again[1].Value.Metadata["tags"].([]string)[0] != "homepage" {
		t.Fatalf("ManifestEntries() returned aliased string-slice metadata: %#v", again[1].Value.Metadata)
	}
}

func TestManifestRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := New().Add(Contribution{
		Source:   "module.a",
		Manifest: Manifest{Source: "module.b"},
		TokenSets: []tokens.Set{
			testSet("pk", "#111111"),
		},
	}).Build(); err == nil {
		t.Fatal("Build() should reject contribution source mismatch")
	}
	if _, err := New().Add(Contribution{
		Manifest: Manifest{Source: "module.a", SchemaVersion: "unknown"},
		TokenSets: []tokens.Set{
			testSet("pk", "#111111"),
		},
	}).Build(); err == nil {
		t.Fatal("Build() should reject unknown manifest schema")
	}
	if _, err := New().Add(Contribution{
		Manifest: Manifest{Source: "module.a", Version: "1.2"},
		TokenSets: []tokens.Set{
			testSet("pk", "#111111"),
		},
	}).Build(); err == nil {
		t.Fatal("Build() should reject invalid manifest semver")
	}
	if _, err := New().Add(Contribution{
		Manifest: Manifest{
			Source: "module.a",
			Compatibility: Compatibility{
				MinCoreVersion: "2.0.0",
				MaxCoreVersion: "1.0.0",
			},
		},
		TokenSets: []tokens.Set{
			testSet("pk", "#111111"),
		},
	}).Build(); err == nil {
		t.Fatal("Build() should reject inverted manifest compatibility range")
	}
}

func TestBuildRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := New().Add(Contribution{Source: " "}).Build(); err == nil {
		t.Fatal("Build() should reject empty contribution source")
	}
	if _, err := New().WithConflictPolicy(ConflictPolicy(99)).Build(); err == nil {
		t.Fatal("Build() should reject unknown conflict policy")
	}
	if _, err := New().Add(Contribution{Source: "base", TokenSets: []tokens.Set{{Name: "bad name"}}}).Build(); err == nil {
		t.Fatal("Build() should reject invalid token set")
	}

	var builder *Builder
	catalog, err := builder.Build()
	if err != nil {
		t.Fatalf("nil Builder.Build() error = %v", err)
	}
	if len(catalog.TokenSetEntries()) != 0 || len(catalog.ThemeEntries()) != 0 || len(catalog.ComponentEntries()) != 0 {
		t.Fatalf("nil Builder.Build() catalog = %#v", catalog)
	}
}

func TestSemanticVersionValidation(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{
		"0.0.0",
		"1.2.3",
		"v1.2.3-alpha.1+build.7",
	} {
		if _, ok := parseSemanticVersion(valid); !ok {
			t.Fatalf("parseSemanticVersion(%q) ok = false", valid)
		}
	}
	for _, invalid := range []string{
		"1.2",
		"01.2.3",
		"1.2.3-alpha.01",
		"1.2.3+",
	} {
		if _, ok := parseSemanticVersion(invalid); ok {
			t.Fatalf("parseSemanticVersion(%q) ok = true", invalid)
		}
	}

	alpha, _ := parseSemanticVersion("1.0.0-alpha.1")
	alphaTwo, _ := parseSemanticVersion("1.0.0-alpha.2")
	alphaBeta, _ := parseSemanticVersion("1.0.0-alpha.beta")
	release, _ := parseSemanticVersion("1.0.0")
	if compareSemanticVersions(alpha, alphaTwo) >= 0 {
		t.Fatal("numeric pre-release identifiers should compare numerically")
	}
	if compareSemanticVersions(alphaTwo, alphaBeta) >= 0 {
		t.Fatal("numeric pre-release identifiers should sort before non-numeric identifiers")
	}
	if compareSemanticVersions(alpha, release) >= 0 {
		t.Fatal("pre-release version should compare lower than release")
	}
}

func TestZeroValueCatalogIsReadable(t *testing.T) {
	t.Parallel()

	var catalog Catalog
	if _, ok := catalog.TokenSet("pk"); ok {
		t.Fatal("zero-value catalog TokenSet() ok = true")
	}
	if _, ok := catalog.Theme("light"); ok {
		t.Fatal("zero-value catalog Theme() ok = true")
	}
	if _, ok := catalog.Component("button.primary"); ok {
		t.Fatal("zero-value catalog Component() ok = true")
	}
	if len(catalog.TokenSetEntries()) != 0 || len(catalog.ThemeEntries()) != 0 || len(catalog.ComponentEntries()) != 0 {
		t.Fatalf("zero-value catalog entries should be empty")
	}
}
