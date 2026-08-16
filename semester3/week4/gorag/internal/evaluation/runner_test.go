package evaluation

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeClient struct {
	responses map[string]Response
	err       error
}

func (client fakeClient) Ask(_ context.Context, question string) (Response, error) {
	if client.err != nil {
		return Response{}, client.err
	}
	return client.responses[question], nil
}

func TestRunnerComparesEveryExpectation(t *testing.T) {
	testCase := Case{
		ID: "payment", Line: 7, Question: "q", ShouldAnswer: true, Scenario: "cross_section",
		ExpectedSources:  []string{"a.md", "nested/b.md"},
		ExpectedHeadings: []string{"Design", "Transactions"},
		AnswerKeywords:   []string{"River", "事务"},
	}
	client := fakeClient{responses: map[string]Response{"q": {
		Answerable: true,
		Answer:     "通过 River 保持事务一致性。",
		Sources: []Source{
			{SourcePath: "docs/a.md", DocumentTitle: "Guide", HeadingPath: []string{"Design"}},
			{SourcePath: "nested/b.md", Title: "Transactions"},
		},
	}}}
	runner, err := NewRunner(client)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	report, err := runner.Run(context.Background(), []Case{testCase})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Successful() {
		var output bytes.Buffer
		_ = report.WriteText(&output)
		t.Fatalf("Run() report failed:\n%s", output.String())
	}
	if report.SourceHits != 2 || report.HeadingHits != 2 || report.KeywordHits != 2 {
		t.Fatalf("Run() metrics = sources %d, headings %d, keywords %d", report.SourceHits, report.HeadingHits, report.KeywordHits)
	}
}

func TestRunnerFailureReportLocatesMismatch(t *testing.T) {
	testCase := Case{
		ID: "redis", Line: 11, Question: "q", ShouldAnswer: true, Scenario: "in_scope",
		ExpectedSources: []string{"Redis.md"}, ExpectedHeadings: []string{"Redis"}, AnswerKeywords: []string{"内存"},
	}
	runner, _ := NewRunner(fakeClient{responses: map[string]Response{"q": {
		Answerable: false, Answer: "无法回答", Sources: []Source{{SourcePath: "other.md", HeadingPath: []string{"Other"}}},
	}}})
	report, err := runner.Run(context.Background(), []Case{testCase})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Successful() || len(report.Results[0].Failures) != 4 {
		t.Fatalf("Run() failures = %#v, want answerable/source/heading/keyword", report.Results[0].Failures)
	}
	var output bytes.Buffer
	if err := report.WriteText(&output); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}
	for _, fragment := range []string{`FAIL line=11 id="redis"`, "field=answerable", "field=source", "field=heading", "field=keyword", "Redis.md", "other.md"} {
		if !strings.Contains(output.String(), fragment) {
			t.Errorf("report missing %q:\n%s", fragment, output.String())
		}
	}
}

func TestRunnerRecordsClientFailureAndContinues(t *testing.T) {
	runner, _ := NewRunner(fakeClient{err: errors.New("model unavailable")})
	cases := []Case{
		{ID: "one", Question: "q1", Scenario: "out_of_scope"},
		{ID: "two", Question: "q2", Scenario: "ambiguous"},
	}
	report, err := runner.Run(context.Background(), cases)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Results) != 2 || report.Results[0].Failures[0].Field != "request" {
		t.Fatalf("Run() report = %#v", report)
	}
}

type contextClient struct{}

func (contextClient) Ask(ctx context.Context, _ string) (Response, error) {
	<-ctx.Done()
	return Response{}, ctx.Err()
}

func TestRunnerPropagatesCancellation(t *testing.T) {
	runner, _ := NewRunner(contextClient{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runner.Run(ctx, []Case{{ID: "one", Question: "q", Scenario: "ambiguous"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}
