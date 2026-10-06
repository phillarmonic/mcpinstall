---
title: Hosts
description: The agent hosts mcpinstall configures, their files, keys, scope rules, and merge behavior.
icon: lucide/server
---

# Hosts

Each host has one small package under `platform/<host>` implementing the
[adapter](architecture.md) contract. The [catalog](architecture.md#the-catalog)
resolves a `--host` value onto the right adapter.

## Catalog

| Catalog name | Host | Scopes | Configuration file | Server key | Format |
| --- | --- | --- | --- | --- | --- |
| `claude` | Claude Code | project, global | `.mcp.json` (project), `~/.claude.json` (global) | `mcpServers` | JSON |
| `cursor` | Cursor | project, global | `.cursor/mcp.json` | `mcpServers` | JSON |
| `copilot` | Copilot in VS Code | project, global | `.vscode/mcp.json` (project), VS Code profile `mcp.json` (global) | `servers` | JSON |
| `copilot-jetbrains` | Copilot for JetBrains | global | `~/.config/github-copilot/intellij/mcp.json` | `servers` | JSON |
| `codex` | Codex (ChatGPT) | project, global | `.codex/config.toml` | `mcp_servers` | TOML |
| `opencode` | OpenCode | project, global | `opencode.json` or `opencode.jsonc` | `mcp` | JSON |

`--host all` expands to every name above. Values are case-insensitive, spaces
and underscores fold to hyphens, and `openai` and `chatgpt` resolve to `codex`.

## Scope

A [scope](glossary.md#scope) decides the root the adapter resolves its path
against:

| Scope | Root | Example result |
| --- | --- | --- |
| `project` (default) | the current directory, or `--root` | `./.mcp.json` |
| `global` | the home directory, or `--root` | `~/.claude.json` |

`--root` overrides both. It is useful for testing and for writing a
configuration into a directory that is not the working directory.

## Per-host details

### Claude Code

Project scope uses the dedicated `.mcp.json`. Global scope keeps the server
alongside the rest of Claude Code's user state in `~/.claude.json`, a file the
tool also maintains.

### Cursor

Both scopes use `.cursor/mcp.json`; the root decides whether that is the
repository or the home directory.

### Copilot in VS Code

Project scope writes the workspace `.vscode/mcp.json`. Global scope writes the
per-profile file, which lives outside the home directory on macOS and Windows:

| OS | Global file |
| --- | --- |
| macOS | `~/Library/Application Support/Code/User/mcp.json` |
| Windows | `%APPDATA%/Code/User/mcp.json` |
| Linux and other | `$XDG_CONFIG_HOME/Code/User/mcp.json`, or `~/.config/Code/User/mcp.json` |

VS Code names its server table `servers` and an earlier spelling used
`mcpServers`; the writer migrates that older key as it writes.

### Copilot for JetBrains

The plugin reads a single user-level file and offers no per-project
configuration, so this adapter is global-only. A project-scope request fails
with `no project-level configuration is available; pass --scope global`
instead of writing outside the project root. The `intellij` path segment is a
fixed literal in the plugin, so every JetBrains IDE shares one file.

### Codex

Codex uses TOML under an `mcp_servers` table. The managed entry sets
`default_tools_approval_mode = "approve"`, because Codex otherwise prompts for
approval on every MCP tool call.

### OpenCode

OpenCode probes several filenames and reads the first that exists, so the
adapter writes to whichever candidate the host would load:

```text
project: opencode.json, opencode.jsonc, .opencode/opencode.json, .opencode/opencode.jsonc
global:  $XDG_CONFIG_HOME/opencode/opencode.jsonc, opencode.json, config.json
```

The local server entry has a shape of its own: the command and its arguments
share one array, and a `type` field marks it as local.

```json
{
  "mcp": {
    "mytool": {
      "type": "local",
      "command": ["mytool-mcp"],
      "enabled": true
    }
  }
}
```

## Merge and safety rules

The JSON and TOML writers share one set of rules, built so an install never
damages a file the user also edits.

- **A valid container is required.** A file that is not valid JSON, JSONC, or
  TOML is preserved untouched and reported as a warning.
- **An existing entry is left alone.** Unless the caller passes `--force`, or
  the entry matches a known legacy template from an earlier version, the entry
  stays exactly as it was.
- **Emptying cleans up.** Removing the managed entry drops the server table
  when it empties, drops the configuration file when nothing remains, and
  removes a directory the tool emptied.
- **Writes are atomic.** Each write goes through a temporary file and a rename,
  so a process that dies mid-write cannot leave a half-written configuration.
- **A dry run changes nothing.** Adapters compute the same paths and warnings
  without touching the filesystem.

JSONC comments do not survive a rewrite. The reader normalises comments and
trailing commas, and the writer emits strict JSON, which is the trade that
keeps commented host files configurable. See [JSONC](glossary.md#jsonc).
