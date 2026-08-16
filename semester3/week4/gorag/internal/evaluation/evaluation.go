// Package evaluation loads and executes the knowledge-base question set.
// It deliberately depends only on the public question/answer contract so the
// same runner can be used with an in-process fake or a running HTTP server.
package evaluation

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const maxJSONLLineBytes = 1024 * 1024

// RequiredScenarios are the negative, paraphrase, and lifecycle cases called
// out by the project evaluation plan.
var RequiredScenarios = []string{
	"out_of_scope",
	"paraphrase",
	"ambiguous",
	"cross_section",
	"deleted_document",
	"reindex",
}

// Case is one line in testdata/evaluation.jsonl. Line is populated by the
// loader and is used only to make failures easy to locate.
type Case struct {
	ID               string   `json:"id"`
	Question         string   `json:"question"`
	ShouldAnswer     bool     `json:"should_answer"`
	ExpectedSources  []string `json:"expected_sources"`
	ExpectedHeadings []string `json:"expected_headings"`
	AnswerKeywords   []string `json:"answer_keywords"`
	Scenario         string   `json:"scenario"`
	Line             int      `json:"-"`
}

type wireCase struct {
	ID               string    `json:"id"`
	Question         string    `json:"question"`
	ShouldAnswer     *bool     `json:"should_answer"`
	ExpectedSources  *[]string `json:"expected_sources"`
	ExpectedHeadings *[]string `json:"expected_headings"`
	AnswerKeywords   *[]string `json:"answer_keywords"`
	Scenario         string    `json:"scenario"`
}

// LoadFile opens a JSONL evaluation set and returns fully validated cases.
func LoadFile(path string) ([]Case, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open evaluation dataset %q: %w", path, err)
	}
	defer file.Close()

	cases, err := LoadJSONL(file)
	if err != nil {
		return nil, fmt.Errorf("load evaluation dataset %q: %w", path, err)
	}
	return cases, nil
}

// LoadJSONL decodes one strict JSON object per non-empty line. Unknown fields
// are rejected so misspelled expectations cannot silently weaken evaluation.
func LoadJSONL(reader io.Reader) ([]Case, error) {
	if reader == nil {
		return nil, errors.New("load JSONL: reader must not be nil")
	}

	var cases []Case
	seen := make(map[string]int)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxJSONLLineBytes)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		var wire wireCase
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&wire); err != nil {
			return nil, fmt.Errorf("line %d: decode JSON object: %w", line, err)
		}
		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			if err == nil {
				err = errors.New("multiple JSON values on one line")
			}
			return nil, fmt.Errorf("line %d: %w", line, err)
		}

		testCase, err := wire.toCase(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if firstLine, exists := seen[testCase.ID]; exists {
			return nil, fmt.Errorf("line %d: duplicate id %q (first declared on line %d)", line, testCase.ID, firstLine)
		}
		seen[testCase.ID] = line
		cases = append(cases, testCase)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan JSONL near line %d: %w", line+1, err)
	}
	if len(cases) == 0 {
		return nil, errors.New("evaluation dataset is empty")
	}
	return cases, nil
}

func (wire wireCase) toCase(line int) (Case, error) {
	var missing []string
	if wire.ShouldAnswer == nil {
		missing = append(missing, "should_answer")
	}
	if wire.ExpectedSources == nil {
		missing = append(missing, "expected_sources")
	}
	if wire.ExpectedHeadings == nil {
		missing = append(missing, "expected_headings")
	}
	if wire.AnswerKeywords == nil {
		missing = append(missing, "answer_keywords")
	}
	if len(missing) > 0 {
		return Case{}, fmt.Errorf("missing required field(s): %s", strings.Join(missing, ", "))
	}

	testCase := Case{
		ID:               strings.TrimSpace(wire.ID),
		Question:         strings.TrimSpace(wire.Question),
		ShouldAnswer:     *wire.ShouldAnswer,
		ExpectedSources:  trimStrings(*wire.ExpectedSources),
		ExpectedHeadings: trimStrings(*wire.ExpectedHeadings),
		AnswerKeywords:   trimStrings(*wire.AnswerKeywords),
		Scenario:         strings.TrimSpace(wire.Scenario),
		Line:             line,
	}
	if err := testCase.validate(); err != nil {
		return Case{}, err
	}
	return testCase, nil
}

func (testCase Case) validate() error {
	var problems []error
	if testCase.ID == "" {
		problems = append(problems, errors.New("id must not be empty"))
	}
	if testCase.Question == "" {
		problems = append(problems, errors.New("question must not be empty"))
	}
	if testCase.Scenario == "" {
		problems = append(problems, errors.New("scenario must not be empty"))
	}
	if testCase.ShouldAnswer {
		if len(testCase.ExpectedSources) == 0 {
			problems = append(problems, errors.New("answerable case needs expected_sources"))
		}
		if len(testCase.ExpectedHeadings) == 0 {
			problems = append(problems, errors.New("answerable case needs expected_headings"))
		}
		if len(testCase.AnswerKeywords) == 0 {
			problems = append(problems, errors.New("answerable case needs answer_keywords"))
		}
	}
	for field, values := range map[string][]string{
		"expected_sources":  testCase.ExpectedSources,
		"expected_headings": testCase.ExpectedHeadings,
		"answer_keywords":   testCase.AnswerKeywords,
	} {
		for index, value := range values {
			if value == "" {
				problems = append(problems, fmt.Errorf("%s[%d] must not be empty", field, index))
			}
		}
	}
	return errors.Join(problems...)
}

// ValidateCoverage checks the dataset-level acceptance criteria.
func ValidateCoverage(cases []Case) error {
	answerable := 0
	scenarios := make(map[string]bool)
	for _, testCase := range cases {
		if err := testCase.validate(); err != nil {
			return fmt.Errorf("case %q: %w", testCase.ID, err)
		}
		if testCase.ShouldAnswer {
			answerable++
		}
		scenarios[testCase.Scenario] = true
	}

	var problems []error
	if answerable < 5 {
		problems = append(problems, fmt.Errorf("need at least five answerable knowledge-base questions, got %d", answerable))
	}
	for _, scenario := range RequiredScenarios {
		if !scenarios[scenario] {
			problems = append(problems, fmt.Errorf("missing required scenario %q", scenario))
		}
	}
	return errors.Join(problems...)
}

func trimStrings(values []string) []string {
	trimmed := make([]string, len(values))
	for index, value := range values {
		trimmed[index] = strings.TrimSpace(value)
	}
	return trimmed
}
