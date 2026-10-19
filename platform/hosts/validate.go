package hosts

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform/common"
)

// validateKiroURL mirrors Kiro CLI's own rule: a remote server needs HTTPS,
// and plain HTTP is accepted only on loopback. Rejecting it here keeps a
// registration the host would silently ignore out of its config.
func validateKiroURL(opts common.Options) error {
	if !common.IsHTTP(opts) {
		return nil
	}
	parsed, err := url.Parse(opts.URL)
	if err != nil {
		return fmt.Errorf("invalid Kiro MCP URL: %w", err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := strings.Trim(parsed.Hostname(), "[]")
	if parsed.Scheme == "http" && (strings.EqualFold(host, "localhost") || isLoopback(host)) {
		return nil
	}
	return errors.New("the Kiro CLI requires HTTPS for remote MCP servers (plain HTTP is accepted only on localhost)")
}

func isLoopback(host string) bool {
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

// validatePrimeAgentName mirrors Prime Agent's own lookup rule: a user server is
// connected only when its name is 1-64 characters of [A-Za-z0-9_-] starting
// with a letter or digit.
func validatePrimeAgentName(opts common.Options) error {
	name := opts.ServerName
	if len(name) == 0 || len(name) > 64 || !isAlphanumeric(name[0]) {
		return fmt.Errorf("server names for Prime Agent must be 1-64 characters of [A-Za-z0-9_-] starting with a letter or digit; got %q", name)
	}
	for index := 0; index < len(name); index++ {
		if !isAlphanumeric(name[index]) && name[index] != '_' && name[index] != '-' {
			return fmt.Errorf("server names for Prime Agent must be 1-64 characters of [A-Za-z0-9_-] starting with a letter or digit; got %q", name)
		}
	}
	return nil
}

func isAlphanumeric(char byte) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9')
}
