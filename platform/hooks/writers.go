package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform/common"
)

func writeInstall(d Descriptor, path string, list []Hook, opts Options) ([]string, []string, error) {
	if d.Agent == "droid" {
		if err := ensureDroidHooksFile(path); err != nil {
			return nil, nil, err
		}
	}
	switch d.Shape {
	case NestedJSON, FlatJSON, AntigravityJSON:
		return writeJSONHooks(d, path, list, opts, false)
	case ZeroJSON:
		return writeZero(d, path, list, opts, false)
	case ZCodeJSON:
		return writeZCode(path, list, opts, false)
	case KimiTOML:
		return writeKimi(path, list, opts, false)
	case KiroV3JSON:
		return writeKiro(d, path, list, opts, false)
	default:
		return nil, nil, fmt.Errorf("unsupported hooks shape for %s", d.Agent)
	}
}

func writeRemove(d Descriptor, path string, list []Hook, opts Options) ([]string, []string, error) {
	switch d.Shape {
	case NestedJSON, FlatJSON, AntigravityJSON:
		return writeJSONHooks(d, path, list, opts, true)
	case ZeroJSON:
		return writeZero(d, path, list, opts, true)
	case ZCodeJSON:
		return writeZCode(path, list, opts, true)
	case KimiTOML:
		return writeKimi(path, list, opts, true)
	case KiroV3JSON:
		return writeKiro(d, path, list, opts, true)
	default:
		return nil, nil, fmt.Errorf("unsupported hooks shape for %s", d.Agent)
	}
}

func writeJSONHooks(d Descriptor, path string, list []Hook, opts Options, removing bool) ([]string, []string, error) {
	cfg, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return nil, nil, err
	}
	if malformed {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", d.Label, path)}, nil
	}

	container, cleanup, ok := jsonContainer(cfg, d, opts.Marker)
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because its hooks container is not an object", d.Label, path)}, nil
	}

	var warnings []string
	// Withdraw this tool's entries wherever they sit before adding the current
	// set. That also clears entries left under an event a previous version
	// mapped differently.
	changed := stripManaged(container, opts.Marker)
	if !removing {
		for _, hook := range list {
			hostEvent, ok := d.Events[hook.Event]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("%s has no %s event; skipped", d.Label, hook.Event))
				continue
			}
			if hook.Matcher != "" && (d.NoMatcher || d.Shape == AntigravityJSON && !d.ToolEvents[hook.Event]) {
				warnings = append(warnings, fmt.Sprintf("%s has no matcher for %s; it was ignored", d.Label, hook.Event))
			}
			entries, ok := sliceOfAny(container[hostEvent])
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preserved %s because it is not an array", hostEvent))
				continue
			}
			entries = append(entries, buildEntry(d, hook))
			container[hostEvent] = entries
			changed = true
		}
		if d.Version > 0 {
			if _, exists := cfg["version"]; !exists {
				cfg["version"] = d.Version
				changed = true
			}
		}
		if d.AddSchemaVersion {
			if _, exists := cfg["schema_version"]; !exists {
				cfg["schema_version"] = 1
				changed = true
			}
		}
	}

	if !changed {
		return nil, warnings, nil
	}
	cleanup()

	if d.DeleteWhenEmpty && metadataOnly(cfg) {
		removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
		if removeErr != nil {
			return nil, warnings, removeErr
		}
		if removed {
			return []string{path}, warnings, nil
		}
		return nil, warnings, nil
	}
	if len(cfg) == 0 {
		removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
		if removeErr != nil {
			return nil, warnings, removeErr
		}
		if removed {
			return []string{path}, warnings, nil
		}
		return nil, warnings, nil
	}
	wrote, warning, err := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
	if warning != "" {
		warnings = append(warnings, warning)
	}
	if err != nil {
		return nil, warnings, err
	}
	if wrote {
		return []string{path}, warnings, nil
	}
	return nil, warnings, nil
}

