# pk-design

Provider-neutral design contracts for the OSS PlatformKit repos.

`pk-design` is intentionally small. It defines the stable design primitives that
modules, apps, renderers, and Pro distributions can extend without importing
frontend runtime code or private product packages.

## Core Surface

- `pkg/tokens`: DTCG-native token sets, import/export, safe reference/value helpers, `$root`, `$extends`, validation reports, and CSS custom-property export
- `pkg/themes`: token-first theme overlays, explicit layer resolution, and canonical `Stack` composition
- `pkg/components`: renderer-neutral component descriptors
- `pkg/catalog`: deterministic contribution catalog and manifests for modules and apps

Renderer adapters, Tailwind config generation, Figma import/export, Storybook
metadata, and client-specific surfaces belong outside this core.

## Extension Model

Modules contribute token sets, themes, and component descriptors through
`catalog.Contribution`. Each contribution may include a manifest with schema,
semantic version, compatibility range, capabilities, and deprecation metadata.
Apps compose those contributions into a `Catalog`, then their renderer of choice
can transform the catalog into CSS, native tokens, component docs, previews, or
runtime UI metadata.

The core packages validate inputs, sort deterministic lists, and return
defensive copies so downstream extensions cannot mutate shared catalog state.

See [docs/CORE_CONTRACT.md](docs/CORE_CONTRACT.md) for the package boundaries
and invariants. `docs/block-manifest.json` is the machine-readable public block
inventory that CI validates for v0.0.0 release readiness.

## Verify

```bash
make verify
make staticcheck
make cover
```

Every public package includes executable examples. Run them with:

```bash
go test ./... -run Example -v
```
