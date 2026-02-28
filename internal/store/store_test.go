package store

import (
	"os"
	"testing"
)

func testDB(t *testing.T) *Store {
	t.Helper()
	f, err := os.CreateTemp("", "markcloud-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	s, err := New(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestUpsertAndGet(t *testing.T) {
	s := testDB(t)

	doc := &Document{
		Path:        "notes/test.md",
		Title:       "Test Doc",
		ContentMD:   "# Test\n\nHello.",
		ContentHTML: "<h1>Test</h1><p>Hello.</p>",
		SHA:         "abc123",
		Public:      false,
		Tags:        []string{"go", "test"},
	}

	err := s.Upsert(doc)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	got, err := s.Get("notes/test.md")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Title != "Test Doc" {
		t.Errorf("expected title 'Test Doc', got %q", got.Title)
	}
	if got.SHA != "abc123" {
		t.Errorf("expected sha 'abc123', got %q", got.SHA)
	}
	if len(got.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(got.Tags))
	}
}

func TestList(t *testing.T) {
	s := testDB(t)

	s.Upsert(&Document{Path: "notes/a.md", Title: "A", SHA: "1"})
	s.Upsert(&Document{Path: "notes/b.md", Title: "B", SHA: "2"})
	s.Upsert(&Document{Path: "other/c.md", Title: "C", SHA: "3"})

	docs, err := s.List("notes/")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs, got %d", len(docs))
	}
}

func TestDelete(t *testing.T) {
	s := testDB(t)

	s.Upsert(&Document{Path: "notes/del.md", Title: "Del", SHA: "1"})
	err := s.Delete("notes/del.md")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	got, err := s.Get("notes/del.md")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestSearch(t *testing.T) {
	s := testDB(t)

	s.Upsert(&Document{
		Path:      "notes/docker.md",
		Title:     "Docker Tips",
		ContentMD: "Configure docker networking with bridge mode.",
		SHA:       "1",
	})
	s.Upsert(&Document{
		Path:      "notes/go.md",
		Title:     "Go Notes",
		ContentMD: "Go is a programming language.",
		SHA:       "2",
	})

	results, err := s.Search("docker networking")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "notes/docker.md" {
		t.Errorf("expected docker doc, got %s", results[0].Path)
	}
}

func TestGetAllSHAs(t *testing.T) {
	s := testDB(t)

	s.Upsert(&Document{Path: "a.md", SHA: "sha1"})
	s.Upsert(&Document{Path: "b.md", SHA: "sha2"})

	shas, err := s.GetAllSHAs()
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(shas) != 2 {
		t.Errorf("expected 2, got %d", len(shas))
	}
	if shas["a.md"] != "sha1" {
		t.Errorf("expected sha1 for a.md")
	}
}
