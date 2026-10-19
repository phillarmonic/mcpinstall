package common

import (
	"fmt"
	"strings"
)

// Transport names accepted by Options.Transport.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// IsHTTP reports whether opts describes a remote HTTP entry. An unset transport
// follows the URL: a Spec that names a URL is remote, one that names only a
// command is stdio.
func IsHTTP(opts Options) bool {
	switch strings.ToLower(strings.TrimSpace(opts.Transport)) {
	case TransportHTTP, "remote", "streamable-http", "streamable_http":
		return true
	case TransportStdio, "local":
		return false
	default:
		return strings.TrimSpace(opts.URL) != ""
	}
}

// Validate checks the fields an install needs. Uninstall ignores the server
// identity, so it never calls this.
func Validate(opts Options) error {
	if strings.TrimSpace(opts.ServerName) == "" {
		return fmt.Errorf("server name is required")
	}
	if IsHTTP(opts) {
		if strings.TrimSpace(opts.URL) == "" {
			return fmt.Errorf("http transport requires a URL")
		}
		return nil
	}
	if strings.TrimSpace(opts.Command) == "" {
		return fmt.Errorf("stdio transport requires a command")
	}
	return nil
}
