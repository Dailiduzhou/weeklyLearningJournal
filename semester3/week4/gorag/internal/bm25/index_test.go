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
	if err := index.IndexChunks(ctx, "7", "version-a", chunks, document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	results, err := index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{})
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
	if err := index.IndexChunks(ctx, "7", "version-a", testChunks("7", "version-a"), document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	replacement := []document.Chunk{{
		DocumentID: "7", SourcePath: "api/auth.md", DocumentTitle: "Authentication",
		HeadingPath: []string{"API"}, Index: 0,
		Content:   "OAuth device flow is now the recommended flow.",
		StartLine: 1, EndLine: 5, ContentHash: "hash-2", DocumentVersion: "version-b",
	}}
	if err := index.IndexChunks(ctx, "7", "version-b", replacement, document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	for _, query := range []string{"JWT tokens expire", "OAuth device flow"} {
		results, err := index.Search(ctx, query, 10, document.MetadataFilter{})
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
	if err := index.IndexChunks(ctx, "7", "version-a", testChunks("7", "version-a"), document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}
	if err := index.IndexChunks(ctx, "8", "version-a", testChunks("8", "version-a"), document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks() error = %v", err)
	}

	if err := index.DeleteDocument(ctx, "7"); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	results, err := index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{})
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
	if _, err := index.Search(ctx, "  ", 5, document.MetadataFilter{}); err == nil {
		t.Fatal("Search() with a blank query should fail")
	}
	if _, err := index.Search(ctx, "query", 0, document.MetadataFilter{}); err == nil {
		t.Fatal("Search() with topK 0 should fail")
	}
	if err := index.IndexChunks(ctx, "7", "version-a", nil, document.DocumentMetadata{}); err == nil {
		t.Fatal("IndexChunks() with no chunks should fail")
	}
	mismatched := testChunks("9", "version-x")
	if err := index.IndexChunks(ctx, "7", "version-a", mismatched, document.DocumentMetadata{}); err == nil {
		t.Fatal("IndexChunks() with mismatched ID/version should fail")
	}
}

func TestSearchAppliesMetadataFilter(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	defer index.Close()

	ccnuboxMeta := document.DocumentMetadata{
		Scalars: map[string]string{"category": "ccnubox", "module": "bff", "status": "done"},
		Lists:   map[string][]string{"tags": {"ccnubox", "ccnubox/bff"}},
	}
	otherMeta := document.DocumentMetadata{
		Scalars: map[string]string{"category": "notes", "module": "cli", "status": "draft"},
		Lists:   map[string][]string{"tags": {"notes", "notes/cli"}},
	}
	if err := index.IndexChunks(ctx, "7", "v1", testChunks("7", "v1"), ccnuboxMeta); err != nil {
		t.Fatalf("IndexChunks(ccnubox) error = %v", err)
	}
	if err := index.IndexChunks(ctx, "8", "v1", testChunks("8", "v1"), otherMeta); err != nil {
		t.Fatalf("IndexChunks(other) error = %v", err)
	}

	// A scalar filter narrows to matching documents.
	results, err := index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox"},
	})
	if err != nil {
		t.Fatalf("Search(scalar filter) error = %v", err)
	}
	for _, result := range results {
		if result.DocumentID != 7 {
			t.Fatalf("Search(scalar filter) returned document %d, want only 7", result.DocumentID)
		}
	}

	// Multiple scalar keys must all match.
	results, err = index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox", "status": "draft"},
	})
	if err != nil {
		t.Fatalf("Search(combined scalar filter) error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search(combined scalar filter) = %d results, want none", len(results))
	}

	// Tags use any-of overlap.
	results, err = index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Tags: []string{"notes/cli", "missing"},
	})
	if err != nil {
		t.Fatalf("Search(tag any-of) error = %v", err)
	}
	for _, result := range results {
		if result.DocumentID != 8 {
			t.Fatalf("Search(tag any-of) returned document %d, want only 8", result.DocumentID)
		}
	}

	// Scalar and tag constraints combine conjunctively.
	results, err = index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Metadata: map[string]string{"module": "bff"},
		Tags:     []string{"ccnubox/bff"},
	})
	if err != nil {
		t.Fatalf("Search(scalar and tags) error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("Search(scalar and tags) returned no results")
	}
	for _, result := range results {
		if result.DocumentID != 7 {
			t.Fatalf("Search(scalar and tags) returned document %d, want only 7", result.DocumentID)
		}
	}

	// Chunks indexed without filterable fields cannot match a filter.
	if err := index.IndexChunks(ctx, "9", "v1", testChunks("9", "v1"), document.DocumentMetadata{}); err != nil {
		t.Fatalf("IndexChunks(no metadata) error = %v", err)
	}
	results, err = index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Metadata: map[string]string{"status": "done"},
	})
	if err != nil {
		t.Fatalf("Search(after untagged insert) error = %v", err)
	}
	for _, result := range results {
		if result.DocumentID == 9 {
			t.Fatal("document without metadata terms matched a filter")
		}
	}

	// An invalid filter is rejected before the index is queried.
	if _, err := index.Search(ctx, "JWT tokens expire", 10, document.MetadataFilter{
		Metadata: map[string]string{"status": " "},
	}); err == nil {
		t.Fatal("Search(invalid filter) error = nil")
	}
}

func TestIndexChunksRejectsInvalidMetadata(t *testing.T) {
	ctx := context.Background()
	index := openTestIndex(t)
	defer index.Close()

	invalid := document.DocumentMetadata{Lists: map[string][]string{"tags": {" "}}}
	if err := index.IndexChunks(ctx, "7", "v1", testChunks("7", "v1"), invalid); err == nil {
		t.Fatal("IndexChunks(invalid metadata) error = nil")
	}
}
