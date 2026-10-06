---
title: Development
description: Building, testing, and documenting mcpinstall, and consuming a local checkout.
icon: lucide/hammer
---

# Development

The module is plain Go with a single runtime dependency. The documentation site
is a small uv project under `docs/`.

## Layout

```text
mcpinstall/
  spec.go            the Spec type
  cli/               the command surface
  install/           request orchestration, guidance, legacy cleanup
  platform/          the adapter contract and shared writers
  platform/<host>/   one adapter per host
  docs/              the Zensical documentation project
```

## Build and test

```bash
go build ./...
go test ./...
go vet ./...
```

The module targets Go 1.26. Its only runtime dependency is
`github.com/pelletier/go-toml/v2`, used by the Codex TOML writer.

## Documentation

The site is a Zensical project rooted at `docs/`, with content in
`docs/docs/`. Serve it with live reload or build it strictly:

```bash
cd docs
uv sync
uv run zensical serve
uv run zensical build --clean --strict
```

Code blocks are highlighted by a bundled Shiki module in
`docs/docs/javascripts` with the matching stylesheet in
`docs/docs/stylesheets`. The glossary is `docs/docs/glossary.md`, enabled by
the `zensical_glossary` extension in `docs/zensical.toml`.

## Consuming a local checkout

Until the module is published, a consumer points at its checkout with a
`replace` directive:

```text
replace github.com/phillarmonic/mcpinstall => ../phillarmonic/mcpinstall
```

Keep that directive in the consuming project, not in this module.
