package retriever

import (
	"context"
	"errors"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type stubRetriever struct {
	documents []*schema.Document
	err       error
}

func (s *stubRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.documents, nil
}

func einoDocument(id string, score float64) *schema.Document {
	document := &schema.Document{
		ID:      id,
		Content: "content of " + id,
		MetaData: map[string]any{
			MetadataSourcePath: id + ".md",
			MetadataChunkIndex: 0,
		},
	}
	return document.WithScore(score)
}

func TestFusionRetrieverFusesByReciprocalRank(t *testing.T) {
	primary := &stubRetriever{documents: []*schema.Document{
		einoDocument("1:v:0", 0.9), einoDocument("2:v:0", 0.8),
	}}
	secondary := &stubRetriever{documents: []*schema.Document{
		einoDocument("2:v:0", 4.0), einoDocument("3:v:0", 3.0),
	}}
	fusion, err := NewFusionRetriever(3, DefaultFusionK, primary, secondary)
	if err != nil {
		t.Fatalf("NewFusionRetriever() error = %v", err)
	}

	documents, err := fusion.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 3 {
		t.Fatalf("len(documents) = %d, want 3", len(documents))
	}
	// 2:v:0 appears in both lists, so it must rank first under RRF.
	if documents[0].ID != "2:v:0" {
		t.Fatalf("first ID = %q, want 2:v:0", documents[0].ID)
	}
	expectedFirst := 1.0/(DefaultFusionK+2) + 1.0/(DefaultFusionK+1)
	if diff := documents[0].Score() - expectedFirst; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("fused score = %v, want %v", documents[0].Score(), expectedFirst)
	}
	if got := documents[0].MetaData[MetadataSimilarity]; got != documents[0].Score() {
		t.Fatalf("MetadataSimilarity = %v, want fused score %v", got, documents[0].Score())
	}
	if documents[1].ID != "1:v:0" || documents[2].ID != "3:v:0" {
		t.Fatalf("remaining order = %q, %q; want 1:v:0, 3:v:0", documents[1].ID, documents[2].ID)
	}
}

func TestFusionRetrieverRespectsCandidateLimit(t *testing.T) {
	primary := &stubRetriever{documents: []*schema.Document{
		einoDocument("1:v:0", 1), einoDocument("2:v:0", 0.9), einoDocument("3:v:0", 0.8),
	}}
	secondary := &stubRetriever{documents: []*schema.Document{
		einoDocument("4:v:0", 9.0),
	}}
	fusion, err := NewFusionRetriever(2, DefaultFusionK, primary, secondary)
	if err != nil {
		t.Fatalf("NewFusionRetriever() error = %v", err)
	}
	documents, err := fusion.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(documents) != 2 {
		t.Fatalf("len(documents) = %d, want 2", len(documents))
	}
	// 4:v:0 and 1:v:0 share the same RRF score (both rank 1 in one list);
	// the deterministic tie-break orders by document ID.
	if documents[0].ID != "1:v:0" || documents[1].ID != "4:v:0" {
		t.Fatalf("order = %q, %q; want 1:v:0, 4:v:0", documents[0].ID, documents[1].ID)
	}
}

func TestFusionRetrieverValidationAndErrors(t *testing.T) {
	if _, err := NewFusionRetriever(0, DefaultFusionK, &stubRetriever{}, &stubRetriever{}); err == nil {
		t.Fatal("zero topK should fail")
	}
	if _, err := NewFusionRetriever(5, 0, &stubRetriever{}, &stubRetriever{}); err == nil {
		t.Fatal("zero k should fail")
	}
	if _, err := NewFusionRetriever(5, DefaultFusionK, nil, &stubRetriever{}); err == nil {
		t.Fatal("nil retriever should fail")
	}
	if _, err := NewFusionRetriever(5, DefaultFusionK, &stubRetriever{}); err == nil {
		t.Fatal("a single retriever should fail")
	}
	fusion, err := NewFusionRetriever(5, DefaultFusionK, &stubRetriever{err: errors.New("boom")}, &stubRetriever{})
	if err != nil {
		t.Fatalf("NewFusionRetriever() error = %v", err)
	}
	if _, err := fusion.Retrieve(context.Background(), "query"); err == nil {
		t.Fatal("sub-retriever failure should propagate")
	}
}
