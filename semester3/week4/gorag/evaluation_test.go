package gorag_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"gorag/internal/evaluation"
)

const evaluationDataset = "testdata/evaluation.jsonl"

func TestEvaluationDataset(t *testing.T) {
	cases, err := evaluation.LoadFile(evaluationDataset)
	if err != nil {
		t.Fatalf("load evaluation dataset: %v", err)
	}
	if err := evaluation.ValidateCoverage(cases); err != nil {
		t.Fatalf("evaluation coverage: %v", err)
	}
}

func selectEvaluationScenarios(t *testing.T, cases []evaluation.Case, includeRaw, excludeRaw string) []evaluation.Case {
	t.Helper()
	include := scenarioSet(includeRaw)
	exclude := scenarioSet(excludeRaw)
	selected := make([]evaluation.Case, 0, len(cases))
	for _, testCase := range cases {
		if len(include) > 0 && !include[testCase.Scenario] {
			continue
		}
		if exclude[testCase.Scenario] {
			continue
		}
		selected = append(selected, testCase)
	}
	if len(selected) == 0 {
		t.Fatal("evaluation scenario filters selected no cases")
	}
	return selected
}

func scenarioSet(raw string) map[string]bool {
	result := make(map[string]bool)
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = true
		}
	}
	return result
}

// TestLiveEvaluation is opt-in so normal unit tests never depend on a running
// model or database. Point the variable at the complete question endpoint,
// for example http://localhost:8080/api/v1/questions.
func TestLiveEvaluation(t *testing.T) {
	endpoint := os.Getenv("GORAG_EVALUATION_ENDPOINT")
	if endpoint == "" {
		t.Skip("set GORAG_EVALUATION_ENDPOINT to run the live end-to-end evaluation")
	}

	cases, err := evaluation.LoadFile(evaluationDataset)
	if err != nil {
		t.Fatalf("load evaluation dataset: %v", err)
	}
	cases = selectEvaluationScenarios(t, cases,
		os.Getenv("GORAG_EVALUATION_INCLUDE_SCENARIOS"),
		os.Getenv("GORAG_EVALUATION_EXCLUDE_SCENARIOS"),
	)
	client, err := evaluation.NewHTTPClient(endpoint, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("create HTTP evaluation client: %v", err)
	}
	runner, err := evaluation.NewRunner(client)
	if err != nil {
		t.Fatalf("create evaluation runner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	report, err := runner.Run(ctx, cases)
	if err != nil {
		t.Fatalf("run live evaluation: %v", err)
	}
	var output bytes.Buffer
	if err := report.WriteText(&output); err != nil {
		t.Fatalf("format live evaluation report: %v", err)
	}
	t.Log("\n" + output.String())
	if !report.Successful() {
		t.Fatalf("live evaluation failed; see the per-case report above")
	}
}
