# pk-design

Minimal design-token engine for the OSS PlatformKit repos.

This repo owns token vocabulary and export adapters. It should not own frontend
runtime rendering or app composition.

## Current Surface

- `pkg/tokens`: semantic token set and CSS custom-property export

## Verify

```bash
go test ./...
```
