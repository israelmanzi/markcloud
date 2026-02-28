package frontmatter

import (
	"testing"
)

func TestParse(t *testing.T) {
	input := `---
tags: [rust, learning]
public: true
---

# My Document

Some content here.`

	meta, body, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Public != true {
		t.Errorf("expected public=true, got %v", meta.Public)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "rust" || meta.Tags[1] != "learning" {
		t.Errorf("unexpected tags: %v", meta.Tags)
	}
	if string(body) != "# My Document\n\nSome content here." {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	input := `# Just a document

No frontmatter here.`

	meta, body, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Public != false {
		t.Errorf("expected public=false, got %v", meta.Public)
	}
	if len(meta.Tags) != 0 {
		t.Errorf("expected no tags, got %v", meta.Tags)
	}
	if string(body) != input {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestSerialize(t *testing.T) {
	meta := &Metadata{
		Tags:   []string{"go", "testing"},
		Public: false,
	}
	body := []byte("# Test Doc\n\nContent.")

	result := Serialize(meta, body)

	parsed, parsedBody, err := Parse(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Public != false {
		t.Errorf("expected public=false")
	}
	if len(parsed.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(parsed.Tags))
	}
	if string(parsedBody) != "# Test Doc\n\nContent." {
		t.Errorf("unexpected body: %q", string(parsedBody))
	}
}
