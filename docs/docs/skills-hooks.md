---
title: Skills and hooks
description: Install Agent Skills and lifecycle hooks beside the MCP server, with the same host matrix and ownership rules.
icon: lucide/webhook
---

# Skills and hooks

`install` registers the MCP server. Two optional Spec surfaces cover the rest of
a host integration:

- `Skills` installs Agent Skills (`SKILL.md` directories) into the roots each
  agent reads.
- `Hooks` installs lifecycle hooks into the host's native hook configuration.

Both are driven by the same `Spec` and the same `cli.Run` entry point.

## Agent Skills

```go
type Skill struct {
    Name    string
    Content string
}

type Spec struct {
    // ...
    Skills      []Skill
    SkillMarker string // defaults to Marker
}
```

`Name` is one path segment and becomes the directory under the host's skills
root. `Content` is the complete `SKILL.md`, including YAML frontmatter:

```go
Skills: []mcpinstall.Skill{{
    Name: "release-checklist",
    Content: `---
name: release-checklist
description: Validate a release before publishing.
---

# Release checklist

1. Run the tests.
2. Build the artifacts.
`,
}},
```

The installer writes `<skills-root>/<Name>/SKILL.md`. It inserts an ownership
marker after the frontmatter:

```html
<!-- <SkillMarker>-managed: skill -->
```

A same-named skill that does not carry the marker is refused unless `--force`
is passed. Uninstall removes only marked files, then prunes empty skill and
skills directories; any supporting files the skill shipped are left in place.

### Skill roots

| Agent | Project | Global |
| --- | --- | --- |
| `claude` | `.claude/skills` | `$CLAUDE_CONFIG_DIR/skills`, else `~/.claude/skills` |
| `agents` | `.agents/skills` | `~/.agents/skills` |
| `devin` | `.devin/skills` | `%APPDATA%/devin/skills` on Windows, else `~/.devin/skills` |
| `grok` | `.grok/skills` | `$GROK_HOME/skills`, else `~/.grok/skills` |
| `copilot-cli` | `.github/skills` | `$COPILOT_HOME/skills`, else `~/.copilot/skills` |
| `hermes` | `.hermes/skills` | `~/.hermes/skills` |

`both` selects `claude` and `agents`. `all` selects every supported skills
host. Aliases such as `claude-code`, `github-copilot`, `grok-build`, and
`devin-cli` resolve to the canonical names.

```bash
mytool install-skills --agent both
mytool install-skills --agent all --scope global
mytool install-skills --agent claude --root /work/project --dry-run
mytool uninstall-skills --agent both
mytool install-skills --agent claude --format json
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--agent` | required | Comma-separated skills hosts, `both`, or `all`. |
| `--scope` | `project` | `project` or `global`. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--force` | `false` | Replace a same-named skill that is not managed. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

## Lifecycle hooks

```go
type HookEvent string

type Hook struct {
    Event          HookEvent
    Matcher        string
    Command        string
    Args           []string
    ID             string
    TimeoutSeconds int
}

