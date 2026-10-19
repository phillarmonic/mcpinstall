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
	Transport             string
	URL                   string
	Headers               map[string]string
	TokenEnvVar           string
	Program               string
	Marker                string
	CursorRuleDescription string
	LegacyNames           []string
	Guidance              fs.FS
	GuidanceDir           string
	Skills                []Skill
	SkillMarker           string
	Hooks                 []Hook
	HookMarker            string
}
```

| Field | Required | Purpose |
| --- | --- | --- |
| `Name` | yes | The server key written into host configuration and removed by uninstall. |
| `Command` | stdio | The executable hosts launch. |
| `Args` | no | Fixed arguments stored beside the command. |
| `Transport` | no | `stdio` or `http`; defaults to `http` when `URL` is set, else `stdio`. |
| `URL` | http | Remote MCP endpoint. |
| `Headers` | no | HTTP headers, typically `Authorization`. |
| `TokenEnvVar` | no | Environment variable name for hosts that support bearer indirection. |
| `Program` | no | The CLI name shown in usage text; defaults to `mcpinstall`. |
| `Marker` | yes | Prefixes the managed guidance markers and names the Cursor rule. |
| `CursorRuleDescription` | no | `alwaysApply` description for the user-level Cursor rule. |
| `LegacyNames` | no | Former server keys that `uninstall-legacy` removes. |
| `Guidance` | no | The embedded guidance tree. |
| `GuidanceDir` | no | The directory inside `Guidance`; defaults to `guidance`. |
| `Skills` | no | Agent Skills written by `install-skills`. |
| `SkillMarker` | no | Ownership token for managed `SKILL.md` files; defaults to `Marker`. |
| `Hooks` | no | Lifecycle hooks written by `install-hooks`. |
| `HookMarker` | no | Ownership token for managed hooks; defaults to `Marker`. |

`Name` and `Marker` are described as required because the CLI needs a server
key and a guidance marker. `Args` is usually empty, with `Command` alone as the
common case. An HTTP Spec supplies `URL` instead of `Command`; setting both is
allowed, because a host that only accepts stdio (Claude Desktop) bridges the
HTTP endpoint through `mcp-remote` rather than launching `Command`.

The marker drives both the HTML comment boundaries and the Cursor rule:

```text
<!-- <marker>-guidance:begin --> ... <!-- <marker>-guidance:end -->
~/.cursor/rules/<marker>.mdc
```

It is also the default ownership token for skills and hooks. A skill file
carries `<!-- <marker>-managed: skill -->`; every shell-form hook command must
contain the marker. Zero, ZCode, and Kiro use it as the prefix of their
generated id or name. See [Skills and hooks](skills-hooks.md) for the details.

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

An HTTP host is the same shape with a URL:

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

Set `cli.Version` from the project's own build so `version` reports the host's
version rather than the module default of `dev`.

Skills and hooks are additional Spec fields, not a separate entry point:

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

## Calling the packages directly

`cli.Run` is the usual entry point, but the layers below it are exported for a
host that needs a programmatic path. `install.Run` takes a request and returns
one result per host. `install.RunSkills` and `install.RunHooks` do the same for
the skills and hooks surfaces. `install.ApplyGuidance` writes guidance without
touching MCP configuration. `catalog.Adapters` and `catalog.Statuses` expose the
MCP host registry; `skills.Adapters` and `hooks.Adapters` expose the extension
hosts. See [Architecture](architecture.md) for the layers.
