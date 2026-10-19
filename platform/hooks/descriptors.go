package hooks

import "github.com/phillarmonic/mcpinstall/platform"

var (
	projectGlobal = []platform.Scope{platform.ScopeProject, platform.ScopeGlobal}
	globalOnly    = []platform.Scope{platform.ScopeGlobal}
)

// The event vocabularies below are the host-native names, kept next to the
// canonical events they satisfy. A host with no equivalent for a portable
// event simply omits it, and a Spec hook for that event is skipped with a
// warning.
var (
	claudeEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		PreCompact:       "PreCompact",
		Stop:             "Stop",
		SessionEnd:       "SessionEnd",
		SubagentStart:    "SubagentStart",
		SubagentStop:     "SubagentStop",
	}
	codexEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		PreCompact:       "PreCompact",
		Stop:             "Stop",
		SessionEnd:       "SessionEnd",
	}
	cursorEvents = map[Event]string{
		SessionStart:       "sessionStart",
		SessionEnd:         "sessionEnd",
		UserPromptSubmit:   "beforeSubmitPrompt",
		PreToolUse:         "preToolUse",
		PostToolUse:        "postToolUse",
		PostToolUseFailure: "postToolUseFailure",
		PreCompact:         "preCompact",
		Stop:               "stop",
	}
	geminiEvents = map[Event]string{
		SessionStart: "SessionStart",
		SessionEnd:   "SessionEnd",
		PreToolUse:   "BeforeTool",
		PostToolUse:  "AfterTool",
		PreCompact:   "PreCompress",
	}
	antigravityEvents = map[Event]string{
		SessionStart: "PreInvocation",
		PreToolUse:   "PreToolUse",
		PostToolUse:  "PostToolUse",
		Stop:         "Stop",
	}
	antigravityTools = map[Event]bool{
		PreToolUse:  true,
		PostToolUse: true,
	}
	devinEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		PostCompact:      "PostCompaction",
		Stop:             "Stop",
		SessionEnd:       "SessionEnd",
	}
	commandCodeEvents = map[Event]string{
		SessionStart: "SessionStart",
		PreToolUse:   "PreToolUse",
		PostToolUse:  "PostToolUse",
		Stop:         "Stop",
	}
	copilotEvents = map[Event]string{
		SessionStart:       "SessionStart",
		UserPromptSubmit:   "UserPromptSubmit",
		PreToolUse:         "PreToolUse",
		PostToolUse:        "PostToolUse",
		PostToolUseFailure: "PostToolUseFailure",
		PreCompact:         "PreCompact",
		Stop:               "Stop",
		SessionEnd:         "SessionEnd",
		SubagentStart:      "SubagentStart",
		SubagentStop:       "SubagentStop",
	}
	droidEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		PreCompact:       "PreCompact",
		Stop:             "Stop",
		SessionEnd:       "SessionEnd",
	}
	museEvents = map[Event]string{
		SessionStart:       "SessionStart",
		UserPromptSubmit:   "UserPromptSubmit",
		PreToolUse:         "PreToolUse",
		PostToolUse:        "PostToolUse",
		PostToolUseFailure: "PostToolUseFailure",
		PreCompact:         "PreCompact",
		Stop:               "Stop",
		SessionEnd:         "SessionEnd",
	}
	codebuddyEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		PreCompact:       "PreCompact",
		Stop:             "Stop",
		SessionEnd:       "SessionEnd",
	}
	ibmBobEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		Stop:             "Stop",
	}
	zeroEvents = map[Event]string{
		SessionStart:  "sessionStart",
		SessionEnd:    "sessionEnd",
		PreToolUse:    "beforeTool",
		PostToolUse:   "afterTool",
		SubagentStart: "specialistStart",
		SubagentStop:  "specialistStop",
	}
	zcodeEvents = map[Event]string{
		SessionStart:       "SessionStart",
		UserPromptSubmit:   "UserPromptSubmit",
		PreToolUse:         "PreToolUse",
		PostToolUse:        "PostToolUse",
		PostToolUseFailure: "PostToolUseFailure",
		Stop:               "Stop",
	}
	kimiEvents = map[Event]string{
		SessionStart:       "SessionStart",
		UserPromptSubmit:   "UserPromptSubmit",
		PreToolUse:         "PreToolUse",
		PostToolUse:        "PostToolUse",
		PostToolUseFailure: "PostToolUseFailure",
		PreCompact:         "PreCompact",
		Stop:               "Stop",
		SessionEnd:         "SessionEnd",
		SubagentStart:      "SubagentStart",
		SubagentStop:       "SubagentStop",
	}
	kiroEvents = map[Event]string{
		SessionStart:     "SessionStart",
		UserPromptSubmit: "UserPromptSubmit",
		PreToolUse:       "PreToolUse",
		PostToolUse:      "PostToolUse",
		Stop:             "Stop",
	}
)

