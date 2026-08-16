package rag

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

func TestChainInvokesGroundedEinoGenerationAndReturnsSources(t *testing.T) {
	document := retrievalDocument("doc", "guide.md", 0, 0.9)
	retrieval := &stubRetriever{documents: []*schema.Document{document}}
	chatModel := &recordingModel{response: schema.AssistantMessage("Use the guide [S1].", nil)}
	builder, err := NewContextBuilder(5)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewChain(context.Background(), retrieval, builder, nil, chatModel)
	if err != nil {
		t.Fatalf("NewChain() error = %v", err)
	}

	result, err := chain.Invoke(context.Background(), "How do I use the guide?")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Message.Content != "Use the guide [S1]." {
		t.Fatalf("Message.Content = %q", result.Message.Content)
	}
	if len(result.Sources) != 1 || result.Sources[0].ID != "S1" {
		t.Fatalf("Sources = %#v", result.Sources)
	}
	if result.SourceDocuments["S1"] != document {
		t.Fatal("SourceDocuments[S1] does not map to the retrieved document")
	}
	messages, calls := chatModel.snapshot()
	if calls != 1 || len(messages) != 2 {
		t.Fatalf("model calls/messages = %d/%d, want 1/2", calls, len(messages))
	}
	if !strings.Contains(messages[1].Content, "How do I use the guide?") ||
		!strings.Contains(messages[1].Content, "[S1]\npath: guide.md") {
		t.Fatalf("model prompt is not grounded:\n%s", messages[1].Content)
	}
}

func TestChainStopsBeforeModelForNoContext(t *testing.T) {
	retrieval := &stubRetriever{}
	chatModel := &recordingModel{response: schema.AssistantMessage("must not run", nil)}
	builder, err := NewContextBuilder(5)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewChain(context.Background(), retrieval, builder, nil, chatModel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chain.Invoke(context.Background(), "unknown question")
	if !errors.Is(err, ErrInsufficientContext) {
		t.Fatalf("Invoke() error = %v, want ErrInsufficientContext", err)
	}
	_, calls := chatModel.snapshot()
	if calls != 0 {
		t.Fatalf("model calls = %d, want 0", calls)
	}
}

func TestChainPropagatesRetrieverFailureAndCancellation(t *testing.T) {
	retrievalFailure := errors.New("retrieval failed")
	retrieval := &stubRetriever{err: retrievalFailure}
	chatModel := &recordingModel{}
	builder, err := NewContextBuilder(5)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewChain(context.Background(), retrieval, builder, nil, chatModel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chain.Invoke(context.Background(), "question")
	if !errors.Is(err, retrievalFailure) {
		t.Fatalf("retrieval error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = chain.Invoke(ctx, "question")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	_, calls := chatModel.snapshot()
	if calls != 0 {
		t.Fatalf("model calls = %d, want 0", calls)
	}
}

func TestChainWrapsGenerationFailure(t *testing.T) {
	modelFailure := errors.New("model unavailable")
	retrieval := &stubRetriever{documents: []*schema.Document{
		retrievalDocument("doc", "guide.md", 0, 0.9),
	}}
	chatModel := &recordingModel{err: modelFailure}
	builder, err := NewContextBuilder(5)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := NewChain(context.Background(), retrieval, builder, nil, chatModel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chain.Invoke(context.Background(), "question")
	if !errors.Is(err, ErrGeneration) || !errors.Is(err, modelFailure) {
		t.Fatalf("generation error = %v", err)
	}
}

type stubRetriever struct {
	documents []*schema.Document
	err       error
}

func (r *stubRetriever) Retrieve(ctx context.Context, _ string, _ ...einoretriever.Option) ([]*schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.documents, r.err
}

type recordingModel struct {
	mu       sync.Mutex
	calls    int
	messages []*schema.Message
	response *schema.Message
	err      error
}

var _ model.BaseChatModel = (*recordingModel)(nil)

func (m *recordingModel) Generate(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.messages = append([]*schema.Message(nil), input...)
	return m.response, m.err
}

func (m *recordingModel) Stream(ctx context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("stream is not implemented by test model")
}

func (m *recordingModel) snapshot() ([]*schema.Message, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*schema.Message(nil), m.messages...), m.calls
}
