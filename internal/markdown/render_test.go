package markdown

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	input := []byte("# Hello\n\nThis is **bold** and `code`.")
	html, err := Render(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := string(html)
	if !strings.Contains(result, "<h1>Hello</h1>") {
		t.Errorf("missing h1: %s", result)
	}
	if !strings.Contains(result, "<strong>bold</strong>") {
		t.Errorf("missing bold: %s", result)
	}
	if !strings.Contains(result, "<code>code</code>") {
		t.Errorf("missing code: %s", result)
	}
}

func TestRenderCodeBlock(t *testing.T) {
	input := []byte("```go\nfmt.Println(\"hello\")\n```")
	html, err := Render(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := string(html)
	if !strings.Contains(result, "<pre") {
		t.Errorf("missing pre block: %s", result)
	}
}

func TestExtractTitle(t *testing.T) {
	input := []byte("# My Great Document\n\nSome content.")
	title := ExtractTitle(input)
	if title != "My Great Document" {
		t.Errorf("expected 'My Great Document', got %q", title)
	}
}

func TestExtractTitleNoHeading(t *testing.T) {
	input := []byte("No heading here, just text.")
	title := ExtractTitle(input)
	if title != "" {
		t.Errorf("expected empty title, got %q", title)
	}
}