// descriptors is the host matrix. A new host is one entry plus a path
// resolver; the writers are shared.
var descriptors = []Descriptor{
	{
		Agent:       "claude",
		Label:       "Claude Code hooks",
		Scopes:      projectGlobal,
		Path:        claudePath,
		Shape:       NestedJSON,
		Events:      claudeEvents,
		RootWrapper: true,
	},
	{
		Agent:           "codex",
		Label:           "Codex CLI hooks",
		Scopes:          globalOnly,
		Path:            codexPath,
		Shape:           NestedJSON,
		Events:          codexEvents,
		RootWrapper:     true,
		DeleteWhenEmpty: true,
	},
	{
		Agent:           "cursor",
		Label:           "Cursor hooks",
		Scopes:          globalOnly,
		Path:            cursorPath,
		Shape:           FlatJSON,
		Events:          cursorEvents,
		RootWrapper:     true,
		Version:         1,
		DeleteWhenEmpty: true,
	},
	{
		Agent:       "gemini",
		Label:       "Gemini CLI hooks",
		Scopes:      globalOnly,
		Path:        geminiPath,
		Shape:       NestedJSON,
		Events:      geminiEvents,
		RootWrapper: true,
	},
	{
		Agent:           "antigravity",
		Label:           "Antigravity CLI hooks",
		Scopes:          globalOnly,
		Path:            antigravityPath,
		Shape:           AntigravityJSON,
		Events:          antigravityEvents,
		ToolEvents:      antigravityTools,
		DeleteWhenEmpty: true,
	},
	{
		Agent:           "grok",
		Label:           "Grok Build CLI hooks",
		Scopes:          globalOnly,
		Path:            grokPath,
		Shape:           NestedJSON,
		Events:          claudeEvents,
		RootWrapper:     true,
		DeleteWhenEmpty: true,
	},
	{
		Agent:           "devin",
		Label:           "Devin CLI hooks",
		Scopes:          globalOnly,
		Path:            devinPath,
		Shape:           NestedJSON,
		Events:          devinEvents,
		RootWrapper:     false,
		DeleteWhenEmpty: true,
	},
	{
		Agent:       "command-code",
		Label:       "Command Code hooks",
		Scopes:      globalOnly,
		Path:        commandCodePath,
		Shape:       NestedJSON,
		Events:      commandCodeEvents,
		RootWrapper: true,
		NoMatcher:   true,
	},
	{
		Agent:           "copilot-cli",
		Label:           "GitHub Copilot CLI hooks",
		Scopes:          globalOnly,
		Path:            copilotCLIPath,
		Shape:           FlatJSON,
		Events:          copilotEvents,
		RootWrapper:     true,
		NoMatcher:       true,
		Version:         1,
		DeleteWhenEmpty: true,
	},
	{
		Agent:           "droid",
		Label:           "Factory Droid hooks",
		Scopes:          globalOnly,
		Path:            droidPath,
		Shape:           NestedJSON,
		Events:          droidEvents,
		RootWrapper:     false,
		DeleteWhenEmpty: true,
	},
	{
		Agent:            "muse",
		Label:            "Meta Muse Code hooks",
		Scopes:           globalOnly,
		Path:             musePath,
		Shape:            NestedJSON,
		Events:           museEvents,
		RootWrapper:      true,
		AddSchemaVersion: true,
	},
	{
		Agent:       "codebuddy",
		Label:       "Tencent CodeBuddy hooks",
		Scopes:      globalOnly,
		Path:        codebuddyPath,
		Shape:       NestedJSON,
		Events:      codebuddyEvents,
		RootWrapper: true,
	},
	{
		Agent:       "ibm-bob",
		Label:       "IBM Bob hooks",
		Scopes:      globalOnly,
		Path:        ibmBobPath,
		Shape:       NestedJSON,
		Events:      ibmBobEvents,
		RootWrapper: true,
	},
	{
		Agent:           "zero",
		Label:           "Zero hooks",
		Scopes:          globalOnly,
		Path:            zeroPath,
		Shape:           ZeroJSON,
		Events:          zeroEvents,
		DeleteWhenEmpty: true,
	},
	{
		Agent:  "zcode",
		Label:  "ZCode hooks",
		Scopes: globalOnly,
		Path:   zcodePath,
		Shape:  ZCodeJSON,
		Events: zcodeEvents,
	},
	{
		Agent:  "kimi",
		Label:  "Kimi Code hooks",
		Scopes: globalOnly,
		Path:   kimiPath,
		Shape:  KimiTOML,
		Events: kimiEvents,
	},
	{
		Agent:           "kiro",
		Label:           "Kiro CLI v3 hooks",
		Scopes:          globalOnly,
		Path:            kiroPath,
		Shape:           KiroV3JSON,
		Events:          kiroEvents,
		DeleteWhenEmpty: true,
	},
}