// jsonContainer returns the object that holds the event map and a cleanup
// function that removes an emptied wrapper or group.
func jsonContainer(cfg map[string]any, d Descriptor, marker string) (map[string]any, func(), bool) {
	if d.Shape == AntigravityJSON {
		group := marker
		value, exists := cfg[group]
		if !exists {
			container := map[string]any{}
			cfg[group] = container
			return container, func() {
				if len(container) == 0 {
					delete(cfg, group)
				}
			}, true
		}
		container, ok := value.(map[string]any)
		if !ok {
			return nil, nil, false
		}
		return container, func() {
			if len(container) == 0 {
				delete(cfg, group)
			}
		}, true
	}
	if d.RootWrapper {
		value, exists := cfg["hooks"]
		if !exists {
			container := map[string]any{}
			cfg["hooks"] = container
			return container, func() {
				if len(container) == 0 {
					delete(cfg, "hooks")
				}
			}, true
		}
		container, ok := value.(map[string]any)
		if !ok {
			return nil, nil, false
		}
		return container, func() {
			if len(container) == 0 {
				delete(cfg, "hooks")
			}
		}, true
	}
	return cfg, func() {}, true
}

func buildEntry(d Descriptor, hook Hook) map[string]any {
	command := commandLine(hook)
	if d.Shape == AntigravityJSON && !d.ToolEvents[hook.Event] {
		return flatEntry(command, hook, true)
	}
	if d.Shape == FlatJSON {
		return flatEntry(command, hook, d.NoMatcher)
	}
	inner := map[string]any{"type": "command", "command": command}
	if hook.TimeoutSeconds > 0 {
		inner["timeout"] = hook.TimeoutSeconds
	}
	entry := map[string]any{"hooks": []any{inner}}
	if !d.NoMatcher {
		entry["matcher"] = hook.Matcher
	}
	return entry
}

func flatEntry(command string, hook Hook, noMatcher bool) map[string]any {
	entry := map[string]any{"type": "command", "command": command}
	if !noMatcher {
		entry["matcher"] = hook.Matcher
	}
	if hook.TimeoutSeconds > 0 {
		entry["timeout"] = hook.TimeoutSeconds
	}
	return entry
}

// ensureDroidHooksFile refuses to create hooks.json while the sibling
// settings.json still holds Droid hooks. Droid reads those only while
// hooks.json is absent, so creating the file would silently disable them.
func ensureDroidHooksFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	settings := filepath.Join(filepath.Dir(path), "settings.json")
	cfg, malformed, err := common.ReadJSONObject(settings)
	if err != nil {
		return err
	}
	if malformed {
		return nil
	}
	if _, exists := cfg["hooks"]; !exists {
		return nil
	}
	return fmt.Errorf("%s keeps Droid hooks under its hooks key, and Droid reads those only while %s does not exist; move that hooks object into %s and re-run", settings, path, path)
}

// stripManaged removes this tool's entries from every event array in the
// container and deletes an event whose array becomes empty.
func stripManaged(container map[string]any, marker string) bool {
	changed := false
	for event, value := range container {
		entries, ok := sliceOfAny(value)
		if !ok {
			continue
		}
		filtered := removeManaged(entries, marker)
		if len(filtered) == len(entries) {
			continue
		}
		changed = true
		if len(filtered) == 0 {
			delete(container, event)
		} else {
			container[event] = filtered
		}
	}
	return changed
}

func removeManaged(entries []any, marker string) []any {
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		if !isManagedEntry(entry, marker) {
			out = append(out, entry)
		}
	}
	return out
}

func isManagedEntry(entry any, marker string) bool {
	value, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	if command, ok := value["command"].(string); ok && strings.Contains(command, marker) {
		return true
	}
	if inner, ok := sliceOfAny(value["hooks"]); ok {
		for _, item := range inner {
			if isManagedEntry(item, marker) {
				return true
			}
		}
	}
	return false
}

func metadataOnly(cfg map[string]any) bool {
	if len(cfg) == 0 {
		return true
	}
	for key := range cfg {
		switch key {
		case "version", "schema_version":
		default:
			return false
		}
	}
	return true
}

