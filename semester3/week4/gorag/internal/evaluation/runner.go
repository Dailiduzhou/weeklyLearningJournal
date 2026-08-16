package evaluation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Client permits both an in-process RAG adapter and HTTPClient to drive the
// same evaluator.
type Client interface {
	Ask(context.Context, string) (Response, error)
}

// Response is the semantic question API response used by evaluation.
type Response struct {
	Answerable bool     `json:"answerable"`
	Answer     string   `json:"answer"`
	Sources    []Source `json:"sources"`
}

// Source contains the traceable source fields from the question API. Title
// and DocumentTitle are both accepted while the transport contract settles;
// HeadingPath is the authoritative structured title hierarchy.
type Source struct {
	ID            string   `json:"id,omitempty"`
	SourcePath    string   `json:"source_path"`
	Title         string   `json:"title,omitempty"`
	DocumentTitle string   `json:"document_title,omitempty"`
	HeadingPath   []string `json:"heading_path,omitempty"`
	StartLine     int      `json:"start_line,omitempty"`
	EndLine       int      `json:"end_line,omitempty"`
	Similarity    float64  `json:"similarity,omitempty"`
}

// Failure records one independently evaluated expectation.
type Failure struct {
	Field    string
	Expected string
	Actual   string
}

// Result is the report for one dataset line.
type Result struct {
	Case     Case
	Response Response
	Failures []Failure
}

// Passed reports whether all expectations for the case matched.
func (result Result) Passed() bool { return len(result.Failures) == 0 }

// Report includes raw results and aggregate metrics used during RAG tuning.
type Report struct {
	Results            []Result
	Total              int
	Passed             int
	AnswerableExpected int
	AnswerableCorrect  int
	RefusalExpected    int
	RefusalCorrect     int
	SourceExpected     int
	SourceHits         int
	HeadingExpected    int
	HeadingHits        int
	KeywordExpected    int
	KeywordHits        int
}

// Successful reports whether every evaluation case passed.
func (report Report) Successful() bool { return report.Total > 0 && report.Passed == report.Total }

// Runner executes cases sequentially to keep reports deterministic. The
// supplied context is passed unchanged to every client call.
type Runner struct{ client Client }

func NewRunner(client Client) (*Runner, error) {
	if client == nil {
		return nil, errors.New("evaluation client must not be nil")
	}
	return &Runner{client: client}, nil
}

