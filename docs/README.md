# mcpinstall documentation

This directory holds the Zensical documentation project for
[`mcpinstall`](https://github.com/phillarmonic/mcpinstall). The published site
is built from `docs/` with:

```bash
uv sync
uv run zensical serve
uv run zensical build --clean --strict
```
