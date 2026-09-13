package retriever

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/repository"
)

type stubChildRetriever struct {
	documents []*schema.Document
	err       error
}

func (r *stubChildRetriever) Retrieve(context.Context, string, ...einoretriever.Option) ([]*schema.Document, error) {
	return r.documents, r.err
}

type stubParentStore struct {
	parents  map[int64]repository.ParentDocument
	requests [][]int64
	err      error
}

func (s *stubParentStore) GetParentDocuments(ctx context.Context, ids []int64) ([]repository.ParentDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.requests = append(s.requests, append([]int64(nil), ids...))
	if s.err != nil {
		return nil, s.err
	}
	parents := make([]repository.ParentDocument, 0, len(ids))
	for _, id := range ids {
		if parent, exists := s.parents[id]; exists {
			parents = append(parents, parent)
		}
	}
	return parents, nil
}

func childDocument(documentID string, index int, score float64) *schema.Document {
	document := &schema.Document{
		ID:      documentID + ":v1:" + strconv.Itoa(index),
		Content: fmt.Sprintf("chunk %d of document %s", index, documentID),
		MetaData: map[string]any{
			MetadataDocumentID: documentID,
			MetadataChunkIndex: index,
		},
	}
	return document.WithScore(score)
}

func TestParentDocumentRetrieverExpandsChildrenToParentDocuments(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{parents: map[int64]repository.ParentDocument{
		17: {DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication", Version: "v1", Content: "whole auth document", StartLine: 1, EndLine: 40},
		42: {DocumentID: 42, SourcePath: "guide.md", Title: "Guide", Version: "v2", Content: "whole guide document", StartLine: 3, EndLine: 90},
	}}
	child := &stubChildRetriever{documents: []*schema.Document{
		childDocument("17", 2, 0.8),
		childDocument("42", 0, 0.7),
		childDocument("17", 0, 0.9),
	}}
	retriever, err := NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	documents, err := retriever.Retrieve(ctx, "query")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("Retrieve() returned %d documents, want 2 parents", len(documents))
	}

	// Eino's parent flow retriever deduplicates by first appearance, so
	// parents arrive in child-rank order: 17 before 42.
	if documents[0].MetaData[MetadataDocumentID] != "17" || documents[1].MetaData[MetadataDocumentID] != "42" {
		t.Fatalf("parent order = [%v, %v], want [17, 42]",
			documents[0].MetaData[MetadataDocumentID], documents[1].MetaData[MetadataDocumentID])
	}
	// Each parent carries the best score among its children.
	if documents[0].Score() != 0.9 || documents[1].Score() != 0.7 {
		t.Fatalf("parent scores = [%v, %v], want [0.9, 0.7]", documents[0].Score(), documents[1].Score())
	}

	first := documents[0]
	if first.ID != "17:v1:parent" {
		t.Fatalf("parent ID = %q, want %q", first.ID, "17:v1:parent")
	}
	if first.Content != "whole auth document" {
		t.Fatalf("parent content = %q", first.Content)
	}
	metadata := first.MetaData
	heading, headingOK := metadata[MetadataHeadingPath].([]string)
	if metadata[MetadataSourcePath] != "api/auth.md" ||
		metadata[MetadataDocumentTitle] != "Authentication" ||
		metadata[MetadataDocumentVersion] != "v1" ||
		!headingOK || len(heading) != 0 ||
		metadata[MetadataStartLine] != 1 ||
		metadata[MetadataEndLine] != 40 ||
		metadata[MetadataChunkIndex] != 0 ||
		metadata[MetadataSimilarity] != 0.9 {
		t.Fatalf("parent metadata = %#v", metadata)
	}

	if len(store.requests) != 1 || len(store.requests[0]) != 2 || store.requests[0][0] != 17 || store.requests[0][1] != 42 {
		t.Fatalf("parent store requests = %#v, want one ordered batch [17, 42]", store.requests)
	}
}

