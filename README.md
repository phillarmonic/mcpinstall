# mcpinstall

Register a Go MCP server with the configuration files of the agent hosts that
support one, and write the matching agent guidance.

`mcpinstall` implements the parts every MCP project otherwise rewrites: the
per-host config adapters, safe JSON/JSONC/TOML merging, managed guidance
sections, and the `install`/`uninstall`/`*-guidance` CLI. A caller supplies a
`Spec` naming the server and pointing at its embedded guidance.

## Supported hosts

| Host | Scope | Configuration file | Server key |
| --- | --- | --- | --- |
| Claude Code | project | `.mcp.json` | `mcpServers` |
| Claude Code | global | `~/.claude.json` | `mcpServers` |
| Cursor | project | `.cursor/mcp.json` | `mcpServers` |
| Cursor | global | `~/.cursor/mcp.json` | `mcpServers` |
| Copilot in VS Code | project | `.vscode/mcp.json` | `servers` |
| Copilot in VS Code | global | VS Code profile `mcp.json` | `servers` |
| Copilot in JetBrains | global | `~/.config/github-copilot/intellij/mcp.json` | `servers` |
| Codex (ChatGPT) | project | `.codex/config.toml` | `mcp_servers` |
| Codex (ChatGPT) | global | `~/.codex/config.toml` | `mcp_servers` |
| OpenCode | project | `opencode.json` | `mcp` |
| OpenCode | global | `~/.config/opencode/opencode.jsonc` | `mcp` |

## Usage

```go
package main

import (
	"embed"
	"io/fs"
	"os"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/cli"
)

//go:embed guidance/*
var guidanceFS embed.FS

func spec() mcpinstall.Spec {
	return mcpinstall.Spec{
		Name:     "mytool",
		Command:  "mytool-mcp",
		Program:  "mytool",
		Marker:   "mytool",
		Guidance: fs.FS(guidanceFS),
	}
}

func main() {
	os.Exit(cli.Run(spec(), os.Args[1:], os.Stdout, os.Stderr))
}
```

The guidance files live under `guidance/` and are named by harness stem
(`opencode.md`, `claude.json`, `cursor.md`, ...). Markdown files are written
verbatim; `.md` bodies that reach a Cursor user-level rule (`.mdc`) are wrapped
in `alwaysApply` frontmatter; JSON files carry a structured `mcp_tools.priority`
list. Empty harness sets are allowed when only MCP registration is wanted.

## Packages

- `mcpinstall`: the `Spec` type that identifies the server.
- `mcpinstall/platform`: `Scope`, `Target`, `Adapter`, and the `common` writers.
- `mcpinstall/platform/catalog`: host names, `--host all` expansion, statuses.
- `mcpinstall/platform/<host>`: one adapter per host.
- `mcpinstall/install`: `Run`, guidance application, legacy cleanup.
- `mcpinstall/cli`: the parameterized command-line surface.

## Development

This module is consumed locally with a `replace` directive until it is
published:

```
replace github.com/phillarmonic/mcpinstall => ../phillarmonic/mcpinstall
```
