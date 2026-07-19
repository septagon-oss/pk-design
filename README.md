# pk-design

[![Go Reference](https://pkg.go.dev/badge/github.com/septagon-oss/pk-design.svg)](https://pkg.go.dev/github.com/septagon-oss/pk-design)
[![CI](https://github.com/septagon-oss/pk-design/actions/workflows/go.yml/badge.svg)](https://github.com/septagon-oss/pk-design/actions/workflows/go.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

`pk-design` provides provider-neutral design contracts for the OSS PlatformKit family. It is intentionally small: it defines the stable design primitives — DTCG-native token sets, theme overlays, component descriptors, and contribution catalogs — that modules, apps, renderers, and downstream distributions can extend without importing frontend runtime code or private product packages.

## Install

```bash
go get github.com/septagon-oss/pk-design@v0.1.0
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/septagon-oss/pk-design/pkg/tokens"
)

func main() {
	css, err := tokens.CSSVars(tokens.Set{
		Name: "pk",
		Values: map[string]tokens.Value{
			"color.text.primary": "#111827",
		},
		Types: map[string]tokens.Type{
			"color.text.primary": tokens.TypeColor,
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(css)
	// :root {
	//   --pk-color-text-primary: #111827;
	// }
}
```

## Current Surface

- `pkg/tokens`: DTCG-native token sets, import/export, safe reference/value helpers, `$root`, `$extends`, validation reports, and CSS custom-property export
- `pkg/themes`: token-first theme overlays, explicit layer resolution, and canonical `Stack` composition
- `pkg/components`: renderer-neutral component descriptors (props, slots, variants, anatomy, token dependencies)
- `pkg/catalog`: deterministic contribution catalog and manifests for modules and apps

Renderer adapters, Tailwind config generation, Figma import/export, Storybook
metadata, and client-specific surfaces belong outside this core. See
[docs/CORE_CONTRACT.md](docs/CORE_CONTRACT.md) for the package boundaries and
invariants; `docs/block-manifest.json` is the machine-readable public block
inventory that CI validates for release readiness.

### Extension Model

Modules contribute token sets, themes, and component descriptors through
`catalog.Contribution`. Each contribution may include a manifest with schema,
semantic version, compatibility range, and capabilities.
Apps compose those contributions into a `Catalog`, then their renderer of choice
can transform the catalog into CSS, native tokens, component docs, previews, or
runtime UI metadata.

The core packages validate inputs, sort deterministic lists, and return
defensive copies so downstream extensions cannot mutate shared catalog state.
Every public package ships executable examples; run them with
`go test ./... -run Example -v`.

## Verify

```bash
make verify   # go test + go vet + staticcheck + race
```

## License

Apache-2.0. See [LICENSE](LICENSE).
