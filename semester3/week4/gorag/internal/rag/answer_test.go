package rag

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
)

type stubInvoker struct {
	result Result
	err    error
	called bool
}

func (s *stubInvoker) Invoke(context.Context, string) (Result, error) {
	s.called = true
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

	answer, gotErr := service.AnswerQuestion(context.Background(), "question")
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
	answer, err := service.AnswerQuestion(context.Background(), "question")
	if err != nil || answer.Answerable || answer.Text != RefusalAnswer {
		t.Fatalf("AnswerQuestion() = %#v, %v", answer, err)
	}
}

func TestAnswerServiceInsufficientContextIsNormalRefusal(t *testing.T) {
	chain := &stubInvoker{err: ErrInsufficientContext}
	service, _ := NewAnswerService(chain)
	answer, err := service.AnswerQuestion(context.Background(), "question")
	if err != nil || answer.Answerable || answer.Text != RefusalAnswer {
		t.Fatalf("AnswerQuestion() = %#v, %v", answer, err)
	}
}

func TestAnswerServiceRejectsUnderspecifiedQuestionBeforeRetrieval(t *testing.T) {
	chain := &stubInvoker{}
	service, _ := NewAnswerService(chain)
	answer, err := service.AnswerQuestion(context.Background(), "它要怎么处理？")
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
	_, err := service.AnswerQuestion(ctx, "question")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if chain.called {
		t.Fatal("chain called after context cancellation")
	}
}