func sliceOfAny(value any) ([]any, bool) {
	if value == nil {
		return nil, true
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range rv.Len() {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

func commandLine(hook Hook) string {
	if len(hook.Args) == 0 {
		return hook.Command
	}
	parts := make([]string, 0, len(hook.Args)+1)
	parts = append(parts, hook.Command)
	for _, arg := range hook.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	safe := true
	for _, r := range arg {
		letterOrDigit := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if !letterOrDigit && !strings.ContainsRune("-_. /:=@%+,~", r) {
			safe = false
			break
		}
	}
	if safe && !strings.ContainsRune(arg, ' ') {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

func writeZero(d Descriptor, path string, list []Hook, opts Options, removing bool) ([]string, []string, error) {
	cfg, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return nil, nil, err
	}
	if malformed {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", d.Label, path)}, nil
	}

	entries, ok := sliceOfAny(cfg["hooks"])
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because hooks is not an array", d.Label, path)}, nil
	}

	kept := make([]any, 0, len(entries))
	used := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if isManagedZeroEntry(entry, opts.Marker) {
			continue
		}
		kept = append(kept, entry)
		if value, ok := entry.(map[string]any); ok {
			if id, ok := value["id"].(string); ok {
				used[id] = struct{}{}
			}
		}
	}

	if removing {
		if len(kept) == len(entries) {
			return nil, nil, nil
		}
		if len(kept) == 0 && d.DeleteWhenEmpty {
			removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
			if removeErr != nil {
				return nil, nil, removeErr
			}
			if removed {
				return []string{path}, nil, nil
			}
			return nil, nil, nil
		}
		cfg["hooks"] = kept
		wrote, warning, writeErr := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
		return pathIfChanged(wrote, path), warningSlice(warning), writeErr
	}

	var warnings []string
	added := false
	for _, hook := range list {
		hostEvent, ok := d.Events[hook.Event]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s has no %s event; skipped", d.Label, hook.Event))
			continue
		}
		kept = append(kept, zeroEntry(hook, hostEvent, opts.Marker, used))
		added = true
	}
	if !added {
		return nil, warnings, nil
	}
	if _, exists := cfg["enabled"]; !exists {
		cfg["enabled"] = true
	}
	cfg["hooks"] = kept
	wrote, warning, writeErr := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
	if warning != "" {
		warnings = append(warnings, warning)
	}
	return pathIfChanged(wrote, path), warnings, writeErr
}

func zeroEntry(hook Hook, hostEvent, marker string, used map[string]struct{}) map[string]any {
	id := hook.ID
	if id == "" {
		id = marker + "-" + string(hook.Event)
	}
	if !strings.HasPrefix(id, marker+"-") {
		id = marker + "-" + id
	}
	id = uniqueID(id, used)
	args := hook.Args
	if args == nil {
		args = []string{}
	}
	return map[string]any{
		"id":      id,
		"name":    marker + " " + string(hook.Event),
		"event":   hostEvent,
		"command": hook.Command,
		"args":    args,
		"enabled": true,
	}
}

func isManagedZeroEntry(entry any, marker string) bool {
	value, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	id, _ := value["id"].(string)
	return strings.HasPrefix(id, marker+"-")
}

func uniqueID(id string, used map[string]struct{}) string {
	if _, exists := used[id]; !exists {
		used[id] = struct{}{}
		return id
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", id, suffix)
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func writeZCode(path string, list []Hook, opts Options, removing bool) ([]string, []string, error) {
	cfg, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return nil, nil, err
	}
	if malformed {
		return nil, []string{fmt.Sprintf("preserved existing ZCode %s because the file is not valid JSON", path)}, nil
	}

	hooksValue, exists := cfg["hooks"]
	if !exists {
		if removing {
			return nil, nil, nil
		}
		cfg["hooks"] = map[string]any{}
		hooksValue = cfg["hooks"]
	}
	hooksMap, ok := hooksValue.(map[string]any)
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing ZCode %s because hooks is not an object", path)}, nil
	}

	eventsValue, exists := hooksMap["events"]
	if !exists {
		if removing {
			return nil, nil, nil
		}
		hooksMap["events"] = map[string]any{}
		eventsValue = hooksMap["events"]
	}
	eventsMap, ok := eventsValue.(map[string]any)
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing ZCode %s because hooks.events is not an object", path)}, nil
	}

	var warnings []string
	changed := stripZCodeEvents(eventsMap, opts.Marker)
	if !removing {
		added := false
		for _, hook := range list {
			hostEvent, ok := zcodeEvents[hook.Event]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("ZCode has no %s event; skipped", hook.Event))
				continue
			}
			groups, ok := sliceOfAny(eventsMap[hostEvent])
			if !ok {
				continue
			}
			groups = append(groups, zcodeGroup(hook, hostEvent, opts.Marker))
			eventsMap[hostEvent] = groups
			added = true
		}
		if !added {
			return nil, warnings, nil
		}
		if _, exists := hooksMap["enabled"]; !exists {
			hooksMap["enabled"] = true
		}
		if _, exists := hooksMap["maxOutputBytes"]; !exists {
			hooksMap["maxOutputBytes"] = 262144
		}
		changed = true
	}

	if !changed {
		return nil, warnings, nil
	}
	wrote, warning, writeErr := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
	if warning != "" {
		warnings = append(warnings, warning)
	}
	return pathIfChanged(wrote, path), warnings, writeErr
}

