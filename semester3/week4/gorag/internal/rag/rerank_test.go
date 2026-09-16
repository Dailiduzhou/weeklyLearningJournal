package rag

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/document"
	"gorag/internal/repository"
	queryretriever "gorag/internal/retriever"
)

type rerankParentStore struct{}

func (rerankParentStore) GetParentDocuments(_ context.Context, ids []int64) ([]repository.ParentDocument, error) {
	// Deliberately reverse storage order: the ordinal must survive independently
	// of parent-store ordering and best-child scores.
	parents := make([]repository.ParentDocument, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		path, title := "low.md", "Low score"
		if id == 1 {
			path, title = "high.md", "High score"
		}
		parents = append(parents, repository.ParentDocument{DocumentID: id, SourcePath: path, Title: title, Version: "v1", Content: "Whole parent " + path, StartLine: 7, EndLine: 30})
	}
	return parents, nil
}

func TestRerankThroughParentContextAndCitationPreservesProvenance(t *testing.T) {
	for _, parent := range []bool{false, true} {
		t.Run(map[bool]string{false: "chunks", true: "parents"}[parent], func(t *testing.T) {
			high := retrievalDocument("high", "high.md", 0, 0.95)
			lowFirst := retrievalDocument("low-first", "low.md", 1, 0.6)
			lowSecond := retrievalDocument("low-second", "low.md", 2, 0.7)
			lowFirst.MetaData[queryretriever.MetadataDocumentID] = "2"
			lowSecond.MetaData[queryretriever.MetadataDocumentID] = "2"
			docs := []*schema.Document{high, lowSecond, lowFirst}
			child := &stubRetriever{documents: docs}
			rerankModel := &recordingModel{response: schema.AssistantMessage(`{"ranking":["C3","C2","C1"]}`, nil)}
			reranked, err := queryretriever.NewLLMRerankRetriever(child, rerankModel, queryretriever.RerankConfig{Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var online einoretriever.Retriever = reranked
			if parent {
				online, err = queryretriever.NewParentDocumentRetriever(context.Background(), online, rerankParentStore{})
				if err != nil {
					t.Fatal(err)
				}
			}
			builder, err := NewContextBuilder(2)
			if err != nil {
				t.Fatal(err)
			}
			answerModel := &recordingModel{response: schema.AssistantMessage("First evidence [S1]. Second evidence [S2].", nil)}
			chain, err := NewChain(context.Background(), online, builder, nil, answerModel)
			if err != nil {
				t.Fatal(err)
			}
			filter := document.MetadataFilter{Metadata: map[string]string{"category": "go"}, Tags: []string{"backend"}}
			result, err := chain.Invoke(context.Background(), "Where should the transaction boundary be?", queryretriever.WithFilter(filter))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Sources) != 2 || result.Sources[0].SourcePath != "low.md" {
				t.Fatalf("rerank lost: %#v", result.Sources)
			}
			wantScore := 0.6
			if parent {
				wantScore = 0.7
			}
			if result.Sources[0].Similarity != wantScore {
				t.Fatalf("score replaced by rank: %#v", result.Sources[0])
			}
			if parent {
				if result.Sources[1].SourcePath != "high.md" {
					t.Fatal("truncated before parent deduplication")
				}
				if result.Sources[0].StartLine != 7 || result.Sources[0].EndLine != 30 || result.Sources[0].DocumentVersion != "v1" || result.SourceDocuments["S1"].Content != "Whole parent low.md" {
					t.Fatal("parent provenance/content changed")
				}
			} else if result.SourceDocuments["S1"].Content != lowFirst.Content || result.Sources[0].ChunkIndex != 1 {
				t.Fatal("chunk provenance/content changed")
			}
			gotFilter, err := queryretriever.FilterFromOptions(child.options...)
			if err != nil || !reflect.DeepEqual(gotFilter, filter) {
				t.Fatalf("filter lost: %#v %v", gotFilter, err)
			}
			messages, calls := rerankModel.snapshot()
			if calls != 1 || strings.Contains(messages[1].Content, "Whole parent") || !strings.Contains(messages[1].Content, lowFirst.Content) {
				t.Fatal("rerank not applied to chunks")
			}
			answer, err := ValidateAnswer(result)
			if err != nil || !answer.Answerable || len(answer.Sources) != 2 || answer.Sources[0].ID != "S1" || answer.Sources[0].SourcePath != "low.md" {
				t.Fatalf("citation validation failed: %#v %v", answer, err)
			}

			// Reuse exactly the same upstream documents after a successful request.
			// Fallback must recover the original score-based order with fresh
			// parent collectors and no stale rerank metadata.
			rerankModel.err = errors.New("model unavailable")
			fallback, err := chain.Invoke(context.Background(), "Where should the transaction boundary be?")
			if err != nil || fallback.Sources[0].SourcePath != "high.md" || fallback.Sources[0].Similarity != 0.95 {
				t.Fatalf("fallback not original: %#v %v", fallback.Sources, err)
			}
			for _, doc := range docs {
				if _, exists := doc.MetaData[queryretriever.MetadataRerankPosition]; exists {
					t.Fatal("input metadata mutated")
				}
			}
		})
	}
}

func TestContextBuilderRejectsBrokenRerankContract(t *testing.T) {
	for _, values := range [][]any{{1, nil}, {1, 1}, {0, 2}, {"1", 2}} {
		docs := []*schema.Document{retrievalDocument("one", "one.md", 0, .9), retrievalDocument("two", "two.md", 0, .8)}
		for i, value := range values {
			if value != nil {
				docs[i].MetaData[queryretriever.MetadataRerankPosition] = value
			}
		}
		builder, err := NewContextBuilder(1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := builder.Build(context.Background(), docs); !errors.Is(err, ErrInvalidContext) {
			t.Fatalf("broken contract accepted: %v", err)
		}
	}
}

type cancelRerankModel struct{ cancel context.CancelFunc }

func (m cancelRerankModel) Generate(ctx context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.cancel()
	return nil, ctx.Err()
}
func (m cancelRerankModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unused")
}

func TestRequestCanceledDuringRerankNeverCallsAnswerModel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	child := &stubRetriever{documents: []*schema.Document{retrievalDocument("one", "one.md", 0, .9), retrievalDocument("two", "two.md", 0, .8)}}
	ranked, err := queryretriever.NewLLMRerankRetriever(child, cancelRerankModel{cancel}, queryretriever.RerankConfig{Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := NewContextBuilder(1)
	if err != nil {
		t.Fatal(err)
	}
	answerModel := &recordingModel{}
	chain, err := NewChain(ctx, ranked, builder, nil, answerModel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.Invoke(ctx, "Where should the transaction boundary be?"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, calls := answerModel.snapshot(); calls != 0 {
		t.Fatal("answer model called after cancellation")
	}
}
