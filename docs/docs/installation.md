---
title: Installation
description: Add mcpinstall to a Go MCP project, define its Spec, and register the server with an agent host.
icon: lucide/package-plus
---

# Installation

`mcpinstall` is a library and a CLI that a host project wires once. The project
supplies a [Spec](embedding.md) and embeds its agent guidance; the module does
the rest.

## Add the module

```bash
go get github.com/phillarmonic/mcpinstall
```

While consuming a local checkout, point the module at it with a `replace`
directive:

```bash
go mod edit -replace github.com/phillarmonic/mcpinstall=../phillarmonic/mcpinstall
go mod tidy
```

## Define a Spec

A [Spec](embedding.md) names the server, the executable hosts launch, the
guidance marker, and the embedded guidance tree:

```go
func spec() mcpinstall.Spec {
	return mcpinstall.Spec{
		Name:     "mytool",
		Command:  "mytool-mcp",
		Program:  "mytool",
		Marker:   "mytool",
		Guidance: fs.FS(guidanceFS),
	}
}
```

An HTTP server sets `URL` instead of relying on `Command`:

```go
func spec() mcpinstall.Spec {
	return mcpinstall.Spec{
		Name:    "mytool",
		URL:     "https://mytool.example.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer " + token},
		Program: "mytool",
		Marker:  "mytool",
	}
}
```

A URL implies the HTTP transport. Set `Transport: "stdio"` explicitly when a
host should launch `Command` even though the Spec also carries a URL.

## Embed the guidance

Put one file per harness under a directory and embed the tree:

```go
//go:embed guidance/*
var guidanceFS embed.FS
```

The filename stem selects the target. `guidance/cursor.md` is the Cursor
guidance, `guidance/codex.md` the Codex guidance, and so on. An empty set is
allowed when only MCP registration is wanted. See [Guidance](guidance.md) for
the destination each stem maps to.

## Wire the command

```go
func main() {
	os.Exit(cli.Run(spec(), os.Args[1:], os.Stdout, os.Stderr))
}
```

`cli.Run` returns the process exit code: `0` for success, `1` when a run fails,
and `2` for a usage error such as an unknown command or a missing `--host` or
`--agent`.

## Register the server

Project scope is the default and writes only inside the current repository:

```bash
mytool install --host claude
```

Global scope writes the user-level configuration for the host:

```bash
mytool install --host all --scope global
```

Preview without writing, or inspect the result as JSON:

```bash
mytool install --host all --dry-run
mytool install --host codex --format json
```

An HTTP server installs the same way, with `--url` and `--header` overriding
the Spec for one run:

```bash
mytool install --host gemini --scope global \
  --url https://mytool.example.com/mcp \
  --header 'Authorization=Bearer token'
```

See the [CLI reference](cli.md) for every flag, and [Hosts](hosts.md) for the
file each host receives.

## Install skills and hooks

The same Spec can carry Agent Skills and lifecycle hooks:

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

```bash
mytool install-skills --agent both
mytool install-hooks --agent all --scope global
```

Skills are refused over an unmanaged same-named directory unless `--force` is
passed. Hooks replace only this tool's entries. See
[Skills and hooks](skills-hooks.md) for the roots, host matrix, and ownership
rules.
