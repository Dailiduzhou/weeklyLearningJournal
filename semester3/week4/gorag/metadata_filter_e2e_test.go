// Package-level end-to-end test for metadata filtering: it wires the real
// transport, answer service, chain, fusion retriever, and parent document
// retriever together, faking only the vector store, the lexical store, and
// the chat model. This guards every wiring seam at once: an HTTP filter must
// reach both search backends and produce a grounded, cited answer.
package gorag_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/bm25"
	"gorag/internal/document"
	"gorag/internal/embedding"
	"gorag/internal/rag"
	"gorag/internal/repository"
	"gorag/internal/retriever"
	"gorag/internal/transport"
)

// recordingVectorStore implements retriever.Searcher and records filters.
type recordingVectorStore struct {
	mu     sync.Mutex
	filter document.MetadataFilter
	limit  int
	calls  int
}

func (s *recordingVectorStore) Search(ctx context.Context, _ []float32, limit int, filter document.MetadataFilter) ([]repository.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.filter, s.limit = filter, limit
	s.calls++
	s.mu.Unlock()
	return []repository.SearchResult{
		{
			DocumentID: 17,
			Chunk: document.Chunk{
				DocumentID: "17", SourcePath: "api/auth.md", DocumentTitle: "Authentication",
				HeadingPath: []string{"API"}, Index: 0, Content: "bff crypto optimization notes",
				StartLine: 5, EndLine: 40, ContentHash: "hash-17",
				DocumentVersion: "v1", EmbeddingModel: "qwen3-embedding:0.6b", EmbeddingDimension: 1024,
			},
			Similarity: 0.92,
		},
	}, nil
}

func (s *recordingVectorStore) recorded() (document.MetadataFilter, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.filter, s.calls
}

// recordingBM25Store implements retriever.BM25Searcher and records filters.
type recordingBM25Store struct {
	mu     sync.Mutex
	filter document.MetadataFilter
	calls  int
}

func (s *recordingBM25Store) Search(ctx context.Context, _ string, _ int, filter document.MetadataFilter) ([]bm25.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.filter = filter
	s.calls++
	s.mu.Unlock()
	return []bm25.SearchResult{
		{
			DocumentID: 17,
			Chunk: document.Chunk{
				DocumentID: "17", SourcePath: "api/auth.md", DocumentTitle: "Authentication",
				HeadingPath: []string{"API"}, Index: 0, Content: "bff crypto optimization notes",
				StartLine: 5, EndLine: 40, ContentHash: "hash-17", DocumentVersion: "v1",
			},
			Score: 6.5,
		},
	}, nil
}

func (s *recordingBM25Store) recorded() (document.MetadataFilter, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.filter, s.calls
}

// inMemoryParentStore implements retriever.ParentStore.
type inMemoryParentStore struct{}

func (inMemoryParentStore) GetParentDocuments(ctx context.Context, ids []int64) ([]repository.ParentDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parents := make([]repository.ParentDocument, 0, len(ids))
	for _, id := range ids {
		if id == 17 {
			parents = append(parents, repository.ParentDocument{
				DocumentID: 17, SourcePath: "api/auth.md", Title: "Authentication",
				Version: "v1", Content: "# Authentication\n\nWhole document about the bff crypto optimization.",
				StartLine: 1, EndLine: 120,
			})
		}
	}
	return parents, nil
}

// unitEmbedder supplies a fixed 1024-dimension query vector.
type unitEmbedder struct{}

func (unitEmbedder) EmbedQuery(context.Context, string) ([]float32, error) {
	return make([]float32, embedding.VectorDimension), nil
}

// alwaysReady implements transport.ReadyChecker.
type alwaysReady struct{}

func (alwaysReady) Check(context.Context) error { return nil }

// scriptedChatModel answers with one grounded citation.
type scriptedChatModel struct{}

func (scriptedChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("bff 的加密优化把令牌签名换成了 HMAC 并缩短了过期时间 [S1]。", nil), nil
}

