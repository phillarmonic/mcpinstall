package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpinstall"
)

const guidanceAll = "all"

// GuidanceRequest describes one project or user-level guidance install or
// uninstall. An empty Harness selects every embedded file.
type GuidanceRequest struct {
	Root     string
	Harness  string
	Force    bool
	DryRun   bool
	Removing bool
	Global   bool
}

type guidanceDest struct {
	rel     string
	managed bool
}

// WriteGuidance writes agent guidance files for a spec to a project root.
func WriteGuidance(spec mcpinstall.Spec, root, harness string) (map[string]string, error) {
	return ApplyGuidance(spec, GuidanceRequest{Root: root, Harness: harness})
}

// ApplyGuidance installs or removes the spec's agent guidance files.
// Returns a map of changed files (harness name -> file path).
func ApplyGuidance(spec mcpinstall.Spec, req GuidanceRequest) (map[string]string, error) {
	if spec.Guidance == nil {
		return nil, errors.New("guidance: spec has no guidance filesystem")
	}

	results := make(map[string]string)
	requested := normalizeGuidanceHarness(req.Harness)
	if requested == "" {
		requested = guidanceAll
	}

	entries, err := fs.ReadDir(spec.Guidance, spec.SourceDir())
	if err != nil {
		return nil, fmt.Errorf("read guidance directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	slices.Sort(names)

	for _, filename := range names {
		harnessName := strings.TrimSuffix(filename, filepath.Ext(filename))
		if requested != guidanceAll && harnessName != requested {
			continue
		}

		dest := guidanceDestFor(spec, harnessName, req.Global)
		outPath := dest.rel
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(req.Root, outPath)
		}
		changed, applyErr := applyGuidanceFile(spec, req, dest, filename, outPath)
		if applyErr != nil {
			return results, applyErr
		}
		if changed {
			results[harnessName] = outPath
		}
	}

	return results, nil
}

// FormatPath renders a written path relative to root, using a ~ prefix when
// root is the home directory.
func FormatPath(root, path string) string {
	return displayPath(root, path)
}

func applyGuidanceFile(spec mcpinstall.Spec, req GuidanceRequest, dest guidanceDest, filename, outPath string) (bool, error) {
	if dest.managed {
		return applyManagedGuidance(spec, req, filename, outPath)
	}
	return applyWholeGuidance(spec, req, dest.rel, filename, outPath)
}

func applyWholeGuidance(spec mcpinstall.Spec, req GuidanceRequest, rel, filename, outPath string) (bool, error) {
	if req.Removing {
		if _, statErr := os.Stat(outPath); errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		} else if statErr != nil {
			return false, fmt.Errorf("stat %s: %w", outPath, statErr)
		}
		if !req.DryRun {
			if err := os.Remove(outPath); err != nil {
				return false, fmt.Errorf("remove %s: %w", outPath, err)
			}
		}
		return true, nil
	}

	if _, statErr := os.Stat(outPath); statErr == nil && !req.Force {
		return false, nil
	}

	content, err := readEncodedGuidance(spec, filename, rel)
	if err != nil {
		return false, err
	}
	return writeGuidanceFile(req.DryRun, outPath, content)
}

func applyManagedGuidance(spec mcpinstall.Spec, req GuidanceRequest, filename, outPath string) (bool, error) {
	// #nosec G304 -- dest is under the requested guidance root
	existing, readErr := os.ReadFile(outPath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return false, fmt.Errorf("read %s: %w", outPath, readErr)
	}
	if errors.Is(readErr, os.ErrNotExist) {
		existing = nil
	}

	begin, end := managedMarkers(spec.Marker)

	if req.Removing {
		next, found := removeManaged(existing, begin, end)
		if !found {
			return false, nil
		}
		if req.DryRun {
			return true, nil
		}
		if len(next) == 0 {
			if err := os.Remove(outPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return false, fmt.Errorf("remove %s: %w", outPath, err)
			}
			return true, nil
		}
		return writeGuidanceFile(false, outPath, next)
	}

	content, err := readEncodedGuidance(spec, filename, filepath.Base(outPath))
	if err != nil {
		return false, err
	}
	next, changed := upsertManaged(existing, content, begin, end)
	if !changed {
		return false, nil
	}
	return writeGuidanceFile(req.DryRun, outPath, next)
}

func readEncodedGuidance(spec mcpinstall.Spec, filename, rel string) ([]byte, error) {
	content, err := fs.ReadFile(spec.Guidance, filepath.Join(spec.SourceDir(), filename))
	if err != nil {
		return nil, fmt.Errorf("read guidance file %s: %w", filename, err)
	}
	return encodeGuidance(spec, rel, content), nil
}

