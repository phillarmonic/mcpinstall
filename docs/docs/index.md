---
title: mcpinstall
description: >-
  mcpinstall registers a stdio MCP server with the configuration files of the
  agent hosts that support one and writes the matching agent guidance.
icon: lucide/plug-zap
---

# mcpinstall

`mcpinstall` is a Go module for distributing one stdio
[MCP](https://modelcontextprotocol.io/) server into the agent hosts a project
supports. A caller supplies a [Spec](embedding.md) that names the server and
points at its embedded guidance; the module owns the per-host configuration
adapters, the JSON, JSONC and TOML merging, the managed guidance sections, and
the `install` and `uninstall` command set.

Every MCP project otherwise rewrites the same code: a Claude config writer, a
Cursor rule, a VS Code `mcp.json` merge, a Codex TOML table, an OpenCode entry,
and the CLI around them. This module implements that once, so a project supplies
only its name and its guidance.

## What it does

- Merges the server entry into each host's configuration without disturbing the
  rest of the file, and preserves a file it cannot parse instead of overwriting
  it. See [Hosts](hosts.md#merge-and-safety-rules).
- Removes the containers and files an uninstall empties.
- Writes agent guidance, using a managed section for hosts that share a file
  with the user. See [Guidance](guidance.md).
- Cleans up server entries and guidance left by a former server name. See
  [`uninstall-legacy`](cli.md#uninstall-legacy).
- Exposes one command surface to every caller through `cli.Run`. See the
  [CLI reference](cli.md).

## Supported hosts

| Host | Catalog name | Scopes | Configuration | Server key | Format |
| --- | --- | --- | --- | --- | --- |
| Claude Code | `claude` | project, global | `.mcp.json`, `~/.claude.json` | `mcpServers` | JSON |
| Cursor | `cursor` | project, global | `.cursor/mcp.json` | `mcpServers` | JSON |
| Copilot in VS Code | `copilot` | project, global | `.vscode/mcp.json`, VS Code profile | `servers` | JSON |
| Copilot for JetBrains | `copilot-jetbrains` | global | `~/.config/github-copilot/intellij/mcp.json` | `servers` | JSON |
| Codex | `codex` | project, global | `.codex/config.toml` | `mcp_servers` | TOML |
| OpenCode | `opencode` | project, global | `opencode.json`, `opencode.jsonc` | `mcp` | JSON |

`--host all` expands to every catalog name. [Hosts](hosts.md) gives the exact
paths and the per-host details.

## At a glance

| | |
| --- | --- |
| Module | [`github.com/phillarmonic/mcpinstall`](https://github.com/phillarmonic/mcpinstall) |
| Go | 1.26 |
| Dependency | [`github.com/pelletier/go-toml/v2`](https://github.com/pelletier/go-toml) |
| Commands | `install`, `uninstall`, `uninstall-legacy`, `install-guidance`, `uninstall-guidance`, `global-guidance`, `hosts`, `version` |
| Guidance markers | `<!-- <marker>-guidance:begin -->` ... `<!-- <marker>-guidance:end -->` |

## Where to go next

- [Installation](installation.md) adds the module to a project and wires the CLI.
- [Hosts](hosts.md) covers every configuration file the adapters touch.
- The [CLI reference](cli.md) lists each command, flag, and output shape.
- [Guidance](guidance.md) explains the harness files and managed sections.
- [Embedding](embedding.md) describes the `Spec` a caller supplies.
- [Architecture](architecture.md) maps the packages and the design rules.
