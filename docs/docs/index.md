---
title: mcpinstall
description: >-
  mcpinstall registers a stdio or HTTP MCP server with the configuration files
  of the agent hosts that support one, installs Agent Skills and lifecycle
  hooks, and writes the matching agent guidance.
icon: lucide/plug-zap
---

# mcpinstall

`mcpinstall` is a Go module for distributing one stdio or HTTP
[MCP](https://modelcontextprotocol.io/) server into the agent hosts a project
supports. A caller supplies a [Spec](embedding.md) that names the server and
points at its embedded guidance; the module owns the per-host configuration
adapters, the JSON, JSONC and TOML merging, the managed guidance sections, the
Agent Skills and lifecycle hook installers, and the complete command set.

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
- Installs Agent Skills into the Claude Code, `.agents`, Devin, Grok, Copilot
  CLI, and Hermes skill roots. See
  [Skills and hooks](skills-hooks.md#agent-skills).
- Installs lifecycle hooks into 17 native host configurations, mapping portable
  events onto each host's vocabulary and preserving other tools' entries. See
  [Skills and hooks](skills-hooks.md#lifecycle-hooks).
- Cleans up server entries and guidance left by a former server name. See
  [`uninstall-legacy`](cli.md#uninstall-legacy).
- Exposes one command surface to every caller through `cli.Run`. See the
  [CLI reference](cli.md).

## Supported hosts

The catalog covers the host matrix ported from ai-memory:

- **Project and global:** Claude Code, Cursor, Copilot in VS Code, Codex,
  OpenCode V1, and OpenCode V2.
- **Global only:** Claude Desktop, Gemini, OpenClaw, OMP, Antigravity, Zero,
  ZCode, Devin, Grok, Kimi, Kiro, Command Code, IBM Bob, Copilot CLI, Droid,
  Zed, Muse, Prime Agent, and Copilot for JetBrains.
- **Project only:** Swival.
- **Guidance only:** DeepSeek Harness and Pi, which have no fixed MCP file.

Most hosts accept both a stdio command and an HTTP URL. ZCode is HTTP-only, and
Claude Desktop bridges HTTP through `mcp-remote`. `--host all` expands to every
supported host; a host with no configuration for the requested scope is skipped
with a warning. [Hosts](hosts.md) gives the exact paths, keys, entry shapes, and
caveats.

## At a glance

| | |
| --- | --- |
| Module | [`github.com/phillarmonic/mcpinstall`](https://github.com/phillarmonic/mcpinstall) |
| Go | 1.26 |
| Dependency | [`github.com/pelletier/go-toml/v2`](https://github.com/pelletier/go-toml) |
| Commands | `install`, `uninstall`, `uninstall-legacy`, `install-guidance`, `uninstall-guidance`, `global-guidance`, `install-skills`, `uninstall-skills`, `install-hooks`, `uninstall-hooks`, `hosts`, `version` |
| Guidance markers | `<!-- <marker>-guidance:begin -->` ... `<!-- <marker>-guidance:end -->` |

## Where to go next

- [Installation](installation.md) adds the module to a project and wires the CLI.
- [Hosts](hosts.md) covers every configuration file the adapters touch.
- [Skills and hooks](skills-hooks.md) covers the extension roots, hook shapes,
  and portable event matrix.
- The [CLI reference](cli.md) lists each command, flag, and output shape.
- [Guidance](guidance.md) explains the harness files and managed sections.
- [Embedding](embedding.md) describes the `Spec` a caller supplies.
- [Architecture](architecture.md) maps the packages and the design rules.
