package evaluation

import (
	"strings"
	"testing"
)

func TestLoadJSONL(t *testing.T) {
	input := strings.Join([]string{
		`{"id":"known","question":"what?","should_answer":true,"expected_sources":["a.md"],"expected_headings":["A"],"answer_keywords":["fact"],"scenario":"in_scope"}`,
		``,
		`{"id":"unknown","question":"else?","should_answer":false,"expected_sources":[],"expected_headings":[],"answer_keywords":[],"scenario":"out_of_scope"}`,
	}, "\n")

	cases, err := LoadJSONL(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadJSONL() error = %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("LoadJSONL() count = %d, want 2", len(cases))
	}
	if cases[0].Line != 1 || cases[1].Line != 3 {
		t.Fatalf("LoadJSONL() lines = %d,%d, want 1,3", cases[0].Line, cases[1].Line)
	}
}

func TestLoadJSONLRejectsInvalidDataset(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "missing required array", input: `{"id":"x","question":"q","should_answer":false,"expected_headings":[],"answer_keywords":[],"scenario":"out_of_scope"}`, want: "expected_sources"},
		{name: "unknown field", input: `{"id":"x","question":"q","should_answer":false,"expected_sources":[],"expected_headings":[],"answer_keywords":[],"scenario":"out_of_scope","typo":1}`, want: "unknown field"},
		{name: "answerable without keyword", input: `{"id":"x","question":"q","should_answer":true,"expected_sources":["a.md"],"expected_headings":["A"],"answer_keywords":[],"scenario":"in_scope"}`, want: "answer_keywords"},
		{name: "duplicate id", input: strings.Join([]string{
			`{"id":"x","question":"q","should_answer":false,"expected_sources":[],"expected_headings":[],"answer_keywords":[],"scenario":"out_of_scope"}`,
			`{"id":"x","question":"q2","should_answer":false,"expected_sources":[],"expected_headings":[],"answer_keywords":[],"scenario":"ambiguous"}`,
		}, "\n"), want: "duplicate id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadJSONL(strings.NewReader(test.input))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadJSONL() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestValidateCoverage(t *testing.T) {
	cases := make([]Case, 0, 5+len(RequiredScenarios))
	for index := 0; index < 5; index++ {
		cases = append(cases, Case{
			ID: "known-" + string(rune('a'+index)), Question: "q", ShouldAnswer: true,
			ExpectedSources: []string{"a.md"}, ExpectedHeadings: []string{"A"},
			AnswerKeywords: []string{"fact"}, Scenario: "in_scope",
		})
	}
	for _, scenario := range RequiredScenarios {
		cases = append(cases, Case{ID: scenario, Question: "q", Scenario: scenario})
	}
	// Required lifecycle/paraphrase cases may themselves be answerable, but the
	// five explicit in-scope cases above make this fixture's intent unambiguous.
	if err := ValidateCoverage(cases); err != nil {
		t.Fatalf("ValidateCoverage() error = %v", err)
	}
}
