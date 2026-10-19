---
title: Hosts
description: The agent hosts mcpinstall configures, their files, keys, scope rules, and merge behavior.
icon: lucide/server
---

# Hosts

Each host has one adapter. The original six are small packages under
`platform/<host>`; the rest are descriptors in `platform/hosts`, where the path,
server table, and entry shape are data rather than hand-written code. The
[catalog](architecture.md#the-catalog) resolves a `--host` value onto the right
adapter.

This page covers MCP registration. Agent Skills and lifecycle hooks use their
own host matrices and configuration shapes; see
[Skills and hooks](skills-hooks.md).

## Catalog

| Catalog name | Host | Scopes | Configuration file | Server key | Format |
| --- | --- | --- | --- | --- | --- |
| `claude` | Claude Code | project, global | `.mcp.json` (project), `~/.claude.json` or `$CLAUDE_CONFIG_DIR/.claude.json` (global) | `mcpServers` | JSON |
| `cursor` | Cursor | project, global | `.cursor/mcp.json` | `mcpServers` | JSON |
| `copilot` | Copilot in VS Code | project, global | `.vscode/mcp.json` (project), VS Code profile `mcp.json` (global) | `servers` | JSON |
| `copilot-jetbrains` | Copilot for JetBrains | global | `~/.config/github-copilot/intellij/mcp.json` | `servers` | JSON |
| `codex` | Codex (ChatGPT) | project, global | `.codex/config.toml` or `$CODEX_HOME/config.toml` | `mcp_servers` | TOML |
| `opencode` | OpenCode V1 | project, global | `opencode.json`, `opencode.jsonc`, or `.opencode/opencode.json(c)` | `mcp` | JSON |
| `opencode2` | OpenCode V2 | project, global | same file as OpenCode V1 | `mcp.servers` | JSON |
| `claude-desktop` | Claude Desktop | global | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS), `%APPDATA%\Claude\...` (Windows), `$XDG_CONFIG_HOME/Claude/...` (Linux) | `mcpServers` | JSON |
| `gemini` | Gemini CLI | global | `~/.gemini/settings.json` | `mcpServers` | JSON |
| `openclaw` | OpenClaw | global | `~/.openclaw/config.json` | `mcp.servers` | JSON |
| `omp` | Oh My Pi / OMP | global | `~/.omp/agent/mcp.json`, a profile's `agent/mcp.json`, or `$PI_CODING_AGENT_DIR/mcp.json` | `mcpServers` | JSON |
| `antigravity` | Antigravity CLI | global | `~/.gemini/config/mcp_config.json` | `mcpServers` | JSON |
| `zero` | Zero | global | `$XDG_CONFIG_HOME/zero/config.json` (default `~/.config/zero/config.json`) | `mcp.servers` | JSON |
| `zcode` | ZCode | global | `~/.zcode/cli/config.json` | `mcp.servers` | JSON |
| `devin` | Devin CLI | global | `~/.devin/config.json` | `mcpServers` | JSON |
| `grok` | Grok Build CLI | global | `$GROK_HOME/config.toml` (default `~/.grok/config.toml`) | `mcp_servers` | TOML |
| `kimi` | Kimi Code | global | `$KIMI_CODE_HOME/mcp.json` (default `~/.kimi-code/mcp.json`) | `mcpServers` | JSON |
| `kiro` | Kiro CLI | global | `$KIRO_HOME/settings/mcp.json` (default `~/.kiro/settings/mcp.json`) | `mcpServers` | JSON |
| `command-code` | Command Code | global | `~/.commandcode/mcp.json` | `mcpServers` | JSON |
| `ibm-bob` | IBM Bob | global | `~/.bob/settings/mcp.json` | `mcpServers` | JSON |
| `copilot-cli` | GitHub Copilot CLI | global | `$COPILOT_HOME/mcp-config.json` (default `~/.copilot/mcp-config.json`) | `mcpServers` | JSON |
| `droid` | Factory Droid | global | `~/.factory/mcp.json` | `mcpServers` | JSON |
| `swival` | Swival CLI | project | `.swival/mcp.json` at the nearest project root | `mcpServers` | JSON |
| `zed` | Zed | global | `~/.config/zed/settings.json` (macOS and Linux), `%APPDATA%\Zed\settings.json` (Windows) | `context_servers` | JSON |
| `muse` | Meta Muse Code | global | `$XDG_CONFIG_HOME/muse/settings.json` (default `~/.config/muse/settings.json`) | `mcpServers` or `mcp_servers` | JSON |
| `prime-agent` | Prime Agent | global | `$PRIME_AGENT_CODING_AGENT_DIR/settings.json` (default `~/.prime/agent/settings.json`) | `mcpServers` | JSON |

`--host all` expands to every name above. Two additional names are selectable
but not swept into `all`:

| Catalog name | Host | Why it has no writer |
| --- | --- | --- |
| `dsh` | DeepSeek Harness | No fixed MCP file; the entry is a Cordis patch fragment. |
| `pi` | Pi | No native `mcp.json`; Pi uses a generated bridge extension. |

