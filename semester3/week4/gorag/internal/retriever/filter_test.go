package retriever

import (
	"context"
	"errors"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"gorag/internal/document"
	"gorag/internal/repository"
)

// TestWithFilterForwardsMetadataFilterToVectorSearch verifies that the Eino
// implementation-specific option reaches the vector store and that an empty
// filter changes nothing.
func TestWithFilterForwardsMetadataFilterToVectorSearch(t *testing.T) {
	ctx := context.Background()
	store := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("api/auth.md", 0), vector: fixedVector(1, 0)},
	}}
	retriever, err := NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, store, Config{})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}
	filter := document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox"},
		Tags:     []string{"ccnubox/bff"},
	}

	if _, err := retriever.Retrieve(ctx, "query"); err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if !store.filter.Empty() {
		t.Fatalf("store filter = %#v, want empty without options", store.filter)
	}

	if _, err := retriever.Retrieve(ctx, "query", WithFilter(filter)); err != nil {
		t.Fatalf("Retrieve(WithFilter) error = %v", err)
	}
	if len(store.filter.Metadata) != 1 || store.filter.Metadata["category"] != "ccnubox" ||
		len(store.filter.Tags) != 1 || store.filter.Tags[0] != "ccnubox/bff" {
		t.Fatalf("store filter = %#v, want the caller's filter", store.filter)
	}

	// An invalid filter is rejected at the retriever boundary.
	invalid := document.MetadataFilter{Metadata: map[string]string{"category": " "}}
	if _, err := retriever.Retrieve(ctx, "query", WithFilter(invalid)); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Retrieve(invalid filter) error = %v, want ErrInvalidConfig", err)
	}
}

// TestFilterOptionCoexistsWithStandardOptions guards the option plumbing: a
// filter option must not shadow, nor be shadowed by, Eino's common options on
// the same call.
func TestFilterOptionCoexistsWithStandardOptions(t *testing.T) {
	ctx := context.Background()
	store := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("a.md", 0), vector: fixedVector(1, 0)},
		{result: searchResult("b.md", 0), vector: fixedVector(0, 1)},
	}}
	retriever, err := NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, store, Config{CandidateTopK: 10, SimilarityThreshold: 0.5})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}

	filter := document.MetadataFilter{Tags: []string{"ccnubox"}}
	_, err = retriever.Retrieve(ctx, "query",
		einoretriever.WithTopK(1), WithFilter(filter), einoretriever.WithScoreThreshold(0.5))
	if err != nil {
		t.Fatalf("Retrieve(mixed options) error = %v", err)
	}
	if store.limit != 1 {
		t.Fatalf("Search limit = %d, want the WithTopK override", store.limit)
	}
	if len(store.filter.Tags) != 1 || store.filter.Tags[0] != "ccnubox" {
		t.Fatalf("store filter = %#v, want the filter option preserved alongside standard options", store.filter)
	}
}

// TestWithFilterForwardsMetadataFilterToBM25Search verifies the same for the
// lexical backend.
func TestWithFilterForwardsMetadataFilterToBM25Search(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeBM25Searcher{}
	retriever, err := NewBM25Retriever(searcher, Config{})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}
	filter := document.MetadataFilter{Tags: []string{"ccnubox/crypto"}}

	if _, err := retriever.Retrieve(ctx, "query"); err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if !searcher.filter.Empty() {
		t.Fatalf("searcher filter = %#v, want empty without options", searcher.filter)
	}
	if _, err := retriever.Retrieve(ctx, "query", WithFilter(filter)); err != nil {
		t.Fatalf("Retrieve(WithFilter) error = %v", err)
	}
	if len(searcher.filter.Tags) != 1 || searcher.filter.Tags[0] != "ccnubox/crypto" {
		t.Fatalf("searcher filter = %#v, want the caller's filter", searcher.filter)
	}
}

// TestFusionForwardsFilterOptions ensures hybrid retrieval preserves metadata
// filters by forwarding call options to both child retrievers verbatim.
func TestFusionForwardsFilterOptions(t *testing.T) {
	ctx := context.Background()
	vectorStore := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("a.md", 0), vector: fixedVector(1, 0)},
	}}
	vectorRetriever, err := NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, vectorStore, Config{})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}
	bm25Searcher := &fakeBM25Searcher{}
	bm25Retriever, err := NewBM25Retriever(bm25Searcher, Config{})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}
	fused, err := NewFusionRetriever(10, DefaultFusionK, vectorRetriever, bm25Retriever)
	if err != nil {
		t.Fatalf("NewFusionRetriever() error = %v", err)
	}

	filter := document.MetadataFilter{Metadata: map[string]string{"module": "bff"}}
	if _, err := fused.Retrieve(ctx, "query", WithFilter(filter)); err != nil {
		t.Fatalf("Retrieve(WithFilter) error = %v", err)
	}
	if vectorStore.filter.Metadata["module"] != "bff" {
		t.Fatalf("vector child filter = %#v, want forwarded filter", vectorStore.filter)
	}
	if bm25Searcher.filter.Metadata["module"] != "bff" {
		t.Fatalf("bm25 child filter = %#v, want forwarded filter", bm25Searcher.filter)
	}
}

// TestParentDocumentRetrieverForwardsFilterOptions ensures filters survive
// parent expansion: children are filtered at search time, so only matching
// documents can contribute parents.
func TestParentDocumentRetrieverForwardsFilterOptions(t *testing.T) {
	ctx := context.Background()
	store := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("api/auth.md", 0), vector: fixedVector(1, 0)},
	}}
	vectorRetriever, err := NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, store, Config{})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}
	parents := &stubParentStore{parents: map[int64]repository.ParentDocument{
		17: {DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication", Version: "v1", Content: "whole document", StartLine: 1, EndLine: 10},
	}}
	parentRetriever, err := NewParentDocumentRetriever(ctx, vectorRetriever, parents)
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	filter := document.MetadataFilter{Tags: []string{"ccnubox"}}
	if _, err := parentRetriever.Retrieve(ctx, "query", WithFilter(filter)); err != nil {
		t.Fatalf("Retrieve(WithFilter) error = %v", err)
	}
	if len(store.filter.Tags) != 1 || store.filter.Tags[0] != "ccnubox" {
		t.Fatalf("underlying child filter = %#v, want the forwarded filter", store.filter)
	}
}
