# mcpinstall

Register a stdio or HTTP MCP server with the configuration files of the agent
hosts that support one, install Agent Skills and lifecycle hooks, and write the
matching agent guidance.

`mcpinstall` implements the parts every MCP project otherwise rewrites: the
per-host config adapters, safe JSON/JSONC/TOML merging, managed guidance
sections, the skills and hooks installers, and the CLI. A caller supplies a
`Spec` naming the server and pointing at its embedded guidance.

## Supported hosts

The catalog covers the host matrix ported from
[ai-memory](https://github.com/akitaonrails/ai-memory), including Claude Code,
Cursor, Copilot in VS Code and JetBrains, Codex, OpenCode V1/V2, Claude Desktop,
Gemini CLI, OpenClaw, OMP, Antigravity, Zero, ZCode, Devin, Grok, Kimi, Kiro,
Command Code, IBM Bob, Copilot CLI, Droid, Swival, Zed, Muse, and Prime Agent.

- **Project and global:** Claude Code, Cursor, Copilot in VS Code, Codex,
  OpenCode V1, and OpenCode V2.
- **Global only:** Claude Desktop, Gemini, OpenClaw, OMP, Antigravity, Zero,
  ZCode, Devin, Grok, Kimi, Kiro, Command Code, IBM Bob, Copilot CLI, Droid,
  Zed, Muse, Prime Agent, and Copilot for JetBrains.
- **Project only:** Swival.
- **Guidance only:** DeepSeek Harness and Pi, which have no fixed MCP file.

Transport support varies by host: most accept both a stdio command and an HTTP
URL, while ZCode is HTTP-only and Claude Desktop bridges HTTP through
`mcp-remote`. See [Hosts](docs/docs/hosts.md) for every path, server key, entry
shape, scope rule, and caveat.

Agent Skills install into the Claude Code, shared `.agents`, Devin, Grok,
Copilot CLI, and Hermes skill roots. Lifecycle hooks cover the hosts with a
native hook configuration: Claude Code, Codex, Cursor, Gemini, Antigravity,
Grok, Devin, Command Code, Copilot CLI, Droid, Muse, CodeBuddy, IBM Bob, Zero,
ZCode, Kimi, and Kiro. See
[Skills and hooks](docs/docs/skills-hooks.md) for the roots, config shapes, and
portable event matrix.

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

A spec whose tools work better on the provider's native tool list sets
`ToolMode: "native"`. OpenCode V2 and Copilot CLI distinguish the two surfaces
today: OpenCode writes `"codemode": false` instead of grouping the tools behind
its Code Mode executor, and Copilot CLI writes `"deferTools": "never"` so tool
search does not defer them. Other hosts ignore the setting, and
`--tool-mode native` overrides it per run.

An HTTP server sets `URL` instead of relying on `Command`. A Spec with a URL
infers the HTTP transport:

```go
return mcpinstall.Spec{
	Name:    "mytool",
	URL:     "https://mytool.example.com/mcp",
	Headers: map[string]string{"Authorization": "Bearer " + token},
	Program: "mytool",
	Marker:  "mytool",
}
```

The guidance files live under `guidance/` and are named by harness stem
(`opencode.md`, `claude.json`, `cursor.md`, ...). Markdown files are written
verbatim; `.md` bodies that reach a Cursor user-level rule (`.mdc`) are wrapped
in `alwaysApply` frontmatter; JSON files carry a structured `mcp_tools.priority`
list. Empty harness sets are allowed when only MCP registration is wanted.

Skills and hooks are declared on the same Spec:

```go
spec := mcpinstall.Spec{
    Name:    "mytool",
    Command: "mytool-mcp",
    Program: "mytool",
    Marker:  "mytool",
    Skills: []mcpinstall.Skill{{
        Name:    "release-checklist",
        Content: "# Release checklist\n\nRun the tests.\n",
    }},
    Hooks: []mcpinstall.Hook{{
        Event:   mcpinstall.HookSessionStart,
        Command: "mytool hook --event session-start",
    }},
}
```

`install-skills` writes a marked `SKILL.md` per host root. `install-hooks`
maps the portable event onto each host's vocabulary and preserves other tools'
entries. Both remove only what they own.

```bash
mytool install-skills --agent both
mytool install-hooks --agent all --scope global
mytool uninstall-hooks --agent claude
```

## Packages

- `mcpinstall`: the `Spec` type that identifies the server.
- `mcpinstall/platform`: `Scope`, `Target`, `Adapter`, `ScopeSupport`, and the `common` writers.
- `mcpinstall/platform/catalog`: host names, `--host all` expansion, statuses.
- `mcpinstall/platform/<host>`: the original per-host adapters.
- `mcpinstall/platform/hosts`: the table-driven host matrix, one descriptor per additional host.
- `mcpinstall/platform/skills`: the Agent Skills roots and writer.
- `mcpinstall/platform/hooks`: the lifecycle hook descriptors and config-shape writers.
- `mcpinstall/install`: `Run`, `RunSkills`, `RunHooks`, guidance application, legacy cleanup.
- `mcpinstall/cli`: the parameterized command-line surface.

## Development

This module is consumed locally with a `replace` directive until it is
published:

```
replace github.com/phillarmonic/mcpinstall => ../phillarmonic/mcpinstall
```
