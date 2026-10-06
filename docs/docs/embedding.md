---
title: Embedding
description: The Spec a caller supplies, the guidance tree, and the cli.Run entry point.
icon: lucide/puzzle
---

# Embedding

A host project depends on `mcpinstall` for two things: the `Spec` that
describes its server, and `cli.Run`, which turns that spec into the command
set. Nothing host-specific leaks into the project, and nothing project-specific
leaks into the module.

## The Spec

```go
type Spec struct {
	Name                  string
	Command               string
	Args                  []string
	Program               string
	Marker                string
	CursorRuleDescription string
	LegacyNames           []string
	Guidance              fs.FS
	GuidanceDir           string
}
```

| Field | Required | Purpose |
| --- | --- | --- |
| `Name` | yes | The server key written into host configuration and removed by uninstall. |
| `Command` | yes | The executable hosts launch. |
| `Args` | no | Fixed arguments stored beside the command. |
| `Program` | no | The CLI name shown in usage text; defaults to `mcpinstall`. |
| `Marker` | yes | Prefixes the managed guidance markers and names the Cursor rule. |
| `CursorRuleDescription` | no | `alwaysApply` description for the user-level Cursor rule. |
| `LegacyNames` | no | Former server keys that `uninstall-legacy` removes. |
| `Guidance` | no | The embedded guidance tree. |
| `GuidanceDir` | no | The directory inside `Guidance`; defaults to `guidance`. |

`Name` and `Marker` are described as required because the CLI needs a server
key and a guidance marker. `Args` is usually empty, with `Command` alone as the
common case.

The marker drives both the HTML comment boundaries and the Cursor rule:

```text
<!-- <marker>-guidance:begin --> ... <!-- <marker>-guidance:end -->
~/.cursor/rules/<marker>.mdc
```

## The entry point

```go
func Run(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer) int
```

`Run` parses the first argument as a subcommand, dispatches it against the
spec, writes output to `stdout` and diagnostics to `stderr`, and returns the
exit code. It never calls `os.Exit`, which keeps it testable, and it takes the
writers as arguments so a host can capture or redirect them.

## A complete host

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

Set `cli.Version` from the project's own build so `version` reports the host's
version rather than the module default of `dev`.

## Calling the packages directly

`cli.Run` is the usual entry point, but the layers below it are exported for a
host that needs a programmatic path. `install.Run` takes a request and returns
one result per host. `install.ApplyGuidance` writes guidance without touching
MCP configuration. `catalog.Adapters` and `catalog.Statuses` expose the host
registry. See [Architecture](architecture.md) for the layers.