func (scriptedChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func TestMetadataFilterFlowsFromHTTPOpenEndedToBothBackends(t *testing.T) {
	ctx := context.Background()
	vectorStore := &recordingVectorStore{}
	bm25Store := &recordingBM25Store{}

	vectorRetriever, err := retriever.NewPgVectorRetriever(unitEmbedder{}, vectorStore, retriever.Config{
		CandidateTopK: 10, SimilarityThreshold: 0.5,
	})
	if err != nil {
		t.Fatalf("NewPgVectorRetriever() error = %v", err)
	}
	bm25Retriever, err := retriever.NewBM25Retriever(bm25Store, retriever.Config{CandidateTopK: 10})
	if err != nil {
		t.Fatalf("NewBM25Retriever() error = %v", err)
	}
	fused, err := retriever.NewFusionRetriever(10, retriever.DefaultFusionK, vectorRetriever, bm25Retriever)
	if err != nil {
		t.Fatalf("NewFusionRetriever() error = %v", err)
	}
	parentRetriever, err := retriever.NewParentDocumentRetriever(ctx, fused, inMemoryParentStore{})
	if err != nil {
		t.Fatalf("NewParentDocumentRetriever() error = %v", err)
	}

	builder, err := rag.NewContextBuilder(5)
	if err != nil {
		t.Fatalf("NewContextBuilder() error = %v", err)
	}
	chain, err := rag.NewChain(ctx, parentRetriever, builder, nil, scriptedChatModel{})
	if err != nil {
		t.Fatalf("NewChain() error = %v", err)
	}
	answerService, err := rag.NewAnswerService(chain)
	if err != nil {
		t.Fatalf("NewAnswerService() error = %v", err)
	}
	handler := transport.NewHandler(answerService, alwaysReady{}, nil)

	body := `{"question":"bff 的加密优化做了什么？","filter":{"metadata":{"category":"ccnubox","module":"bff"},"tags":["ccnubox/bff"]}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var answer rag.Answer
	if err := json.Unmarshal(response.Body.Bytes(), &answer); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !answer.Answerable || len(answer.Sources) != 1 || answer.Sources[0].SourcePath != "api/auth.md" {
		t.Fatalf("answer = %#v, want a grounded answer citing the parent document", answer)
	}
	if answer.Sources[0].StartLine != 1 || answer.Sources[0].EndLine != 120 {
		t.Fatalf("source lines = %d..%d, want the whole-document parent range", answer.Sources[0].StartLine, answer.Sources[0].EndLine)
	}

	wantFilter := document.MetadataFilter{
		Metadata: map[string]string{"category": "ccnubox", "module": "bff"},
		Tags:     []string{"ccnubox/bff"},
	}
	if got, calls := vectorStore.recorded(); calls != 1 ||
		len(got.Metadata) != 2 || got.Metadata["module"] != "bff" || got.Tags[0] != "ccnubox/bff" {
		t.Fatalf("vector backend filter = %#v, calls = %d, want %#v once", got, calls, wantFilter)
	}
	if got, calls := bm25Store.recorded(); calls != 1 ||
		len(got.Metadata) != 2 || got.Metadata["category"] != "ccnubox" || got.Tags[0] != "ccnubox/bff" {
		t.Fatalf("bm25 backend filter = %#v, calls = %d, want %#v once", got, calls, wantFilter)
	}

	// An invalid filter never reaches retrieval: 400 without a search.
	request = httptest.NewRequest(http.MethodPost, "/api/v1/questions",
		strings.NewReader(`{"question":"问题","filter":{"tags":[""]}}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_filter") {
		t.Fatalf("status = %d, body = %s, want 400 invalid_filter", response.Code, response.Body.String())
	}
	if _, calls := vectorStore.recorded(); calls != 1 {
		t.Fatalf("vector search calls = %d, want no new search for an invalid request", calls)
	}
	if _, calls := bm25Store.recorded(); calls != 1 {
		t.Fatalf("bm25 search calls = %d, want no new search for an invalid request", calls)
	}
}
