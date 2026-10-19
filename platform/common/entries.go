package common

// This file holds the entry shapes shared by the host adapters. A host that
// needs a different shape starts from one of these helpers and adds or renames
// the fields it documents, so the server identity still travels one way.

// ManagedServer builds the plain stdio entry shared by the JSON hosts: a
// command plus its arguments. Hosts whose shape differs build their own from
// the same Options.
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

// HeadersMap converts the caller's headers into the map type JSON and TOML
// writers expect. It returns nil when there are no headers, so an entry stays
// free of an empty object.
func HeadersMap(opts Options) map[string]any {
	if len(opts.Headers) == 0 {
		return nil
	}
	headers := make(map[string]any, len(opts.Headers))
	for name, value := range opts.Headers {
		headers[name] = value
	}
	return headers
}

// ManagedHTTP builds the plain remote entry: a URL under urlKey plus optional
// headers. Hosts that name the URL differently (Gemini's httpUrl, Antigravity's
// serverUrl) pass their key.
func ManagedHTTP(opts Options, urlKey string) map[string]any {
	entry := map[string]any{urlKey: opts.URL}
	if headers := HeadersMap(opts); headers != nil {
		entry["headers"] = headers
	}
	return entry
}

// AddType sets the transport discriminator a typed host requires.
func AddType(entry map[string]any, kind string) map[string]any {
	entry["type"] = kind
	return entry
}

// AddEnabled marks an entry enabled where the host defaults to disabled.
func AddEnabled(entry map[string]any) map[string]any {
	entry["enabled"] = true
	return entry
}
