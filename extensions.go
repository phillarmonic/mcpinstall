package mcpinstall

import (
	"github.com/phillarmonic/mcpinstall/platform/hooks"
	"github.com/phillarmonic/mcpinstall/platform/skills"
)

// Skill is one Agent Skill installed by install-skills. Name becomes the
// directory under the host's skills root, and Content is the complete SKILL.md
// body, frontmatter included.
type Skill = skills.Skill

// HookEvent is a portable lifecycle event. Each hooks host maps it onto its
// own event vocabulary.
type HookEvent = hooks.Event

// Hook is one managed lifecycle hook installed by install-hooks.
type Hook = hooks.Hook

// Portable hook events.
const (
	HookSessionStart       = hooks.SessionStart
	HookUserPromptSubmit   = hooks.UserPromptSubmit
	HookPreToolUse         = hooks.PreToolUse
	HookPostToolUse        = hooks.PostToolUse
	HookPostToolUseFailure = hooks.PostToolUseFailure
	HookPreCompact         = hooks.PreCompact
	HookPostCompact        = hooks.PostCompact
	HookStop               = hooks.Stop
	HookSessionEnd         = hooks.SessionEnd
	HookSubagentStart      = hooks.SubagentStart
	HookSubagentStop       = hooks.SubagentStop
)
