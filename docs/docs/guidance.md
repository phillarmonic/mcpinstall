---
title: Guidance
description: How mcpinstall writes agent guidance, maps harness files to destinations, and manages shared files.
icon: lucide/notebook-pen
---

# Guidance

MCP registration tells a host how to launch a server. Guidance tells the model
when to prefer that server's tools. `mcpinstall` writes both from one
[Spec](embedding.md): the configuration adapters handle registration, and the
guidance tree handles the instructions.

## Harness files

The guidance tree holds one file per [harness](glossary.md#harness), named by
its stem. `guidance/cursor.md` targets Cursor, `guidance/codex.md` targets
Codex, and so on. The extension decides how the body is written, so a project
can embed Markdown, a `.mdc` Cursor rule, or JSON.

## Destinations

| Harness file | Project scope | Global scope | Written as |
| --- | --- | --- | --- |
| `claude.md` | `.claude/instructions.md` | `~/.claude/instructions.md` | whole file |
| `cursor.md` | `.cursor/instructions.md` | `~/.cursor/rules/<marker>.mdc` | whole file |
| `copilot.md` | `.copilot/instructions.md` | `~/.copilot/instructions.md` | whole file |
| `codex.md` | `AGENTS.md` | `~/.codex/AGENTS.md` | managed section |
| `opencode.md` | `AGENTS.md` | `~/.config/opencode/AGENTS.md` | managed section |

A harness the module does not special-case falls back to
`.<harness>/instructions.md`. The user-level Cursor rule gets `alwaysApply`
frontmatter whose description comes from `Spec.CursorRuleDescription`, or a
generic default built from `Spec.Program`.

## Whole-file and managed guidance

A whole-file destination is owned by the tool. The file is written verbatim on
install and removed on uninstall. An existing file is not overwritten unless
`--force` is passed, so a user's own `instructions.md` is left alone.

Codex and OpenCode read the shared `AGENTS.md`. The module writes only a
delimited [managed section](glossary.md#managed-section) and leaves the rest of
the file to the user:

```markdown
<!-- mytool-guidance:begin -->
...guidance body...
<!-- mytool-guidance:end -->
```

An install upserts the section in place and reports no change when the body
already matches. An uninstall removes the section and deletes the file only
when nothing else remains. Because Codex and OpenCode share `AGENTS.md`, they
share one managed section rather than competing for the file.

## Installing and removing

Project guidance comes from `install-guidance` and `uninstall-guidance`.
User-level guidance comes from `global-guidance install` and
`global-guidance uninstall`. Both accept `--root`, `--harness`, `--force`, and
`--dry-run`. The harness value is a file stem, so any stem in the guidance tree
can be selected. See the
[CLI reference](cli.md#install-guidance-and-uninstall-guidance).

Registration and guidance are separate commands. `install` writes MCP
configuration only; the guidance commands read the embedded tree named by
`Spec.Guidance`, so a host can register the server and apply the instructions
in whichever order suits it.

## Legacy cleanup

`uninstall-legacy` removes what a former server name left behind, without
touching the current server. For every name in `Spec.LegacyNames` it removes
the server entry from each host configuration, deletes the Cursor rule
`<name>.mdc`, strips the `<name>-guidance` managed section from `AGENTS.md`,
and deletes other guidance files whose body still contains the legacy name. The
scope defaults to `both`, so a single run cleans project and user-level files.
