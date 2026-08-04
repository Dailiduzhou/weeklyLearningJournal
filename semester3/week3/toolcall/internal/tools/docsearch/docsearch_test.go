package docsearch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchReturnsSourceAndNoResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "guide.md"), []byte("Contexts propagate cancellation through every model and tool call."), 0o600); err != nil {
		t.Fatal(err)
	}
	search, err := New(dir, 100, 5)
	if err != nil {
		t.Fatal(err)
	}
	result := search.Execute(context.Background(), json.RawMessage(`{"query":"context cancellation"}`))
	if !result.OK {
		t.Fatalf("search failed: %+v", result.Error)
	}
	data := result.Data.(map[string]any)
	matches := data["matches"].([]Match)
	if len(matches) != 1 || matches[0].Source != "guide.md" || matches[0].Score <= 0 {
		t.Fatalf("unexpected matches: %#v", matches)
	}

	empty := search.Execute(context.Background(), json.RawMessage(`{"query":"postgresql"}`))
	if !empty.OK || empty.Data.(map[string]any)["count"].(int) != 0 {
		t.Fatalf("unexpected empty result: %+v", empty)
	}
}

func TestMissingDocumentDirectoryIsEmptyIndex(t *testing.T) {
	search, err := New(filepath.Join(t.TempDir(), "missing"), 100, 5)
	if err != nil {
		t.Fatal(err)
	}
	result := search.Execute(context.Background(), json.RawMessage(`{"query":"anything"}`))
	if !result.OK || result.Data.(map[string]any)["count"].(int) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