func TestParentDocumentRetrieverUsesFreshScoresPerRequest(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{parents: map[int64]repository.ParentDocument{
		17: {DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication", Version: "v1", Content: "whole auth document", StartLine: 1, EndLine: 40},
	}}
	child := &stubChildRetriever{documents: []*schema.Document{childDocument("17", 0, 0.9)}}
	retriever, err := NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	first, err := retriever.Retrieve(ctx, "query")
	if err != nil || len(first) != 1 || first[0].Score() != 0.9 {
		t.Fatalf("first Retrieve() = %#v, error %v", first, err)
	}
	child.documents = []*schema.Document{childDocument("17", 1, 0.4)}
	second, err := retriever.Retrieve(ctx, "query")
	if err != nil || len(second) != 1 || second[0].Score() != 0.4 {
		t.Fatalf("second Retrieve() = %#v, error %v, want parent score 0.4 without leakage from the previous request", second, err)
	}
}

func TestParentDocumentRetrieverReturnsEmptyWithoutChildren(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{parents: map[int64]repository.ParentDocument{}}
	retriever, err := NewParentDocumentRetriever(ctx, &stubChildRetriever{}, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	documents, err := retriever.Retrieve(ctx, "query")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 0 {
		t.Fatalf("Retrieve() = %#v, want no parents", documents)
	}
	if len(store.requests) != 0 {
		t.Fatalf("parent store was called %#v times for an empty child result set", store.requests)
	}
}

func TestParentDocumentRetrieverDropsParentsMissingFromStore(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{parents: map[int64]repository.ParentDocument{
		17: {DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication", Version: "v1", Content: "whole auth document", StartLine: 1, EndLine: 40},
	}}
	child := &stubChildRetriever{documents: []*schema.Document{
		childDocument("17", 0, 0.9),
		childDocument("42", 0, 0.7), // indexed before this feature: no parent row
	}}
	retriever, err := NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	documents, err := retriever.Retrieve(ctx, "query")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 1 || documents[0].MetaData[MetadataDocumentID] != "17" {
		t.Fatalf("Retrieve() = %#v, want only the stored parent 17", documents)
	}
}

func TestParentDocumentRetrieverPropagatesChildError(t *testing.T) {
	ctx := context.Background()
	childFailure := errors.New("child search failed")
	retriever, err := NewParentDocumentRetriever(ctx, &stubChildRetriever{err: childFailure}, &stubParentStore{})
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	if _, err := retriever.Retrieve(ctx, "query"); !errors.Is(err, childFailure) {
		t.Fatalf("Retrieve() error = %v, want child failure", err)
	}
}

func TestParentDocumentRetrieverPropagatesStoreError(t *testing.T) {
	ctx := context.Background()
	storeFailure := errors.New("database unavailable")
	store := &stubParentStore{err: storeFailure}
	child := &stubChildRetriever{documents: []*schema.Document{childDocument("17", 0, 0.9)}}
	retriever, err := NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	_, err = retriever.Retrieve(ctx, "query")
	if !errors.Is(err, storeFailure) || !errors.Is(err, ErrParentFetch) {
		t.Fatalf("Retrieve() error = %v, want wrapped ErrParentFetch", err)
	}
}

func TestParentDocumentRetrieverRejectsInvalidParentIDMetadata(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{parents: map[int64]repository.ParentDocument{}}
	broken := childDocument("not-a-number", 0, 0.9)
	child := &stubChildRetriever{documents: []*schema.Document{broken}}
	retriever, err := NewParentDocumentRetriever(ctx, child, store)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	if _, err := retriever.Retrieve(ctx, "query"); !errors.Is(err, ErrParentFetch) {
		t.Fatalf("Retrieve() error = %v, want ErrParentFetch for malformed metadata", err)
	}
}

func TestParentDocumentRetrieverRejectsCancelledContext(t *testing.T) {
	ctx := context.Background()
	retriever, err := NewParentDocumentRetriever(ctx, &stubChildRetriever{}, &stubParentStore{})
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()

	if _, err := retriever.Retrieve(cancelled, "query"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Retrieve(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestNewParentDocumentRetrieverValidatesDependencies(t *testing.T) {
	ctx := context.Background()
	store := &stubParentStore{}
	child := &stubChildRetriever{}
	if _, err := NewParentDocumentRetriever(ctx, nil, store); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewParentDocumentRetriever(nil child) error = %v, want ErrInvalidConfig", err)
	}
	if _, err := NewParentDocumentRetriever(ctx, child, nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewParentDocumentRetriever(nil store) error = %v, want ErrInvalidConfig", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := NewParentDocumentRetriever(cancelled, child, store); !errors.Is(err, context.Canceled) {
		t.Fatalf("NewParentDocumentRetriever(cancelled) error = %v, want context.Canceled", err)
	}
}
