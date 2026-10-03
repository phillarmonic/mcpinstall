package common

// normalizeJSONC rewrites a JSONC document into the strict JSON that
// encoding/json accepts, removing // and /* */ comments and trailing commas.
//
// Several hosts ship their configuration as JSONC rather than JSON. OpenCode's
// schema sets allowComments and allowTrailingCommas, and VS Code's mcp.json
// carries commented example blocks, so a strict parse of an otherwise ordinary
// configuration would fail and the merge would degrade to a warning for the
// host's most common file shape.
//
// The scan is string-aware: comment markers and commas inside string literals —
// a URL's "//", a comma in a path — survive untouched.
//
// Reading is tolerant; writing is not. A configuration this package rewrites is
// written back as strict JSON, so a file's comments do not survive a merge. The
// alternative — refusing to merge any commented file — would leave those hosts
// unconfigurable, which is worse than a normalised rewrite.
func normalizeJSONC(data []byte) []byte {
	return removeTrailingCommas(stripComments(data))
}

// stripComments removes line and block comments from a JSONC document. A line
// comment keeps its newline so the surrounding tokens stay separated; a block
// comment is replaced by a space for the same reason.
func stripComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out = append(out, c)
		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out = append(out, data[i])
			}
		case c == '/' && i+1 < len(data) && data[i+1] == '*':
			i += 2
			for i+1 < len(data) && (data[i] != '*' || data[i+1] != '/') {
				i++
			}
			i++
			out = append(out, ' ')
		default:
			out = append(out, c)
		}
	}
	return out
}

// removeTrailingCommas drops commas that directly precede a closing brace or
// bracket, which JSONC permits and encoding/json rejects. It expects
// comment-free input and is string-aware.
func removeTrailingCommas(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false
	for i := range len(data) {
		c := data[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\n' || data[j] == '\r') {
				j++
			}
			if j < len(data) && (data[j] == '}' || data[j] == ']') {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
