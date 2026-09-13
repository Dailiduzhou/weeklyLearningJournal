package rag

import (
	"context"
	"errors"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"github.com/cloudwego/eino/schema"

	queryretriever "gorag/internal/retriever"
)

type stubInvoker struct {
	result  Result
	err     error
	called  bool
	options []einoretriever.Option
}

func (s *stubInvoker) Invoke(_ context.Context, _ string, opts ...einoretriever.Option) (Result, error) {
	s.called = true
	s.options = append([]einoretriever.Option(nil), opts...)
	return s.result, s.err
}

func TestValidateAnswerAcceptsKnownCitationsAndDeduplicatesSources(t *testing.T) {
	result := Result{
		Message: &schema.Message{Content: "结论一 [S2]，结论二 [S1]，再次说明 [S2]。"},
		Sources: []Source{
			{ID: "S1", SourcePath: "docs/a.md", DocumentTitle: "A", HeadingPath: []string{"H"}, StartLine: 2, EndLine: 4, Similarity: .8},
			{ID: "S2", SourcePath: "docs/b.md", DocumentTitle: "B", StartLine: 7, EndLine: 9, Similarity: .9},
		},
		SourceDocuments: map[string]*schema.Document{"S1": {}, "S2": {}},
	}

	answer, err := ValidateAnswer(result)
	if err != nil {
		t.Fatalf("ValidateAnswer() error = %v", err)
	}
	if !answer.Answerable || answer.Text != result.Message.Content {
		t.Fatalf("unexpected answer: %#v", answer)
	}
	if got, want := len(answer.Sources), 2; got != want {
		t.Fatalf("sources length = %d, want %d", got, want)
	}
	if answer.Sources[0].ID != "S2" || answer.Sources[1].ID != "S1" {
		t.Fatalf("sources not in first-citation order: %#v", answer.Sources)
	}
	result.Sources[0].HeadingPath[0] = "changed"
	if answer.Sources[1].HeadingPath[0] != "H" {
		t.Fatal("answer source aliases internal heading path")
	}
}

func TestValidateAnswerRejectsInvalidCitations(t *testing.T) {
	source := Source{ID: "S1"}
	tests := []struct {
		name    string
		message *schema.Message
		sources []Source
	}{
		{name: "nil message", sources: []Source{source}},
		{name: "empty answer", message: &schema.Message{}},
		{name: "missing citation", message: &schema.Message{Content: "没有引用"}, sources: []Source{source}},
		{name: "model refusal token", message: &schema.Message{Content: modelRefusalToken}, sources: []Source{source}},
		{name: "model refusal prose", message: &schema.Message{Content: "现有材料不足，无法回答。[S1]"}, sources: []Source{source}},
		{name: "unknown", message: &schema.Message{Content: "伪造 [S2]"}, sources: []Source{source}},
		{name: "zero", message: &schema.Message{Content: "伪造 [S0]"}, sources: []Source{source}},
		{name: "malformed", message: &schema.Message{Content: "伪造 [S 1]"}, sources: []Source{source}},
		{name: "duplicate context ID", message: &schema.Message{Content: "结论 [S1]"}, sources: []Source{source, source}},
		{name: "invalid context ID", message: &schema.Message{Content: "结论 [S1]"}, sources: []Source{{ID: "S01"}}},
		{name: "not retrieved", message: &schema.Message{Content: "伪造 [S1]"}, sources: []Source{source}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			documents := make(map[string]*schema.Document, len(test.sources))
			for _, source := range test.sources {
				documents[source.ID] = &schema.Document{}
			}
			if test.name == "not retrieved" {
				documents = nil
			}
			if _, err := ValidateAnswer(Result{Message: test.message, Sources: test.sources, SourceDocuments: documents}); err == nil {
				t.Fatal("ValidateAnswer() error = nil")
			}
		})
	}
}

func TestAnswerServiceNormalizesFailuresToRefusal(t *testing.T) {
	dependencyErr := errors.New("database password must not leak")
	chain := &stubInvoker{err: dependencyErr}
	service, err := NewAnswerService(chain)
	if err != nil {
		t.Fatal(err)
	}

	answer, gotErr := service.AnswerQuestion(context.Background(), "question", Filter{})
	if !errors.Is(gotErr, dependencyErr) {
		t.Fatalf("error = %v, want wrapped dependency error", gotErr)
	}
	if answer.Answerable || answer.Text != RefusalAnswer || len(answer.Sources) != 0 {
		t.Fatalf("answer = %#v, want refusal", answer)
	}
}