Selecting either explicitly returns that guidance instead of writing a file.

Values are case-insensitive, spaces and underscores fold to hyphens, and the
catalog accepts the aliases each host is commonly known by (`claude-code`,
`vscode-copilot`, `opencode-v2`, `gemini-cli`, `agy`, `kimi-code`, `kiro-cli`,
`cmdc`, `bob`, `factory-droid`, `oh-my-pi`, `muse-code`, `primeagent`,
`z-code`, `open-claw`, `deepseek-harness`).

ai-memory documents hand-written recipes for Warp, Cline, Kilo Code, Goose,
Qwen Code, and OpenHuman but ships no `install-mcp` writer for them, so they are
not in this catalog. Crush, Pool, CodeBuddy, Hermes, and GrizzyBot are hook or
skill integrations rather than MCP config writers and are also out of scope.

## Scope

A [scope](glossary.md#scope) decides the root the adapter resolves its path
against:

| Scope | Root | Example result |
| --- | --- | --- |
| `project` (default) | the current directory, or `--root` | `./.mcp.json` |
| `global` | the home directory, or `--root` | `~/.claude.json` |

`--root` overrides both. A host that has only a user-level file is skipped with
a warning when a project-scope run includes it, rather than failing the other
hosts. A project-only host (`swival`) is skipped the same way in a global run.

A host's own relocation variable (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`,
`GROK_HOME`, `KIMI_CODE_HOME`, `KIRO_HOME`, `COPILOT_HOME`,
`PRIME_AGENT_CODING_AGENT_DIR`, `PI_CODING_AGENT_DIR`, `PI_CONFIG_DIR`,
`OMP_PROFILE` with its `PI_PROFILE` fallback, or `XDG_CONFIG_HOME`)
takes precedence over `--root`, because that is the path the host itself reads.
Clear the variable when `--root` must be authoritative.

## Transport

A `Spec` can describe a stdio server or an HTTP one. Most hosts accept both;
the adapter translates the shared entry into the shape that host documents.

| Host | stdio | HTTP |
| --- | --- | --- |
| Claude Code | `command`/`args` | `type: http`, `url`, `headers` |
| Cursor | `command`/`args` | `url`, `headers` |
| Copilot in VS Code | `type: stdio`, `command`, `args` | `type: http`, `url`, `headers` |
| Copilot for JetBrains | `command`/`args` | `type: http`, `url`, `headers` |
| Codex | `command`/`args` | `url`, `http_headers` or `bearer_token_env_var` |
| OpenCode V1/V2 | `type: local`, one `command` array | `type: remote`, `url`, `headers` |
| Gemini CLI | `command`/`args`, `timeout` | `httpUrl`, `timeout`, `headers` |
| Antigravity CLI | `command`/`args`, `timeout` | `serverUrl`, `timeout`, `headers` |
| OpenClaw | `command`/`args` | `url`, `transport: streamable-http` |
| OMP | `type: stdio`, `enabled` | `type: http`, `enabled` |
| Zero | `type: stdio` | `type: http`, `url`, `headers` |
| Devin | `command`/`args` | `url`, `transport: http` |
| Grok | `command`/`args`, `enabled` | `url`, `enabled`, `headers` |
| Kimi Code | `command`/`args` | `url`, `headers` |
| Kiro CLI | `command`/`args` | `url`, `headers` (HTTPS or loopback HTTP) |
| Command Code | `transport: stdio`, `enabled` | `transport: http`, `enabled` |
| IBM Bob | `command`/`args` | `type: streamable-http`, `protocolNegotiation: legacy` |
| Copilot CLI | `type: local`, `tools: ["*"]` | `type: http`, `tools: ["*"]` |
| Factory Droid | `command`/`args` | `type: http`, `url`, `headers` |
| Swival | `type: stdio` | `type: http`, `url`, `headers` |
| Zed | `command`/`args` | `url`, `headers` |
| Muse | `type: stdio`, `mode: optional` | `type: http`, `mode: optional` |
| Prime Agent | `command`/`args` | `type: http`, `bearerTokenEnvVar` |
| ZCode | not supported; skipped with a warning | `type: http`, `url`, `headers` |

Claude Desktop's own file starts only stdio servers, so an HTTP Spec is bridged
through `npx -y mcp-remote <url>` and each header becomes an `mcp-remote
--header` argument.

## Per-host details

### Claude Code

Project scope uses the dedicated `.mcp.json`. Global scope keeps the server
alongside the rest of Claude Code's user state in `~/.claude.json`, or under
`$CLAUDE_CONFIG_DIR` when that variable relocates the config home.

### Cursor

Both scopes use `.cursor/mcp.json`; the root decides whether that is the
repository or the home directory.

### Copilot in VS Code

Project scope writes the workspace `.vscode/mcp.json`. Global scope writes the
per-profile file, which lives outside the home directory on macOS and Windows.

VS Code names its server table `servers` and an earlier spelling used
`mcpServers`; the writer migrates that older key as it writes. Both forms carry
an explicit `type`.

### Copilot for JetBrains

