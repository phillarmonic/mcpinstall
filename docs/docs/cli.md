---
title: CLI reference
description: Every command and flag mcpinstall exposes, plus its text and JSON output.
icon: lucide/terminal
---

# CLI reference

A host calls `cli.Run(spec, args, stdout, stderr)` and returns its exit code.
The command set is identical for every caller.

## Commands

| Command | Purpose |
| --- | --- |
| `install` | Register the server with a host. |
| `uninstall` | Remove the server from a host. |
| `uninstall-legacy` | Remove entries and guidance left by a former server name. |
| `install-guidance` | Install project agent guidance. |
| `uninstall-guidance` | Remove project agent guidance. |
| `global-guidance install` | Install user-level agent guidance. |
| `global-guidance uninstall` | Remove user-level agent guidance. |
| `install-skills` | Install the Spec's Agent Skills. |
| `uninstall-skills` | Remove the Spec's Agent Skills. |
| `install-hooks` | Install the Spec's lifecycle hooks. |
| `uninstall-hooks` | Remove the Spec's lifecycle hooks. |
| `hosts` | List the supported hosts and their capabilities. |
| `version` | Print the version. |

## install and uninstall

`install` and `uninstall` take the same flags and differ only in direction.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--host` | required | Comma-separated hosts, or `all`. |
| `--scope` | `project` | `project` or `global`. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--command` | the Spec command | Executable hosts launch for a stdio entry. |
| `--transport` | inferred | `stdio` or `http`; a URL implies `http`. |
| `--url` | the Spec URL | Remote MCP endpoint for an HTTP entry. |
| `--header` | the Spec headers | HTTP header as `name=value`; repeatable. |
| `--token-env-var` | the Spec value | Environment variable holding the bearer for hosts that support indirection. |
| `--force` | `false` | Overwrite an existing managed entry. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

```bash
mytool install --host claude
mytool install --host claude,cursor --scope global
mytool install --host all --command /opt/mytool/mytool-mcp
mytool install --host gemini --scope global --url https://mytool.example.com/mcp \
  --header 'Authorization=Bearer token'
mytool uninstall --host codex --dry-run
```

A host that has no configuration for the requested scope is skipped with a
warning, so `--host all --scope project` still configures the project hosts.
ZCode is HTTP-only: a stdio run skips it with a warning rather than failing the
run.

Text output names the scope and root, one line per host, and the paths it wrote
or removed:

```text
project scope: /work/project

claude: wrote .mcp.json
cursor: already current

1 configured, 1 already current
```

A dry run prefixes `dry run: no files were written` and uses `would write` and
`would remove`.

`--format json` emits the same run as an envelope:

```json
{
  "action": "install",
  "scope": "project",
  "root": "/work/project",
  "dry_run": false,
  "results": [
    {
      "host": "claude",
      "paths": [".mcp.json"]
    },
    {
      "host": "cursor"
    }
  ]
}
```

Each result carries `host`, an `error` for a host that failed without stopping
the others, the `paths` it changed, and any `warnings`.

## uninstall-legacy

Removes a former server id and the guidance that still names it. The current
entry and guidance are left in place. The command is a usage error when the
Spec declares no `LegacyNames`.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--host` | `all` | Comma-separated hosts, or `all`. |
| `--scope` | `both` | `project`, `global`, or `both`. |
| `--root` | cwd or home | Project directory, or the home override for global scope. |
| `--format` | `text` | `text` or `json`. |
| `--dry-run` | `false` | Report pending changes without writing. |

```bash
mytool uninstall-legacy
mytool uninstall-legacy --scope project --dry-run
```

See [Guidance](guidance.md#legacy-cleanup) for what the cleanup removes.

## install-guidance and uninstall-guidance

Write or remove project agent guidance without touching MCP configuration.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--root` | `.` | Project root directory. |
| `--harness` | `all` | A harness file stem, such as `cursor` or `codex`, or `all`. |
| `--force` | `false` | Overwrite existing guidance files. |
| `--dry-run` | `false` | Report pending changes without writing. |

```bash
mytool install-guidance
mytool install-guidance --harness cursor --force
mytool uninstall-guidance
```

## global-guidance

Writes user-level guidance under the home directory so a working tree does not
need project guidance files. It takes the same flags as `install-guidance`,
where `--root` names the directory to treat as home.

```bash
mytool global-guidance install
mytool global-guidance uninstall --harness codex
```

## install-skills and uninstall-skills

Write or remove the `SKILL.md` files declared by `Spec.Skills`. Each skill lands
under the host's skills root, marked with `<!-- <marker>-managed: skill -->`.
An unmanaged same-named skill is refused unless `--force` is passed.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--agent` | required | `claude`, `agents`, `devin`, `grok`, `copilot-cli`, `hermes`, `both`, or `all`. |
| `--scope` | `project` | `project` or `global`. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--force` | `false` | Replace a same-named skill that is not managed. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

```bash
mytool install-skills --agent both
mytool install-skills --agent all --scope global
mytool uninstall-skills --agent claude
```

See [Skills and hooks](skills-hooks.md#agent-skills) for the per-host roots.

## install-hooks and uninstall-hooks

Write or remove lifecycle hooks in the host's native configuration. Portable
events are mapped onto each host's vocabulary; a re-apply replaces only this
tool's entries and preserves other tools' hooks.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--agent` | required | Comma-separated hooks hosts, or `all`. |
| `--scope` | `global` | `global`, or `project` for Claude Code. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

```bash
mytool install-hooks --agent claude
mytool install-hooks --agent all --scope global
mytool uninstall-hooks --agent cursor
```

Only Claude Code has a project-level hook file. `--agent all --scope project`
configures Claude and skips the global-only hosts with a warning. See
[Skills and hooks](skills-hooks.md#lifecycle-hooks) for the host matrix and the
unsupported hosts.

## hosts

```bash
mytool hosts
```

Prints one line per MCP host with its format and server key, marks a host with
no writer as `guidance only`, and explains the `all` meta name. `all` expands
only to the supported hosts. Agent Skills and hooks use separate matrices; see
[Skills and hooks](skills-hooks.md).

## version

```bash
mytool version
```

Prints the `cli.Version` value, which a host sets from its own build.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The run succeeded, including a run that had nothing to change. |
| `1` | A run failed, such as a host that could not write its configuration. |
| `2` | A usage error: unknown command, invalid flag, or a missing `--host` or `--agent`. |
