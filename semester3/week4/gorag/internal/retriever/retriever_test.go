package retriever

import (
	"context"
	"errors"
	"math"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"gorag/internal/document"
	"gorag/internal/embedding"
	"gorag/internal/repository"
)

func TestPgVectorRetrieverFixedVectorsSortTopKThresholdAndMetadata(t *testing.T) {
	query := fixedVector(1, 0)
	store := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("c.md", 0), vector: fixedVector(0.6, 0.8)},
		{result: searchResult("d.md", 0), vector: fixedVector(0.4, math.Sqrt(0.84))},
		{result: searchResult("a.md", 1), vector: fixedVector(1, 0)},
		{result: searchResult("b.md", 0), vector: fixedVector(0.8, 0.6)},
	}}
	retriever, err := NewPgVectorRetriever(staticEmbedder{vector: query}, store, Config{
		CandidateTopK:       4,
		SimilarityThreshold: 0.5,
	})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}

	documents, err := retriever.Retrieve(context.Background(), "fixed query",
		einoretriever.WithTopK(2), einoretriever.WithScoreThreshold(0.7))
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if store.limit != 2 {
		t.Fatalf("Search limit = %d, want 2", store.limit)
	}
	if len(documents) != 2 {
		t.Fatalf("len(documents) = %d, want 2", len(documents))
	}
	if got := documents[0].MetaData[MetadataSourcePath]; got != "a.md" {
		t.Fatalf("first source = %v, want a.md", got)
	}
	if got := documents[1].MetaData[MetadataSourcePath]; got != "b.md" {
		t.Fatalf("second source = %v, want b.md", got)
	}
	if documents[0].Score() != 1 || math.Abs(documents[1].Score()-0.8) > 1e-6 {
		t.Fatalf("scores = %v, %v; want 1, 0.8", documents[0].Score(), documents[1].Score())
	}
	metadata := documents[0].MetaData
	for _, key := range []string{
		MetadataDocumentID, MetadataSourcePath, MetadataDocumentTitle,
		MetadataHeadingPath, MetadataStartLine, MetadataEndLine,
		MetadataChunkIndex, MetadataContentHash, MetadataDocumentVersion,
		MetadataEmbeddingModel, MetadataEmbeddingDimension, MetadataSimilarity,
	} {
		if _, ok := metadata[key]; !ok {
			t.Errorf("metadata %q is missing", key)
		}
	}
}

func TestPgVectorRetrieverTieBreakIsStable(t *testing.T) {
	store := &cosineStore{candidates: []vectorCandidate{
		{result: searchResult("z.md", 0), vector: fixedVector(1, 0)},
		{result: searchResult("a.md", 2), vector: fixedVector(1, 0)},
		{result: searchResult("a.md", 1), vector: fixedVector(1, 0)},
	}}
	retriever, err := NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, store, Config{
		CandidateTopK: 3, SimilarityThreshold: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	documents, err := retriever.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.md:v1:1", "a.md:v1:2", "z.md:v1:0"}
	for i, document := range documents {
		got := document.MetaData[MetadataSourcePath].(string) + ":" + document.MetaData[MetadataDocumentVersion].(string) + ":" + string(rune('0'+document.MetaData[MetadataChunkIndex].(int)))
		if got != want[i] {
			t.Fatalf("document %d identity = %q, want %q", i, got, want[i])
		}
	}
}

func TestPgVectorRetrieverErrorsAndCancellation(t *testing.T) {
	embedFailure := errors.New("ollama unavailable")
	retriever, err := NewPgVectorRetriever(staticEmbedder{err: embedFailure}, &cosineStore{}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = retriever.Retrieve(context.Background(), "query")
	if !errors.Is(err, ErrEmbedding) || !errors.Is(err, embedFailure) {
		t.Fatalf("embedding error = %v", err)
	}

	searchFailure := errors.New("database unavailable")
	retriever, err = NewPgVectorRetriever(staticEmbedder{vector: fixedVector(1, 0)}, &cosineStore{err: searchFailure}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = retriever.Retrieve(context.Background(), "query")
	if !errors.Is(err, ErrSearch) || !errors.Is(err, searchFailure) {
		t.Fatalf("search error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = retriever.Retrieve(ctx, "query")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

type staticEmbedder struct {
	vector []float32
	err    error
}

func (e staticEmbedder) EmbedQuery(ctx context.Context, _ string) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return e.vector, e.err
}

type vectorCandidate struct {
	result repository.SearchResult
	vector []float32
}

type cosineStore struct {
	candidates []vectorCandidate
	limit      int
	err        error
}

func (s *cosineStore) Search(ctx context.Context, query []float32, limit int) ([]repository.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.limit = limit
	if s.err != nil {
		return nil, s.err
	}
	results := make([]repository.SearchResult, 0, len(s.candidates))
	for _, candidate := range s.candidates {
		result := candidate.result
		result.Similarity = cosine(query, candidate.vector)
		results = append(results, result)
	}
	return results, nil
}

func cosine(left, right []float32) float64 {
	var dot, leftNorm, rightNorm float64
	for i := range left {
		dot += float64(left[i] * right[i])
		leftNorm += float64(left[i] * left[i])
		rightNorm += float64(right[i] * right[i])
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func fixedVector(first, second float64) []float32 {
	vector := make([]float32, embedding.VectorDimension)
	vector[0] = float32(first)
	vector[1] = float32(second)
	return vector
}

func searchResult(sourcePath string, chunkIndex int) repository.SearchResult {
	return repository.SearchResult{
		DocumentID: 1,
		Chunk: document.Chunk{
			DocumentID: "1", SourcePath: sourcePath, DocumentTitle: "Title",
			HeadingPath: []string{"Title", "Section"}, Index: chunkIndex,
			Content: "body", StartLine: 2, EndLine: 4, ContentHash: "hash",
			DocumentVersion: "v1", EmbeddingModel: embedding.DefaultModel,
			EmbeddingDimension: embedding.VectorDimension,
		},
	}
}