// Run records endpoint failures per case and continues so one bad question
// does not hide later regressions. Cancellation of the outer context stops the
// run immediately and returns the partial report with context.Cause.
func (runner *Runner) Run(ctx context.Context, cases []Case) (Report, error) {
	if runner == nil || runner.client == nil {
		return Report{}, errors.New("evaluation runner is not initialized")
	}
	if len(cases) == 0 {
		return Report{}, errors.New("evaluation cases must not be empty")
	}

	report := Report{Total: len(cases)}
	for _, testCase := range cases {
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("evaluation stopped before case %q: %w", testCase.ID, context.Cause(ctx))
		}
		if err := testCase.validate(); err != nil {
			return report, fmt.Errorf("validate case %q: %w", testCase.ID, err)
		}

		response, err := runner.client.Ask(ctx, testCase.Question)
		if err != nil && ctx.Err() != nil {
			return report, fmt.Errorf("evaluation stopped in case %q: %w", testCase.ID, context.Cause(ctx))
		}

		result := Result{Case: testCase, Response: response}
		if err != nil {
			result.Failures = append(result.Failures, Failure{Field: "request", Expected: "successful response", Actual: err.Error()})
		} else {
			compare(&result, &report)
		}
		if result.Passed() {
			report.Passed++
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func compare(result *Result, report *Report) {
	testCase := result.Case
	response := result.Response

	if testCase.ShouldAnswer {
		report.AnswerableExpected++
	} else {
		report.RefusalExpected++
	}
	if response.Answerable == testCase.ShouldAnswer {
		if testCase.ShouldAnswer {
			report.AnswerableCorrect++
		} else {
			report.RefusalCorrect++
		}
	} else {
		result.Failures = append(result.Failures, Failure{
			Field: "answerable", Expected: strconv.FormatBool(testCase.ShouldAnswer), Actual: strconv.FormatBool(response.Answerable),
		})
	}

	actualSources := sourcePaths(response.Sources)
	if len(testCase.ExpectedSources) == 0 {
		if len(actualSources) > 0 {
			result.Failures = append(result.Failures, Failure{Field: "sources", Expected: "[]", Actual: formatList(actualSources)})
		}
	} else {
		for _, expected := range testCase.ExpectedSources {
			report.SourceExpected++
			if containsSource(actualSources, expected) {
				report.SourceHits++
				continue
			}
			result.Failures = append(result.Failures, Failure{Field: "source", Expected: expected, Actual: formatList(actualSources)})
		}
	}

	actualHeadings := headings(response.Sources)
	for _, expected := range testCase.ExpectedHeadings {
		report.HeadingExpected++
		if containsText(actualHeadings, expected) {
			report.HeadingHits++
			continue
		}
		result.Failures = append(result.Failures, Failure{Field: "heading", Expected: expected, Actual: formatList(actualHeadings)})
	}

	answer := normalize(response.Answer)
	for _, expected := range testCase.AnswerKeywords {
		report.KeywordExpected++
		if strings.Contains(answer, normalize(expected)) {
			report.KeywordHits++
			continue
		}
		result.Failures = append(result.Failures, Failure{Field: "keyword", Expected: expected, Actual: response.Answer})
	}
}

func sourcePaths(sources []Source) []string {
	paths := make([]string, 0, len(sources))
	for _, source := range sources {
		paths = append(paths, strings.TrimSpace(strings.ReplaceAll(source.SourcePath, "\\", "/")))
	}
	return paths
}

func containsSource(actual []string, expected string) bool {
	expected = path.Clean(strings.ReplaceAll(strings.TrimSpace(expected), "\\", "/"))
	for _, candidate := range actual {
		candidate = path.Clean(candidate)
		if strings.EqualFold(candidate, expected) || strings.HasSuffix(strings.ToLower(candidate), "/"+strings.ToLower(expected)) {
			return true
		}
	}
	return false
}

func headings(sources []Source) []string {
	var values []string
	for _, source := range sources {
		if strings.TrimSpace(source.Title) != "" {
			values = append(values, source.Title)
		}
		if strings.TrimSpace(source.DocumentTitle) != "" {
			values = append(values, source.DocumentTitle)
		}
		for _, heading := range source.HeadingPath {
			if strings.TrimSpace(heading) != "" {
				values = append(values, heading)
			}
		}
		if len(source.HeadingPath) > 0 {
			values = append(values, strings.Join(source.HeadingPath, " > "))
		}
	}
	return values
}

func containsText(actual []string, expected string) bool {
	expected = normalize(expected)
	for _, candidate := range actual {
		if strings.Contains(normalize(candidate), expected) {
			return true
		}
	}
	return false
}

func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func formatList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	return "[" + strings.Join(values, ", ") + "]"
}

// WriteText emits a stable human-readable report. Every failure includes the
// JSONL line, case id, scenario, field, expected value, and actual value.
func (report Report) WriteText(writer io.Writer) error {
	if writer == nil {
		return errors.New("report writer must not be nil")
	}
	if _, err := fmt.Fprintf(writer,
		"evaluation total=%d passed=%d failed=%d answerable=%d/%d refusal=%d/%d sources=%d/%d headings=%d/%d keywords=%d/%d\n",
		report.Total, report.Passed, report.Total-report.Passed,
		report.AnswerableCorrect, report.AnswerableExpected,
		report.RefusalCorrect, report.RefusalExpected,
		report.SourceHits, report.SourceExpected,
		report.HeadingHits, report.HeadingExpected,
		report.KeywordHits, report.KeywordExpected,
	); err != nil {
		return fmt.Errorf("write report summary: %w", err)
	}

	for _, result := range report.Results {
		status := "PASS"
		if !result.Passed() {
			status = "FAIL"
		}
		if _, err := fmt.Fprintf(writer, "%s line=%d id=%q scenario=%q\n", status, result.Case.Line, result.Case.ID, result.Case.Scenario); err != nil {
			return fmt.Errorf("write report case %q: %w", result.Case.ID, err)
		}
		failures := append([]Failure(nil), result.Failures...)
		sort.SliceStable(failures, func(i, j int) bool { return failures[i].Field < failures[j].Field })
		for _, failure := range failures {
			if _, err := fmt.Fprintf(writer, "  field=%s expected=%q actual=%q\n", failure.Field, failure.Expected, failure.Actual); err != nil {
				return fmt.Errorf("write report failure for case %q: %w", result.Case.ID, err)
			}
		}
	}
	return nil
}