// stripZCodeEvents removes managed groups from every event key, including
// keys this version no longer maps, and prunes an event that becomes empty.
func stripZCodeEvents(events map[string]any, marker string) bool {
	changed := false
	for event, value := range events {
		groups, ok := sliceOfAny(value)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(groups))
		for _, group := range groups {
			if !isManagedZCodeGroup(group, marker) {
				filtered = append(filtered, group)
			}
		}
		if len(filtered) == len(groups) {
			continue
		}
		changed = true
		if len(filtered) == 0 {
			delete(events, event)
		} else {
			events[event] = filtered
		}
	}
	return changed
}

func zcodeGroup(hook Hook, hostEvent, marker string) map[string]any {
	args := hook.Args
	if args == nil {
		args = []string{}
	}
	inner := map[string]any{
		"type":          "process",
		"command":       hook.Command,
		"args":          args,
		"enabled":       true,
		"statusMessage": marker + " " + string(hook.Event),
	}
	if hook.TimeoutSeconds > 0 {
		inner["timeoutMs"] = hook.TimeoutSeconds * 1000
	}
	group := map[string]any{"hooks": []any{inner}}
	if hook.Matcher != "" {
		group["matcher"] = hook.Matcher
	}
	return group
}

func isManagedZCodeGroup(group any, marker string) bool {
	value, ok := group.(map[string]any)
	if !ok {
		return false
	}
	inner, ok := sliceOfAny(value["hooks"])
	if !ok {
		return false
	}
	for _, item := range inner {
		handler, ok := item.(map[string]any)
		if !ok {
			continue
		}
		message, _ := handler["statusMessage"].(string)
		if strings.HasPrefix(message, marker+" ") {
			return true
		}
	}
	return false
}

func writeKimi(path string, list []Hook, opts Options, removing bool) ([]string, []string, error) {
	cfg, malformed, err := common.ReadTOMLObject(path)
	if err != nil {
		return nil, nil, err
	}
	if malformed {
		return nil, []string{fmt.Sprintf("preserved existing Kimi Code %s because the file is not valid TOML", path)}, nil
	}

	entries, ok := sliceOfAny(cfg["hooks"])
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing Kimi Code %s because hooks is not an array", path)}, nil
	}
	var warnings []string

	if removing {
		kept := make([]any, 0, len(entries))
		for _, entry := range entries {
			if !isManagedKimiEntry(entry, opts.Marker) {
				kept = append(kept, entry)
			}
		}
		if len(kept) == len(entries) {
			return nil, nil, nil
		}
		if len(kept) == 0 {
			delete(cfg, "hooks")
		} else {
			cfg["hooks"] = kept
		}
		if len(cfg) == 0 {
			removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
			if removeErr != nil {
				return nil, nil, removeErr
			}
			if removed {
				return []string{path}, nil, nil
			}
			return nil, nil, nil
		}
		wrote, warning, writeErr := common.WriteTOML(path, cfg, common.Options{DryRun: opts.DryRun})
		return pathIfChanged(wrote, path), warningSlice(warning), writeErr
	}

	added := false
	for _, hook := range list {
		hostEvent, ok := kimiEvents[hook.Event]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Kimi Code has no %s event; skipped", hook.Event))
			continue
		}
		if hook.Matcher != "" {
			warnings = append(warnings, fmt.Sprintf("Kimi Code has no matcher for %s; it was ignored", hook.Event))
		}
		entries = append(entries, map[string]any{
			"event":   hostEvent,
			"command": commandLine(hook),
		})
		added = true
	}
	if !added {
		return nil, warnings, nil
	}
	cfg["hooks"] = entries
	wrote, warning, writeErr := common.WriteTOML(path, cfg, common.Options{DryRun: opts.DryRun})
	if warning != "" {
		warnings = append(warnings, warning)
	}
	return pathIfChanged(wrote, path), warnings, writeErr
}