type Spec struct {
    // ...
    Hooks      []Hook
    HookMarker string // defaults to Marker
}
```

The portable events are:

| Constant | Value |
| --- | --- |
| `HookSessionStart` | `session-start` |
| `HookUserPromptSubmit` | `user-prompt` |
| `HookPreToolUse` | `pre-tool-use` |
| `HookPostToolUse` | `post-tool-use` |
| `HookPostToolUseFailure` | `post-tool-use-failure` |
| `HookPreCompact` | `pre-compact` |
| `HookPostCompact` | `post-compact` |
| `HookStop` | `stop` |
| `HookSessionEnd` | `session-end` |
| `HookSubagentStart` | `subagent-start` |
| `HookSubagentStop` | `subagent-stop` |

Each host maps those onto its own vocabulary. A hook for an event a host does
not expose is skipped with a warning rather than guessed at.

```go
Hooks: []mcpinstall.Hook{
    {
        Event:   mcpinstall.HookSessionStart,
        Command: "mytool hook --event session-start",
    },
    {
        Event:   mcpinstall.HookPreToolUse,
        Matcher: "Bash",
        Command: "mytool hook --event pre-tool-use",
    },
},
```

### Host matrix

| Agent | Scopes | Configuration | Shape |
| --- | --- | --- | --- |
| `claude` | project, global | `.claude/settings.local.json` at the repo root; `$CLAUDE_CONFIG_DIR/settings.json` or `~/.claude/settings.json` | nested `hooks` map |
| `codex` | global | `$CODEX_HOME/hooks.json` or `~/.codex/hooks.json` | nested `hooks` map |
| `cursor` | global | `~/.cursor/hooks.json` | flat `hooks` map with `version: 1` |
| `gemini` | global | `~/.gemini/settings.json` | nested `hooks` map |
| `antigravity` | global | `~/.gemini/config/hooks.json` | named group; nested tool events, flat lifecycle events |
| `grok` | global | `$GROK_HOME/hooks/<marker>.json` or `~/.grok/hooks/<marker>.json` | nested `hooks` map |
| `devin` | global | `~/.devin/hooks.v1.json` | event map at the root |
| `command-code` | global | `~/.commandcode/settings.json` | nested `hooks` map without matchers |
| `copilot-cli` | global | `$COPILOT_HOME/hooks/<marker>.json` or `~/.copilot/hooks/<marker>.json` | flat `hooks` map with `version: 1` |
| `droid` | global | `~/.factory/hooks.json` | event map at the root |
| `muse` | global | `$XDG_CONFIG_HOME/muse/settings.json` or `~/.config/muse/settings.json` | nested `hooks` map plus `schema_version` |
| `codebuddy` | global | `~/.codebuddy/settings.json` | nested `hooks` map |
| `ibm-bob` | global | `~/.bob/settings/settings.json` | nested `hooks` map |
| `zero` | global | `$XDG_CONFIG_HOME/zero/hooks.json` or `~/.config/zero/hooks.json` | exec-form `hooks` array |
| `zcode` | global | `~/.zcode/cli/config.json` | `hooks.events` with `process` entries |
| `kimi` | global | `$KIMI_CODE_HOME/config.toml` or `~/.kimi-code/config.toml` | `[[hooks]]` tables |
| `kiro` | global | `$KIRO_HOME/hooks/<marker>.json` or `~/.kiro/hooks/<marker>.json` | versioned `v1` registration |

Only Claude Code has a project-level hook file. `--agent all --scope project`
writes Claude's file and skips the global-only hosts with a warning.

### Event differences

Most hosts use a subset of the portable events:

- **Claude Code, Grok:** all nine mapped events (session start, prompt, both
  tool boundaries, pre-compact, stop, session end, and both subagent
  boundaries).
- **Codex:** the same, without subagent boundaries.
- **Cursor:** session, prompt, both tool boundaries plus failure, compaction,
  and stop.
- **Gemini:** session start/end, `BeforeTool`/`AfterTool`, and `PreCompress`;
  there is no prompt-submit or stop event.
- **Antigravity:** `PreInvocation` for session start, both tool boundaries, and
  stop; there is no session-end event.
- **Devin:** `PostCompaction` is fed by the portable `post-compact` event; there
  are no subagent boundaries.
- **Command Code:** session start, both tool boundaries, and stop.
- **Copilot CLI:** the full Claude set plus `PostToolUseFailure`.
- **Droid, CodeBuddy:** session start, prompt, both tool boundaries, compaction,
  stop, and session end.
- **Muse:** Droid's set plus `PostToolUseFailure`.
- **IBM Bob:** session start, prompt, both tool boundaries, and stop.
- **Zero:** session start/end, both tool boundaries, and `specialistStart` /
  `specialistStop` for the subagent boundaries; no prompt or compaction event.
- **ZCode:** session start, prompt, both tool boundaries plus failure, and
  stop; stop is also the turn-close path.
- **Kimi:** the full Claude set plus `PostToolUseFailure`.
- **Kiro:** session start, prompt, both tool boundaries, and stop.

Compaction is the one split worth calling out: `pre-compact` reaches the hosts
that fire before compaction, and `post-compact` reaches Devin after its
`PostCompaction`. Set both portable events when one Spec targets both kinds of
host; each host skips the one it does not expose.

An event a host does not expose is skipped with a warning. The writer still
processes the rest of the list.

### Ownership and merging

A re-apply replaces only this tool's hooks. Other tools' entries under the same
event are preserved. `Command` is written verbatim for shell-form hosts, so
inline any environment (`VAR=value command`) or use an absolute path there;
Zero and ZCode take the executable and arguments separately.

- For shell-form hosts (Claude, Codex, Cursor, Gemini, Antigravity, Grok, Devin,
  Command Code, Copilot CLI, Droid, Muse, CodeBuddy, IBM Bob, and Kimi), every
  command must contain `HookMarker`. A command that does not is rejected before
  anything is written.
- Zero, ZCode, and Kiro have no command marker field, so their generated `id`,
  `statusMessage`, or `name` is prefixed with `HookMarker`. `Hook.ID` can
  override the generated identifier.
- `Matcher` is written where the host supports it. Command Code and Copilot CLI
  omit it; Kimi and Antigravity's lifecycle events ignore it with a warning.
- `Args` are appended to the shell command for shell-form hosts, quoted as one
  argument each. Zero and ZCode store the executable and argument vector
  separately, which is their native exec form.
- `TimeoutSeconds` becomes `timeout` on nested handlers, `timeoutMs` on ZCode,
  and `timeout` seconds on Kiro. Kiro defaults SessionStart to 5 seconds and
  every other event to 1.

Dedicated hook files (`codex`, `grok`, `copilot-cli`, `kiro`, `zero`, `cursor`,
`antigravity`, `devin`, `droid`) are removed once the last managed entry is
gone. Shared settings files keep every unrelated key.

```bash
mytool install-hooks --agent claude
mytool install-hooks --agent all --scope global
mytool install-hooks --agent claude --scope project --root /work/project
mytool install-hooks --agent cursor --dry-run
mytool uninstall-hooks --agent claude
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `--agent` | required | Comma-separated hooks hosts, or `all`. |
| `--scope` | `global` | `global`, or `project` for Claude Code. |
| `--root` | cwd, or home for global | Directory the scope resolves against. |
| `--dry-run` | `false` | Report pending changes without writing. |
| `--format` | `text` | `text` or `json`. |

### Unsupported hook hosts

These hosts do not take a plain configuration merge, so they are not in the
hooks matrix:

| Host | Reason |
| --- | --- |
| `opencode`, `opencode2` | A generated TypeScript plugin. |
| `omp`, `pi` | A generated TypeScript extension. |
| `openclaw` | A plugin package. |
| `hermes`, `pool` | Print-only YAML. |
| `kiro-v2` | Hooks live in per-agent configs; use `kiro` for the v3 file. |
| `dsh` | No lifecycle hook configuration. |

Selecting one explicitly returns the reason. `--agent all` includes only the
supported hosts.

## Direct package calls

The CLI is a wrapper around the exported layers:

```go
results, err := install.RunSkills(install.SkillRequest{
    Spec:   spec,
    Agents: []string{"claude", "agents"},
    Scope:  platform.ScopeProject,
    Root:   root,
}, false)

results, err := install.RunHooks(install.HookRequest{
    Spec:  spec,
    Hosts: []string{"claude"},
    Scope: platform.ScopeProject,
    Root:  root,
}, false)
```

The host tables live in `platform/skills` and `platform/hooks`. Each exports
`Adapters`, `Names`, `Lookup`, and `ResolveList`, plus `InstallOne` and
`RemoveOne` for a single host.
