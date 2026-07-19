# pk-design Core Contract

`pk-design` owns the small OSS design contract that PlatformKit modules and
apps can share without depending on a renderer, build tool, hosted product, or
private repository.

## Scope

The core includes four primitives:

- `pkg/tokens`: DTCG-native token sets with stable import/export, safe reference/value helpers, alias resolution, `$root`, `$extends`, validation reports, and CSS export.
- `pkg/themes`: token-first theme overlays, explicit layer resolution, and canonical stack composition for base, primitive, semantic, module, app, client, tenant, platform, and accessibility layers.
- `pkg/components`: provider-neutral component descriptors for props, slots, variants, anatomy, and token requirements.
- `pkg/catalog`: deterministic aggregation of design contributions and manifests from modules and apps.

The core deliberately excludes:

- Tailwind, CSS-in-JS, native, Figma, Storybook, and documentation adapters.
- Runtime rendering and frontend component implementations.
- Client, demo, staging, tenant, billing, or hosted-cloud assumptions.
- Private `septagon-dev` imports.

## Extension Model

Modules should describe their design surface as a `catalog.Contribution`:

```go
catalog.Contribution{
    Manifest: catalog.Manifest{
        Source: "booking_management",
        SchemaVersion: catalog.ManifestSchemaVersion,
        Version: "1.0.0",
        Capabilities: []string{"tokens", "components"},
    },
    TokenSets: []tokens.Set{bookingTokens},
    Themes: []themes.Theme{bookingTheme},
    Components: []components.Descriptor{bookingCalendarDescriptor},
}
```

Apps compose contributions with `catalog.New().Add(...).Build()`. Downstream
packages can then adapt the catalog into Tailwind config, CSS files, native
resources, Figma metadata, Storybook stories, or runtime UI registries.

Pro packages should extend this contract by adding adapters and richer module
contributions. They should not fork the core semantics unless an OSS core
primitive is genuinely missing.

Theme stacks should use `themes.NewStack(...)` when callers want the canonical
PlatformKit order: base, primitive, semantic, module, app, client, tenant,
platform, accessibility. `themes.ResolveLayers(...)` remains available when a
caller intentionally needs explicit ordering.

## Invariants

- Inputs are normalized at package boundaries.
- Lists are sorted where ordering is not semantically meaningful.
- Duplicate contract keys fail by default.
- Token paths cannot be both a leaf and a parent group.
- Optional token metadata must reference declared tokens.
- Token aliases must resolve without missing references or cycles.
- DTCG JSON import preserves typed object values and JSON number fidelity.
- `$root` is supported as the final segment of a token path.
- Group `$extends` copies missing descendant tokens without overwriting explicit target tokens.
- Theme overlays cannot change the base token namespace by accident.
- Layer/mode resolution applies global layers plus layers matching the requested mode.
- Canonical theme stacks preserve contribution order inside the same layer kind.
- Component enum defaults must be declared enum values.
- Contribution manifests declare source, schema, semantic version, capabilities, and compatibility.
- Manifest compatibility ranges must be valid semantic versions and cannot be inverted.
- Catalog builders snapshot contributions when they are added.
- Catalog reads return defensive copies of maps and slices.
- Core packages do not import frontend, Figma, Storybook, Tailwind, or private code.

These invariants make the package safe for long-lived module ecosystems: a
module can contribute design data independently, an app can compose it
deterministically, and each renderer can evolve outside the core.

## Block Manifest

`docs/block-manifest.json` is the machine-readable release inventory for the
design core. CI validates that every public design block declares identity,
ownership, version, package, status, contracts, composition laws, extension
points, and evidence files.

The manifest is intentionally small. It does not try to describe every helper;
it describes only the public blocks that modules and downstream adapters are
expected to compose.
