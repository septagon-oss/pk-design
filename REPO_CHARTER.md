# pk-design Charter

## Purpose

Design tokens, themes, component descriptions, and contribution catalog contracts for PlatformKit OSS. Provider-neutral primitives that back UI toolkits across any rendering target.

## In Scope

- Design tokens (`pkg/tokens`): DTCG-compatible token resolution
- Theme system (`pkg/themes`): typed overlays and composition
- Component descriptors (`pkg/components`): role, slot, and behaviour contracts
- Contribution catalog (`pkg/catalog`): registry of UI capabilities

## Out of Scope

- Renderers, view engines, or framework-specific adapters
- CSS, HTML, or JavaScript bundles
- Accessibility runtime helpers (live in platformkit-ui)
- Marketing or brand-specific assets (colours, logos)

## Dependencies

None (zero-dependency module).
