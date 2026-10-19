---
title: Glossary
description: Terms used across the mcpinstall documentation.
icon: lucide/book-open
---

# Glossary

## MCP

The Model Context Protocol, a standard for connecting a model to external tools
and data. `mcpinstall` registers either a stdio server, which the host launches
as a child process and talks to over standard input and output, or an HTTP
server reached at a URL.

## MCP server entry

The record written into a host configuration: a command and its arguments for a
stdio server, or a URL and headers for an HTTP one. The shape varies by host.
Most JSON hosts use a command plus an arguments array, Codex adds a TOML
approval key, Gemini names the URL `httpUrl`, Antigravity names it `serverUrl`,
and OpenCode merges the command and arguments into one array.

## Transport

Whether a server is launched locally (`stdio`) or reached remotely (`http`). A
Spec that sets a URL infers `http`; otherwise it is `stdio`. Most hosts accept
both, ZCode accepts only HTTP, and Claude Desktop bridges HTTP through the
`mcp-remote` stdio shim.

## Host

A desktop or editor agent that reads a configuration file to discover MCP
servers. The catalog covers Claude Code, Cursor, Copilot in VS Code and
JetBrains, Codex, OpenCode V1/V2, Claude Desktop, Gemini, OpenClaw, OMP,
Antigravity, Zero, ZCode, Devin, Grok, Kimi, Kiro, Command Code, IBM Bob,
Copilot CLI, Droid, Swival, Zed, Muse, and Prime Agent.

## Adapter

The small package that knows where one host keeps its MCP configuration and how
to merge the server entry without disturbing the rest of the file. Every
adapter implements the `platform.Adapter` interface.

## Catalog

The single registry that maps a `--host` name onto an adapter, expands the
`all` meta name, and reports each adapter's capabilities.

## Capabilities

What an adapter can configure: its format (`json` or `toml`), its server table
key (`mcpServers`, `servers`, `mcp_servers`, `mcp`, `mcp.servers`, or
`context_servers`), and whether the host has a machine-readable MCP
configuration.

## Scope

Whether a configuration applies to one project or to every project. `project`
is the default; `global` targets user-level files.

## Project scope

Configuration that lives inside the current repository, such as `./.mcp.json`.
The root defaults to the working directory.

## Global scope

Configuration that applies to every project, resolved against the home
directory, such as `~/.claude.json`.

## Target

The base location an adapter resolves its configuration path against: the scope
plus the root directory.

## Spec

The struct a caller supplies to describe one server: its name, command or URL,
transport, headers, guidance marker, embedded guidance tree, any former names,
and the optional Agent Skills and lifecycle hooks. It is the only
project-specific input the module needs.

## Marker

The short prefix that namespaces the managed guidance section and the
user-level Cursor rule. A marker of `mytool` produces the
`<!-- mytool-guidance:begin -->` markers and `mytool.mdc`. It is also the
default ownership token for skills and hooks.

## Agent Skill

A directory containing a `SKILL.md` file that an agent loads on demand. The
installer writes `<skills-root>/<name>/SKILL.md` and embeds
`<!-- <marker>-managed: skill -->` after the frontmatter.

## Lifecycle hook

A host event that runs a command, such as session start or a tool call. A
`Hook` names a portable event and the command; each host descriptor maps the
event onto its own vocabulary and configuration shape.

## Portable event

The host-neutral name a caller uses for a lifecycle boundary, such as
`session-start` or `pre-tool-use`. A host with no equivalent skips it with a
warning.

## Hook marker

The ownership token for hook entries. Shell-form commands must contain it;
Zero, ZCode, and Kiro use it as the prefix of their generated id or name. It
defaults to the Spec [marker](#marker).

## Guidance

The agent instructions written alongside MCP registration, held in the embedded
tree a Spec points at and named one file per harness.

## Harness

An agent surface that reads instructions from a known location, such as Cursor
rules or `AGENTS.md`. The guidance file stem selects the harness.

## Managed section

A block of a shared file that the tool owns, delimited by
`<!-- <marker>-guidance:begin -->` and `<!-- <marker>-guidance:end -->`. An
install rewrites the block in place; an uninstall removes it and leaves the
rest of the file alone.

## Entry name

The server key an uninstall deletes. It defaults to the Spec name and can be
overridden so a run removes a specific key, which legacy cleanup uses.

## Legacy name

A former server key a project used before a rename. `uninstall-legacy` removes
these entries and the guidance that still names them, leaving the current
server in place.

## Dry run

A run that computes every pending change and reports the paths without writing
anything.

## Atomic write

A write that goes through a temporary file in the same directory and a rename,
so a process that dies mid-write cannot leave a partially written
configuration.

## JSONC

JSON with comments and trailing commas. Several hosts ship it; the readers
normalise it before parsing. Comments do not survive a rewrite, because the
writers emit strict JSON.
