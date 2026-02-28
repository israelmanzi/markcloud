package sync

import (
	"os"
	"testing"

	"github.com/israelmanzi/markcloud/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	f, err := os.CreateTemp("", "markcloud-sync-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	s, err := store.New(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDiff(t *testing.T) {
	s := testStore(t)
	h := NewHandler(s)

	s.Upsert(&store.Document{Path: "old.md", SHA: "sha1"})
	s.Upsert(&store.Document{Path: "unchanged.md", SHA: "sha2"})

	manifest := []ManifestEntry{
		{Path: "unchanged.md", SHA: "sha2"},
		{Path: "old.md", SHA: "sha1-updated"},
		{Path: "new.md", SHA: "sha3"},
	}

	result, err := h.Diff(manifest)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}

	if len(result.NeedContent) != 2 {
		t.Errorf("expected 2 need_content, got %d: %v", len(result.NeedContent), result.NeedContent)
	}
}

func TestApply(t *testing.T) {
	s := testStore(t)
	h := NewHandler(s)

	s.Upsert(&store.Document{Path: "deleted.md", SHA: "old"})

	files := []FileEntry{
		{
			Path:    "notes/new.md",
			Content: "---\ntags: [go]\npublic: true\n---\n\n# New Doc\n\nHello.",
			SHA:     "sha1",
		},
	}

	allPaths := []string{"notes/new.md"}

	err := h.Apply(files, allPaths)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	doc, _ := s.Get("notes/new.md")
	if doc == nil {
		t.Fatal("expected new doc to exist")
	}
	if doc.Title != "New Doc" {
		t.Errorf("expected title 'New Doc', got %q", doc.Title)
	}
	if !doc.Public {
		t.Error("expected public=true")
	}

	old, _ := s.Get("deleted.md")
	if old != nil {
		t.Error("expected deleted doc to be removed")
	}
}
