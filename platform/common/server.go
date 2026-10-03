package common

// This file holds helpers shared by the host adapters for describing the MCP
// server entry. The server identity travels in Options, so the platform package
// stays free of any project-specific name.

// EntryName returns the server key an uninstall should delete. An empty
// override deletes the configured ServerName.
func EntryName(opts Options) string {
	if opts.EntryName != "" {
		return opts.EntryName
	}
	return opts.ServerName
}

// ManagedServer builds the plain MCP server entry shared by the JSON hosts: a
// command plus its arguments. Hosts whose shape differs (OpenCode) build their
// own from the same Options.
func ManagedServer(opts Options) map[string]any {
	args := make([]any, 0, len(opts.Args))
	for _, arg := range opts.Args {
		args = append(args, arg)
	}
	return map[string]any{
		"command": opts.Command,
		"args":    args,
	}
}