func isManagedKimiEntry(entry any, marker string) bool {
	value, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	command, _ := value["command"].(string)
	return strings.Contains(command, marker)
}

func writeKiro(d Descriptor, path string, list []Hook, opts Options, removing bool) ([]string, []string, error) {
	cfg, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return nil, nil, err
	}
	if malformed {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", d.Label, path)}, nil
	}
	if value, exists := cfg["version"]; exists {
		version, ok := value.(string)
		if !ok || version != "v1" {
			return nil, []string{fmt.Sprintf("preserved existing %s %s because version is %v, not v1", d.Label, path, value)}, nil
		}
	}

	entries, ok := sliceOfAny(cfg["hooks"])
	if !ok {
		return nil, []string{fmt.Sprintf("preserved existing %s %s because hooks is not an array", d.Label, path)}, nil
	}

	if removing {
		kept := make([]any, 0, len(entries))
		for _, entry := range entries {
			if !isManagedKiroEntry(entry, opts.Marker) {
				kept = append(kept, entry)
			}
		}
		if len(kept) == len(entries) {
			return nil, nil, nil
		}
		if len(kept) == 0 && d.DeleteWhenEmpty {
			removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
			if removeErr != nil {
				return nil, nil, removeErr
			}
			if removed {
				return []string{path}, nil, nil
			}
			return nil, nil, nil
		}
		cfg["hooks"] = kept
		wrote, warning, writeErr := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
		return pathIfChanged(wrote, path), warningSlice(warning), writeErr
	}

	used := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if value, ok := entry.(map[string]any); ok {
			if name, ok := value["name"].(string); ok {
				used[name] = struct{}{}
			}
		}
	}
	var warnings []string
	added := false
	for _, hook := range list {
		hostEvent, ok := d.Events[hook.Event]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%s has no %s event; skipped", d.Label, hook.Event))
			continue
		}
		entries = append(entries, kiroEntry(hook, hostEvent, opts.Marker, used))
		added = true
	}
	if !added {
		return nil, warnings, nil
	}
	cfg["version"] = "v1"
	cfg["hooks"] = entries
	wrote, warning, writeErr := common.WriteJSON(path, cfg, common.Options{DryRun: opts.DryRun})
	if warning != "" {
		warnings = append(warnings, warning)
	}
	return pathIfChanged(wrote, path), warnings, writeErr
}

func kiroEntry(hook Hook, hostEvent, marker string, used map[string]struct{}) map[string]any {
	name := hook.ID
	if name == "" {
		name = marker + "-" + string(hook.Event)
	}
	if !strings.HasPrefix(name, marker+"-") {
		name = marker + "-" + name
	}
	name = uniqueID(name, used)
	timeout := hook.TimeoutSeconds
	if timeout == 0 {
		if hook.Event == SessionStart {
			timeout = 5
		} else {
			timeout = 1
		}
	}
	return map[string]any{
		"name":    name,
		"trigger": hostEvent,
		"action": map[string]any{
			"type":    "command",
			"command": commandLine(hook),
		},
		"timeout": timeout,
		"enabled": true,
	}
}

func isManagedKiroEntry(entry any, marker string) bool {
	value, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	name, _ := value["name"].(string)
	return strings.HasPrefix(name, marker+"-")
}

func pathIfChanged(changed bool, path string) []string {
	if changed {
		return []string{path}
	}
	return nil
}

func warningSlice(warning string) []string {
	if warning == "" {
		return nil
	}
	return []string{warning}
}
