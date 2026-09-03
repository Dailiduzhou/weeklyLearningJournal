package bm25

import (
	"context"
	"testing"

	"gorag/internal/document"
)

func testChunks(docID, version string) []document.Chunk {
	return []document.Chunk{
		{
			DocumentID: docID, SourcePath: "api/auth.md", DocumentTitle: "Authentication",
			HeadingPath: []string{"API", "Authentication"}, Index: 0,
			Content:   "JWT tokens are signed with HMAC and expire after one hour.",
			StartLine: 1, EndLine: 12, ContentHash: "hash-1", DocumentVersion: version,
		},
		{
			DocumentID: docID, SourcePath: "api/auth.md", DocumentTitle: "Authentication",
			HeadingPath: []string{"API", "Sessions"}, Index: 1,
			Content:   "Sessions are stored in PostgreSQL and refreshed on activity.",
			StartLine: 13, EndLine: 30, ContentHash: "hash-1", DocumentVersion: version,
		},
	}
}

func openTestIndex(t *testing.T) *Index {
	t.Helper()
	index, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = index.Close() })
	return index
}

func TestIndexChunksAndSearchReturnStoredMetadata(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	chunks := testChunks("7", "version-a")
	if err := index.IndexChunks(ctx, "7", "version-a", chunks); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	results, err := index.Search(ctx, "JWT tokens expire", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search() returned no results for a term present in the index")
	}
	found := false
	for _, result := range results {
		if result.Index != 0 {
			continue
		}
		found = true
		if result.DocumentID != 7 || result.Score <= 0 {
			t.Fatalf("unexpected hit: %#v", result)
		}
		if result.SourcePath != "api/auth.md" || result.DocumentTitle != "Authentication" ||
			result.DocumentVersion != "version-a" || result.ContentHash != "hash-1" {
			t.Fatalf("metadata roundtrip failed: %#v", result)
		}
		if len(result.HeadingPath) != 2 || result.HeadingPath[0] != "API" {
			t.Fatalf("heading path roundtrip failed: %#v", result.HeadingPath)
		}
		if result.StartLine != 1 || result.EndLine != 12 || result.Chunk.Content == "" {
			t.Fatalf("location/content roundtrip failed: %#v", result)
		}
	}
	if !found {
		t.Fatalf("expected chunk 0 in results: %#v", results)
	}
}

func TestIndexChunksReplacesPreviousVersion(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	if err := index.IndexChunks(ctx, "7", "version-a", testChunks("7", "version-a")); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	replacement := []document.Chunk{{
		DocumentID: "7", SourcePath: "api/auth.md", DocumentTitle: "Authentication",
		HeadingPath: []string{"API"}, Index: 0,
		Content:   "OAuth device flow is now the recommended flow.",
		StartLine: 1, EndLine: 5, ContentHash: "hash-2", DocumentVersion: "version-b",
	}}
	if err := index.IndexChunks(ctx, "7", "version-b", replacement); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	for _, query := range []string{"JWT tokens expire", "OAuth device flow"} {
		results, err := index.Search(ctx, query, 10)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", query, err)
		}
		for _, result := range results {
			if result.DocumentVersion != "version-b" {
				t.Fatalf("query %q returned stale version %q", query, result.DocumentVersion)
			}
		}
	}
}

func TestDeleteDocumentRemovesAllChunks(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	if err := index.IndexChunks(ctx, "7", "version-a", testChunks("7", "version-a")); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}
	if err := index.IndexChunks(ctx, "8", "version-a", testChunks("8", "version-a")); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	if err := index.DeleteDocument(ctx, "7"); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	results, err := index.Search(ctx, "JWT tokens expire", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	for _, result := range results {
		if result.DocumentID == 7 {
			t.Fatalf("document 7 was not deleted: %#v", results)
		}
	}
}

func TestSearchRejectsInvalidArguments(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	if _, err := index.Search(ctx, "  ", 5); err == nil {
		t.Fatal("Search() with a blank query should fail")
	}
	if _, err := index.Search(ctx, "query", 0); err == nil {
		t.Fatal("Search() with topK 0 should fail")
	}
	if err := index.IndexChunks(ctx, "7", "version-a", nil); err == nil {
		t.Fatal("IndexChunks() with no chunks should fail")
	}
	mismatched := testChunks("9", "version-x")
	if err := index.IndexChunks(ctx, "7", "version-a", mismatched); err == nil {
		t.Fatal("IndexChunks() with mismatched ID/version should fail")
	}
}