func TestAnswerServiceInvalidModelCitationBecomesRefusal(t *testing.T) {
	chain := &stubInvoker{result: Result{
		Message: &schema.Message{Content: "hallucination [S99]"},
		Sources: []Source{{ID: "S1"}},
	}}
	service, _ := NewAnswerService(chain)
	answer, err := service.AnswerQuestion(context.Background(), "question", Filter{})
	if err != nil || answer.Answerable || answer.Text != RefusalAnswer {
		t.Fatalf("AnswerQuestion() = %#v, %v", answer, err)
	}
}

func TestAnswerServiceInsufficientContextIsNormalRefusal(t *testing.T) {
	chain := &stubInvoker{err: ErrInsufficientContext}
	service, _ := NewAnswerService(chain)
	answer, err := service.AnswerQuestion(context.Background(), "question", Filter{})
	if err != nil || answer.Answerable || answer.Text != RefusalAnswer {
		t.Fatalf("AnswerQuestion() = %#v, %v", answer, err)
	}
}

func TestAnswerServiceRejectsUnderspecifiedQuestionBeforeRetrieval(t *testing.T) {
	chain := &stubInvoker{}
	service, _ := NewAnswerService(chain)
	answer, err := service.AnswerQuestion(context.Background(), "它要怎么处理？", Filter{})
	if err != nil || answer.Answerable || answer.Text != RefusalAnswer {
		t.Fatalf("AnswerQuestion() = %#v, %v", answer, err)
	}
	if chain.called {
		t.Fatal("chain called for an underspecified question")
	}
}

func TestAnswerServicePropagatesCancellation(t *testing.T) {
	chain := &stubInvoker{}
	service, _ := NewAnswerService(chain)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.AnswerQuestion(ctx, "question", Filter{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if chain.called {
		t.Fatal("chain called after context cancellation")
	}
}

func TestAnswerQuestionForwardsMetadataFilterAsRetrieverOption(t *testing.T) {
	chain := &stubInvoker{result: Result{
		Message:         &schema.Message{Content: "答案 [S1]。"},
		Sources:         []Source{{ID: "S1", SourcePath: "docs/a.md", DocumentTitle: "A", HeadingPath: []string{"H"}, StartLine: 2, EndLine: 4}},
		SourceDocuments: map[string]*schema.Document{"S1": {}},
	}}
	service, err := NewAnswerService(chain)
	if err != nil {
		t.Fatal(err)
	}

	// An empty filter must not add options: retrieval behaviour is unchanged.
	if _, err := service.AnswerQuestion(context.Background(), "question", Filter{}); err != nil {
		t.Fatalf("AnswerQuestion(empty filter) error = %v", err)
	}
	if len(chain.options) != 0 {
		t.Fatalf("chain options = %d, want none for an empty filter", len(chain.options))
	}

	filter := Filter{Metadata: map[string]string{"category": "ccnubox"}, Tags: []string{"ccnubox/bff"}}
	answer, err := service.AnswerQuestion(context.Background(), "question", filter)
	if err != nil {
		t.Fatalf("AnswerQuestion(filter) error = %v", err)
	}
	if !answer.Answerable {
		t.Fatalf("AnswerQuestion(filter) = %#v, want a grounded answer", answer)
	}
	extracted, err := queryretriever.FilterFromOptions(chain.options...)
	if err != nil {
		t.Fatalf("FilterFromOptions() error = %v", err)
	}
	if extracted.Metadata["category"] != "ccnubox" || extracted.Tags[0] != "ccnubox/bff" {
		t.Fatalf("forwarded filter = %#v, want the caller's filter", extracted)
	}

	// A malformed filter is rejected before the chain runs.
	invalid := Filter{Tags: []string{" "}}
	if _, err := service.AnswerQuestion(context.Background(), "question", invalid); err == nil {
		t.Fatal("AnswerQuestion(invalid filter) error = nil")
	}
}
