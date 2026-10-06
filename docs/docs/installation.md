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
and `2` for a usage error such as an unknown command or a missing `--host`.

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

See the [CLI reference](cli.md) for every flag, and [Hosts](hosts.md) for the
file each host receives.
