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
