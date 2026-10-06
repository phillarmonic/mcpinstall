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
| `hosts` | List the supported hosts and their capabilities. |
| `version` | Print the version. |

## install and uninstall

`install` and `uninstall` take the same flags and differ only in direction.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--host` | required | Comma-separated hosts, or `all`. |
| `--scope` | `project` | `project` or `global`. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--command` | the Spec command | Executable hosts launch. |
| `--force` | `false` | Overwrite an existing managed entry. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

```bash
mytool install --host claude
mytool install --host claude,cursor --scope global
mytool install --host all --command /opt/mytool/mytool-mcp
mytool uninstall --host codex --dry-run
```

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

## hosts

```bash
mytool hosts
```

Prints one line per host with its format, server key, and the `all` meta name.

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
| `2` | A usage error: unknown command, invalid flag, or a missing `--host`. |