func writeGuidanceFile(dryRun bool, outPath string, content []byte) (bool, error) {
	if dryRun {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o750); err != nil {
		return false, fmt.Errorf("create %s directory: %w", filepath.Dir(outPath), err)
	}
	// #nosec G304 G703 -- dest is under the requested guidance root
	if err := os.WriteFile(outPath, content, 0o600); err != nil {
		return false, fmt.Errorf("write %s: %w", outPath, err)
	}
	return true, nil
}

func normalizeGuidanceHarness(name string) string {
	folded := strings.ToLower(strings.TrimSpace(name))
	folded = strings.ReplaceAll(folded, "_", "-")
	folded = strings.Join(strings.Fields(folded), "-")
	switch folded {
	case "openai", "chatgpt", "chat-gpt":
		return "codex"
	default:
		return folded
	}
}

func guidanceDestFor(spec mcpinstall.Spec, harnessName string, global bool) guidanceDest {
	switch {
	case harnessName == "cursor" && global:
		return guidanceDest{rel: filepath.Join(".cursor", "rules", spec.CursorRule())}
	case harnessName == "codex" && global:
		return guidanceDest{rel: filepath.Join(".codex", "AGENTS.md"), managed: true}
	case harnessName == "codex":
		return guidanceDest{rel: "AGENTS.md", managed: true}
	// OpenCode reads AGENTS.md, the same file Codex uses, so both harnesses
	// share one managed section rather than competing for the file: the body is
	// identical, and upsertManaged rewrites the section in place. The
	// user-level copy lives beside OpenCode's own configuration.
	case harnessName == "opencode" && global:
		return guidanceDest{rel: filepath.Join(".config", "opencode", "AGENTS.md"), managed: true}
	case harnessName == "opencode":
		return guidanceDest{rel: "AGENTS.md", managed: true}
	// Claude Code reads ~/.claude/CLAUDE.md as user memory and
	// .claude/CLAUDE.md as project memory. Copilot CLI also reads the project
	// file, so both harnesses share one managed section there.
	case harnessName == "claude":
		return guidanceDest{rel: filepath.Join(".claude", "CLAUDE.md"), managed: true}
	// Copilot CLI reads ~/.copilot/copilot-instructions.md as personal
	// instructions. The repository file is the one both Copilot CLI and
	// Copilot in VS Code read.
	case harnessName == "copilot" && global:
		return guidanceDest{rel: filepath.Join(".copilot", "copilot-instructions.md"), managed: true}
	case harnessName == "copilot":
		return guidanceDest{rel: filepath.Join(".github", "copilot-instructions.md"), managed: true}
	// Copilot for JetBrains reads the same repository file, plus its own
	// global-copilot-instructions.md in the plugin's configuration directory.
	case harnessName == "copilot-jetbrains" && global:
		return guidanceDest{rel: filepath.Join(copilotJetBrainsConfigDir(), "github-copilot", "intellij", "global-copilot-instructions.md"), managed: true}
	case harnessName == "copilot-jetbrains":
		return guidanceDest{rel: filepath.Join(".github", "copilot-instructions.md"), managed: true}
	default:
		return guidanceDest{rel: filepath.Join("."+harnessName, "instructions.md")}
	}
}

// copilotJetBrainsConfigDir returns the directory the JetBrains plugin
// resolves its configuration against, mirroring the MCP adapter: LOCALAPPDATA
// on Windows and XDG_CONFIG_HOME elsewhere, each falling back to the home
// directory. An absolute variable is returned as-is so the caller can write
// outside the home root.
func copilotJetBrainsConfigDir() string {
	return copilotJetBrainsConfigDirFor(runtime.GOOS, os.Getenv)
}

func copilotJetBrainsConfigDirFor(goos string, lookupEnv func(string) string) string {
	switch goos {
	case "windows":
		if local := strings.TrimSpace(lookupEnv("LOCALAPPDATA")); local != "" {
			return local
		}
		return filepath.Join("AppData", "Local")
	default:
		if configHome := strings.TrimSpace(lookupEnv("XDG_CONFIG_HOME")); configHome != "" {
			return configHome
		}
		return ".config"
	}
}

func encodeGuidance(spec mcpinstall.Spec, rel string, body []byte) []byte {
	if filepath.Ext(rel) != ".mdc" {
		return body
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: ")
	b.WriteString(spec.CursorDescription())
	b.WriteString("\nalwaysApply: true\n---\n\n")
	b.Write(body)
	return []byte(b.String())
}
