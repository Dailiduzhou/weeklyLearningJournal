package retriever

import (
	"context"
	"errors"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"gorag/internal/bm25"
	"gorag/internal/document"
)

type fakeBM25Searcher struct {
	results []bm25.SearchResult
	err     error
	query   string
	topK    int
}

func (s *fakeBM25Searcher) Search(ctx context.Context, query string, topK int) ([]bm25.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.err != nil {
		return nil, s.err
	}
	s.query, s.topK = query, topK
	return append([]bm25.SearchResult(nil), s.results...), nil
}

func bm25Hit(docID int64, version string, index int, path, title, content string, score float64) bm25.SearchResult {
	return bm25.SearchResult{
		DocumentID: docID,
		Chunk: document.Chunk{
			DocumentID: itoa(docID), SourcePath: path, DocumentTitle: title,
			HeadingPath: []string{"Heading"}, Index: index, Content: content,
			StartLine: index + 1, EndLine: index + 10, ContentHash: "hash",
			DocumentVersion: version,
		},
		Score: score,
	}
}

func itoa(value int64) string {
	digits := ""
	if value == 0 {
		return "0"
	}
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

func TestBM25RetrieverAppliesTopKThresholdAndMetadata(t *testing.T) {
	searcher := &fakeBM25Searcher{results: []bm25.SearchResult{
		bm25Hit(2, "v1", 0, "b.md", "Title B", "content b", 3.5),
		bm25Hit(1, "v1", 1, "a.md", "Title A", "content a", 7.25),
		bm25Hit(3, "v1", 0, "c.md", "Title C", "content c", 0.1),
	}}
	retriever, err := NewBM25Retriever(searcher, Config{CandidateTopK: 3, SimilarityThreshold: 0})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}

	documents, err := retriever.Retrieve(context.Background(), "lexical query",
		einoretriever.WithTopK(2), einoretriever.WithScoreThreshold(1.0))
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if searcher.topK != 2 {
		t.Fatalf("search topK = %d, want 2", searcher.topK)
	}
	if len(documents) != 2 || documents[0].MetaData[MetadataSourcePath] != "a.md" ||
		documents[1].MetaData[MetadataSourcePath] != "b.md" {
		t.Fatalf("documents = %#v, want a.md then b.md", documents)
	}
	if documents[0].Score() != 7.25 {
		t.Fatalf("score = %v, want 7.25", documents[0].Score())
	}
	for _, key := range []string{
		MetadataDocumentID, MetadataSourcePath, MetadataDocumentTitle,
		MetadataHeadingPath, MetadataStartLine, MetadataEndLine,
		MetadataChunkIndex, MetadataContentHash, MetadataDocumentVersion, MetadataSimilarity,
	} {
		if _, ok := documents[0].MetaData[key]; !ok {
			t.Errorf("metadata %q is missing", key)
		}
	}
}

func TestBM25RetrieverValidation(t *testing.T) {
	if _, err := NewBM25Retriever(nil, Config{}); err == nil {
		t.Fatal("nil searcher should fail")
	}
	if _, err := NewBM25Retriever(&fakeBM25Searcher{}, Config{CandidateTopK: -1}); err == nil {
		t.Fatal("negative topK should fail")
	}
	retriever, err := NewBM25Retriever(&fakeBM25Searcher{}, Config{CandidateTopK: 5})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}
	if _, err := retriever.Retrieve(context.Background(), "   "); err == nil {
		t.Fatal("blank query should fail")
	}
	searcher := &fakeBM25Searcher{err: bm25.ErrSearch}
	broken, err := NewBM25Retriever(searcher, Config{CandidateTopK: 5})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}
	if _, err := broken.Retrieve(context.Background(), "query"); !errors.Is(err, ErrSearch) {
		t.Fatalf("Retrieve() error = %v, want ErrSearch", err)
	}
}