The plugin reads a single user-level file and offers no per-project
configuration, so this adapter is global-only and a project-scope run skips it
with a warning. The `intellij` path segment is a fixed literal in the plugin, so
every JetBrains IDE shares one file.

### Codex

Codex uses TOML under an `mcp_servers` table. The managed entry sets
`default_tools_approval_mode = "approve"`, because Codex otherwise prompts for
approval on every MCP tool call. An HTTP entry uses `http_headers` for a literal
header, or `bearer_token_env_var` when the caller names a variable through
`--token-env-var`.

### OpenCode

OpenCode probes several filenames and reads the first that exists, so the
adapter writes to whichever candidate the host would load:

```text
project: opencode.json, opencode.jsonc, .opencode/opencode.json, .opencode/opencode.jsonc
global:  $XDG_CONFIG_HOME/opencode/opencode.jsonc, opencode.json, config.json
```

V1 keeps the local server entry directly under `mcp`; V2 nests it under
`mcp.servers`, has no `enabled` field, and sets `oauth: false` on a remote entry
so the beta does not probe for OAuth metadata.

V2 routes MCP tools through Code Mode by default, grouping them behind the
`execute` tool. `--tool-mode native` (or `ToolMode: "native"` on the Spec) adds
`"codemode": false` so each tool stays on the provider's native tool list.
V1 has no equivalent field and ignores the request.

### Gemini CLI and Antigravity CLI

Both use a 5000 ms `timeout`. Gemini names the remote URL `httpUrl`;
Antigravity names it `serverUrl`. Neither accepts the other's spelling.

### OpenClaw, Zero, ZCode, and OpenCode V2

These four nest their servers under `mcp.servers`. ZCode is additionally
HTTP-only: its schema keeps only `type`, `url`, `headers`, `enabled`, and
`timeoutMs`, and silently drops an entry with `command` or `args`.

### Grok Build CLI and Codex

Both are TOML. Grok puts static headers under `[mcp_servers.<name>.headers]`;
Codex uses `[mcp_servers.<name>.http_headers]`. The key names are not
interchangeable.

### Kimi Code and Kiro CLI

Kimi Code treats an entry with `url` and no `transport` as streamable HTTP. Kiro
does the same. Kiro requires HTTPS for a remote server and accepts plain HTTP
only on loopback. Some upstreams need a schema-flavor query marker (`flavor=moonshot`
for Kimi, `flavor=bedrock` for Kiro); pass the full URL, marker included, when
your server needs one.

### IBM Bob

Bob's automatic negotiation probes `server/discover` before the standard MCP
initialize handshake, so the entry pins `protocolNegotiation` to `legacy`.

### Copilot CLI and Factory Droid

Copilot CLI's local and remote entries both carry `tools: ["*"]`. With
`--tool-mode native` they also carry `deferTools: "never"`, which keeps the
server's tools always loaded instead of deferred to tool search. Droid expands
`${VAR}` only inside headers, so a bearer supplied through `--token-env-var` is
written as a variable reference rather than copied into the file.

### Prime Agent

Prime Agent lists a user server only when it has an OAuth grant or a
`bearerTokenEnvVar` whose variable is set. The adapter writes the camelCase key
Prime Agent's loader reads.

### Muse

Muse drops its whole MCP block when both `mcpServers` and `mcp_servers` are
present, so the writer picks the map the file already uses and refuses a file
that holds both. A legacy `mcp_servers` entry renames `type` to `transport`, and
the file always carries `schema_version: 1`; Muse fails every command at startup
without it.

### Swival

Swival is project-scoped by design. The adapter walks up from the target root to
the nearest directory holding `.git` or `swival.toml`, then writes
`.swival/mcp.json`.

### Zed

Zed reads remote MCP servers from the top-level `context_servers` map in its
user `settings.json`.

### Claude Desktop

Claude Desktop's JSON config does not register HTTP transports directly. An
HTTP Spec is bridged through the community `mcp-remote` stdio shim, which needs
Node.js. The Windows MSIX package path is not auto-detected; point `--root` at
the active profile's `LocalCache\Roaming\Claude` directory when Claude Desktop
is installed as an MSIX package.

## Merge and safety rules

The JSON and TOML writers share one set of rules, built so an install never
damages a file the user also edits.

- **A valid container is required.** A file that is not valid JSON, JSONC, or
  TOML is preserved untouched and reported as a warning.
- **An existing entry is left alone.** Unless the caller passes `--force`, or
  the entry matches a known legacy template from an earlier version, the entry
  stays exactly as it was.
- **Emptying cleans up.** Removing the managed entry drops the server table
  when it empties, prunes a nested map, and drops the configuration file when
  nothing remains. Directories are left in place.
- **Writes are atomic.** Each write goes through a temporary file and a rename,
  so a process that dies mid-write cannot leave a half-written configuration.
- **A dry run changes nothing.** Adapters compute the same paths and warnings
  without touching the filesystem.

JSONC comments do not survive a rewrite. The reader normalises comments and
trailing commas, and the writer emits strict JSON, which is the trade that
keeps commented host files configurable. See [JSONC](glossary.md#jsonc).
