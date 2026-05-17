# Contributing

This repository is the small OSS upstream for PlatformKit.

Early contributions should preserve the minimal surface:

- keep core framework packages provider-neutral
- do not add private `septagon-dev` imports
- do not add client, demo, staging, or hosted-cloud assumptions
- keep renderers, Tailwind, Figma, Storybook, and app composition outside core
- return defensive copies from registries or catalogs that expose mutable data
- make normalization deterministic and fail fast on invalid extension metadata
- add tests for contract behavior before expanding APIs

Run before opening a pull request:

```bash
make verify
```