var aliases = map[string]string{
	"claude-code":        "claude",
	"claudecode":         "claude",
	"codex-cli":          "codex",
	"cursor-agent":       "cursor",
	"gemini-cli":         "gemini",
	"agy":                "antigravity",
	"antigravity-cli":    "antigravity",
	"grok-build":         "grok",
	"grok-cli":           "grok",
	"devin-cli":          "devin",
	"commandcode":        "command-code",
	"cmdc":               "command-code",
	"command-code-cli":   "command-code",
	"copilot":            "copilot-cli",
	"github-copilot":     "copilot-cli",
	"github-copilot-cli": "copilot-cli",
	"factory-droid":      "droid",
	"muse-code":          "muse",
	"musecode":           "muse",
	"code-buddy":         "codebuddy",
	"bob":                "ibm-bob",
	"ibm-bob-shell":      "ibm-bob",
	"z-code":             "zcode",
	"zai":                "zcode",
	"kimi-code":          "kimi",
	"kiro-cli":           "kiro",
	"kiro-v3":            "kiro",
	"kiro-cli-v3":        "kiro",
}

// unsupported names hosts whose hooks are not a plain configuration merge.
// They remain selectable enough to explain why they are absent.
var unsupported = map[string]string{
	"opencode":  "OpenCode hooks are a generated TypeScript plugin",
	"opencode2": "OpenCode V2 hooks are a generated TypeScript plugin",
	"omp":       "Oh My Pi hooks are a generated TypeScript extension",
	"pi":        "Pi hooks are a generated TypeScript extension",
	"openclaw":  "OpenClaw hooks are a plugin package",
	"hermes":    "Hermes hooks are print-only YAML",
	"pool":      "Pool hooks are print-only YAML",
	"kiro-v2":   "Kiro v2 hooks live in per-agent configs; use kiro for v3",
	"dsh":       "DSH has no lifecycle hook configuration",
}
