package rag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	queryretriever "gorag/internal/retriever"
)

func TestContextBuilderStableNumberingLimitAndReverseMapping(t *testing.T) {
	builder, err := NewContextBuilder(2)
	if err != nil {
		t.Fatal(err)
	}
	low := retrievalDocument("low", "z.md", 0, 0.6)
	highSecond := retrievalDocument("high-second", "b.md", 1, 0.9)
	highFirst := retrievalDocument("high-first", "a.md", 2, 0.9)

	built, err := builder.Build(context.Background(), []*schema.Document{low, highSecond, highFirst})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(built.Sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(built.Sources))
	}
	if built.Sources[0].ID != "S1" || built.Sources[0].SourcePath != "a.md" {
		t.Fatalf("Sources[0] = %#v, want S1/a.md", built.Sources[0])
	}
	if built.Sources[1].ID != "S2" || built.Sources[1].SourcePath != "b.md" {
		t.Fatalf("Sources[1] = %#v, want S2/b.md", built.Sources[1])
	}
	if built.DocumentsBySource["S1"] != highFirst || built.DocumentsBySource["S2"] != highSecond {
		t.Fatal("DocumentsBySource does not preserve the exact retrieval documents")
	}
	if !strings.Contains(built.Text, "[S1]\npath: a.md") || !strings.Contains(built.Text, "[S2]\npath: b.md") {
		t.Fatalf("context text lacks stable source blocks:\n%s", built.Text)
	}
}

func TestContextBuilderEmptyInvalidAndCancellation(t *testing.T) {
	builder, err := NewContextBuilder(1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = builder.Build(context.Background(), nil)
	if !errors.Is(err, ErrInsufficientContext) {
		t.Fatalf("empty Build() error = %v", err)
	}

	invalid := retrievalDocument("bad", "bad.md", 0, 0.8)
	delete(invalid.MetaData, queryretriever.MetadataStartLine)
	_, err = builder.Build(context.Background(), []*schema.Document{invalid})
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("invalid Build() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = builder.Build(ctx, []*schema.Document{retrievalDocument("one", "one.md", 0, 0.8)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Build() error = %v", err)
	}
}

func retrievalDocument(id, sourcePath string, chunkIndex int, score float64) *schema.Document {
	document := &schema.Document{
		ID:      id,
		Content: "content for " + id,
		MetaData: map[string]any{
			queryretriever.MetadataDocumentID:      "1",
			queryretriever.MetadataSourcePath:      sourcePath,
			queryretriever.MetadataDocumentTitle:   "Title",
			queryretriever.MetadataHeadingPath:     []string{"Title", "Section"},
			queryretriever.MetadataStartLine:       2,
			queryretriever.MetadataEndLine:         5,
			queryretriever.MetadataChunkIndex:      chunkIndex,
			queryretriever.MetadataDocumentVersion: "v1",
		},
	}
	return document.WithScore(score)
}
