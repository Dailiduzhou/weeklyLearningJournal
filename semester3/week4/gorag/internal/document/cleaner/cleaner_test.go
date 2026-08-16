package cleaner

import (
	"reflect"
	"strings"
	"testing"
)

func TestCleanNormalizesBOMNewlinesWhitespaceAndComments(t *testing.T) {
	input := "\ufeff---\r\ntitle: Example\r\ntags: docs\r\n---\r\n\r\n# Heading\r\n<!-- remove me -->\r\n\r\n\r\n- item\r\n\r<table>\r\n| A | B |\r\n| - | - |\r\n\r\n```html\r\n<!-- keep inside code -->\r\n```\r\n"
	result := Clean(input, Options{ParseFrontMatter: true})

	if result.FrontMatter["title"] != "Example" || result.FrontMatter["tags"] != "docs" {
		t.Fatalf("front matter = %#v", result.FrontMatter)
	}
	if strings.Contains(result.Content, "\r") || strings.HasPrefix(result.Content, "\ufeff") {
		t.Fatalf("content was not normalized: %q", result.Content)
	}
	if strings.Contains(result.Content, "remove me") {
		t.Errorf("HTML comment was not removed: %q", result.Content)
	}
	if !strings.Contains(result.Content, "<!-- keep inside code -->") {
		t.Errorf("comment inside fenced code was removed: %q", result.Content)
	}
	if strings.Contains(result.Content, "\n\n\n") {
		t.Errorf("excess blank lines remain: %q", result.Content)
	}
	for _, semantic := range []string{"# Heading", "- item", "| A | B |", "```html"} {
		if !strings.Contains(result.Content, semantic) {
			t.Errorf("semantic Markdown %q was not preserved", semantic)
		}
	}
}

func TestCleanReportsOriginalLineNumbers(t *testing.T) {
	input := "---\ntitle: Example\n---\n\n# Heading\n\n<!-- remove me -->\nbody"
	result := Clean(input, Options{ParseFrontMatter: true})

	want := []int{5, 6, 8}
	if !reflect.DeepEqual(result.LineNumbers, want) {
		t.Fatalf("LineNumbers = %v, want %v", result.LineNumbers, want)
	}
	if !strings.Contains(result.Content, "# Heading") || !strings.Contains(result.Content, "body") {
		t.Fatalf("Content = %q", result.Content)
	}
}

func TestCleanCanRetainFrontMatter(t *testing.T) {
	input := "---\ntitle: Keep\n---\nBody"
	result := Clean(input, Options{})
	if result.Content != input {
		t.Fatalf("Clean() content = %q, want unchanged front matter", result.Content)
	}
	if result.FrontMatter != nil {
		t.Fatalf("Clean() metadata = %#v, want nil", result.FrontMatter)
	}
}

func TestCleanHandlesMultilineAndInlineComments(t *testing.T) {
	input := "before <!-- first\nstill comment --> after <!-- second --> end"
	result := Clean(input, Options{})
	if result.Content != "before \n after  end" {
		t.Fatalf("Clean() = %q", result.Content)
	}
}
