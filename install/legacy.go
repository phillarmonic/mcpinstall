package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpinstall"
)

// legacyFile is one pre-rename guidance path under a scope root.
// named files are deleted whenever they exist. markers strips the old
// managed section and leaves the rest of the file. Every other file is
// deleted only when its body still names a legacy server.
type legacyFile struct {
	rel     string
	named   bool
	markers bool
}

// RemoveLegacyGuidance deletes guidance left by a former server name under
// root. Current guidance files are left in place. Paths in the result are
// displayed relative to root. A dry run reports the same paths and writes
// nothing. A spec with no LegacyNames is a no-op.
func RemoveLegacyGuidance(spec mcpinstall.Spec, root string, dryRun bool) ([]string, error) {
	if len(spec.LegacyNames) == 0 || spec.Guidance == nil {
		return nil, nil
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

	var changed []string
	for _, target := range legacyTargets(spec, names) {
		path := filepath.Join(root, target.rel)
		hit, hitErr := removeLegacyFile(spec, path, target, dryRun)
		if hitErr != nil {
			return changed, hitErr
		}
		if hit {
			changed = append(changed, FormatPath(root, path))
		}
	}
	return changed, nil
}

func legacyTargets(spec mcpinstall.Spec, filenames []string) []legacyFile {
	targets := make([]legacyFile, 0, len(filenames)+len(spec.LegacyNames)*2)
	for _, name := range spec.LegacyNames {
		targets = append(targets, legacyFile{rel: filepath.Join(".cursor", "rules", name+".mdc"), named: true})
	}
	for _, filename := range filenames {
		harness := strings.TrimSuffix(filename, filepath.Ext(filename))
		switch harness {
		case "cursor":
			targets = append(targets, legacyFile{rel: filepath.Join(".cursor", "instructions.md")})
		case "codex":
			targets = append(targets,
				legacyFile{rel: "AGENTS.md", markers: true},
				legacyFile{rel: filepath.Join(".codex", "AGENTS.md"), markers: true},
			)
		case "opencode":
			// OpenCode's guidance lives in AGENTS.md, not the default
			// .opencode/instructions.md, so the derived target has to say so.
			targets = append(targets,
				legacyFile{rel: "AGENTS.md", markers: true},
				legacyFile{rel: filepath.Join(".config", "opencode", "AGENTS.md"), markers: true},
			)
		default:
			targets = append(targets, legacyFile{rel: filepath.Join("."+harness, "instructions.md")})
		}
	}
	return targets
}

func removeLegacyFile(spec mcpinstall.Spec, path string, target legacyFile, dryRun bool) (bool, error) {
	// #nosec G304 -- path is under the scope root the caller asked to clean
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}

	switch {
	case target.named:
		return deleteLegacy(path, dryRun)
	case target.markers:
		for _, name := range spec.LegacyNames {
			begin, end := managedMarkers(name)
			next, found := removeMarked(data, begin, end)
			if !found {
				continue
			}
			if dryRun {
				return true, nil
			}
			if len(next) == 0 {
				return deleteLegacy(path, false)
			}
			// #nosec G703 -- path is under the scope root the caller asked to clean
			if writeErr := os.WriteFile(path, next, 0o600); writeErr != nil {
				return false, fmt.Errorf("write %s: %w", path, writeErr)
			}
			return true, nil
		}
		return false, nil
	default:
		for _, name := range spec.LegacyNames {
			if strings.Contains(string(data), name) {
				return deleteLegacy(path, dryRun)
			}
		}
		return false, nil
	}
}

func deleteLegacy(path string, dryRun bool) (bool, error) {
	if dryRun {
		return true, nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("remove %s: %w", path, err)
	}
	return true, nil
}
