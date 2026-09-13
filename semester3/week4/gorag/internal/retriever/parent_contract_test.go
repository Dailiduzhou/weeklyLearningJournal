package retriever_test

import (
	"context"
	"strings"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/rag"
	"gorag/internal/repository"
	"gorag/internal/retriever"
)

// contractParentStore implements retriever.ParentStore with in-memory rows.
type contractParentStore struct {
	parents map[int64]repository.ParentDocument
}

func (s *contractParentStore) GetParentDocuments(ctx context.Context, ids []int64) ([]repository.ParentDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parents := make([]repository.ParentDocument, 0, len(ids))
	for _, id := range ids {
		if parent, exists := s.parents[id]; exists {
			parents = append(parents, parent)
		}
	}
	return parents, nil
}

type contractChildRetriever struct {
	documents []*schema.Document
}

func (r *contractChildRetriever) Retrieve(context.Context, string, ...einoretriever.Option) ([]*schema.Document, error) {
	return r.documents, nil
}

// TestParentDocumentsSatisfyContextBuilderContract guards the cross-package
// boundary: the documents a parent-expanded retrieval returns must carry the
// metadata the RAG context builder requires, including source-accurate lines
// and the best-child similarity.
func TestParentDocumentsSatisfyContextBuilderContract(t *testing.T) {
	ctx := context.Background()
	store := &contractParentStore{parents: map[int64]repository.ParentDocument{
		17: {DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication", Version: "v1", Content: "# Authentication\n\nWhole cleaned document body.", StartLine: 5, EndLine: 120},
	}}
	childChunk := &schema.Document{
		ID:      "17:v1:3",
		Content: "single embedded chunk",
		MetaData: map[string]any{
			retriever.MetadataDocumentID: "17",
			retriever.MetadataChunkIndex: 3,
		},
	}
	childChunk = childChunk.WithScore(0.86)
	child := &contractChildRetriever{documents: []*schema.Document{childChunk}}
	parentRetriever, err := retriever.NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	documents, err := parentRetriever.Retrieve(ctx, "how does authentication work?")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("Retrieve() returned %d documents, want the single parent", len(documents))
	}

	builder, err := rag.NewContextBuilder(5)
	if err != nil {
		t.Fatalf("NewContextBuilder() error = %v", err)
	}
	built, err := builder.Build(ctx, documents)
	if err != nil {
		t.Fatalf("Build() error = %v, want parent documents to satisfy the context contract", err)
	}
	if len(built.Sources) != 1 {
		t.Fatalf("built sources = %d, want 1", len(built.Sources))
	}
	source := built.Sources[0]
	if source.DocumentID != "17" || source.SourcePath != "api/auth.md" ||
		source.DocumentTitle != "Authentication" || source.DocumentVersion != "v1" ||
		source.StartLine != 5 || source.EndLine != 120 || source.Similarity != 0.86 {
		t.Fatalf("source = %#v, want whole-document citation with best-child similarity", source)
	}
	if !strings.Contains(built.Text, "Whole cleaned document body.") {
		t.Fatalf("built text does not contain the parent content: %q", built.Text)
	}
	if built.DocumentsBySource["S1"] != documents[0] {
		t.Fatal("source mapping does not point back at the parent document")
	}
}
