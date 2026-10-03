package install

import (
	"strings"
	"testing"
)

func TestUpsertManagedCreatesAndReplacesTheOwnedSection(t *testing.T) {
	body := []byte("kit tools first\n")
	created, changed := upsertManaged(nil, body, testBegin, testEnd)
	if !changed {
		t.Fatal("expected a create")
	}
	if string(created) != managedBlock(body, testBegin, testEnd) {
		t.Fatalf("created = %q", created)
	}

	same, changed := upsertManaged(created, body, testBegin, testEnd)
	if changed {
		t.Fatalf("identical section reported a change: %q", same)
	}

	updated, changed := upsertManaged(created, []byte("new body"), testBegin, testEnd)
	if !changed {
		t.Fatal("expected a replace")
	}
	if !strings.Contains(string(updated), "new body") {
		t.Fatalf("updated = %q", updated)
	}
}

func TestUpsertManagedAppendsToExistingNotes(t *testing.T) {
	existing := []byte("# User notes\nKeep this.\n")
	next, changed := upsertManaged(existing, []byte("kit"), testBegin, testEnd)
	if !changed {
		t.Fatal("expected an append")
	}
	text := string(next)
	for _, want := range []string{"# User notes", "Keep this.", testBegin, "kit", testEnd} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
}

func TestRemoveManagedLeavesSurroundingText(t *testing.T) {
	existing, _ := upsertManaged([]byte("keep me\n"), []byte("kit"), testBegin, testEnd)
	next, found := removeManaged(existing, testBegin, testEnd)
	if !found {
		t.Fatal("expected to find the section")
	}
	if string(next) != "keep me\n" {
		t.Fatalf("next = %q", next)
	}

	_, found = removeManaged(next, testBegin, testEnd)
	if found {
		t.Fatal("second remove found a section")
	}
}
