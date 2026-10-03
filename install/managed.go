package install

import "strings"

// managedMarkers returns the HTML comment markers that delimit a managed
// guidance section for the given prefix, for example "backstage" gives
// "<!-- backstage-guidance:begin -->" and "<!-- backstage-guidance:end -->".
func managedMarkers(marker string) (begin, end string) {
	return "<!-- " + marker + "-guidance:begin -->", "<!-- " + marker + "-guidance:end -->"
}

func upsertManaged(existing, body []byte, beginMark, endMark string) (next []byte, changed bool) {
	block := managedBlock(body, beginMark, endMark)
	start, end, ok := markedSpan(existing, beginMark, endMark)
	if !ok {
		if len(strings.TrimSpace(string(existing))) == 0 {
			return []byte(block), true
		}
		s := string(existing)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		return []byte(s + "\n" + block), true
	}
	updated := string(existing[:start]) + block + string(existing[end:])
	if updated == string(existing) {
		return existing, false
	}
	return []byte(updated), true
}

func removeManaged(existing []byte, beginMark, endMark string) (next []byte, found bool) {
	return removeMarked(existing, beginMark, endMark)
}

func removeMarked(existing []byte, begin, endMark string) (next []byte, found bool) {
	start, end, ok := markedSpan(existing, begin, endMark)
	if !ok {
		return existing, false
	}
	for start > 0 && existing[start-1] == '\n' {
		start--
		if start > 0 && existing[start-1] == '\n' {
			break
		}
	}
	out := strings.TrimSpace(string(existing[:start]) + string(existing[end:]))
	if out == "" {
		return nil, true
	}
	return []byte(out + "\n"), true
}

func managedBlock(body []byte, beginMark, endMark string) string {
	text := strings.TrimRight(string(body), "\n")
	return beginMark + "\n" + text + "\n" + endMark + "\n"
}

func markedSpan(existing []byte, begin, endMark string) (start, end int, ok bool) {
	s := string(existing)
	start = strings.Index(s, begin)
	endAt := strings.Index(s, endMark)
	if start < 0 || endAt < start {
		return 0, 0, false
	}
	end = endAt + len(endMark)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	return start, end, true
}
