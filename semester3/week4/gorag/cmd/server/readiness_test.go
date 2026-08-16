package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		switch destination := destinations[i].(type) {
		case *bool:
			*destination = value.(bool)
		case *string:
			*destination = value.(string)
		}
	}
	return nil
}

type fakeDatabase struct {
	pingErr error
	rows    []fakeRow
	queries int
}

func (d *fakeDatabase) Ping(context.Context) error { return d.pingErr }
func (d *fakeDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	row := d.rows[d.queries]
	d.queries++
	return row
}

type fakeEmbedder struct {
	model     string
	dimension int
	vector    []float32
	err       error
}

func (e fakeEmbedder) EmbedQuery(context.Context, string) ([]float32, error) { return e.vector, e.err }
func (e fakeEmbedder) Model() string                                         { return e.model }
func (e fakeEmbedder) Dimension() int                                        { return e.dimension }

func TestDependencyCheckerChecksDatabaseModelsAndDimension(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/tags" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"embed:latest"},{"model":"answer"}]}`))
	}))
	defer ollama.Close()

	database := &fakeDatabase{rows: []fakeRow{{values: []any{true}}, {values: []any{"vector(3)"}}}}
	embedder := fakeEmbedder{model: "embed", dimension: 3, vector: make([]float32, 3)}
	checker, err := newDependencyChecker(database, ollama.Client(), ollama.Client(), ollama.URL, ollama.URL, embedder, "embed", "ollama", "answer", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestDependencyCheckerChecksOpenAICompatibleAnswerModel(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(goodModels()))
	}))
	defer ollama.Close()

	var gotPath, gotAuthorization string
	answer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		gotAuthorization = request.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-flash"}]}`))
	}))
	defer answer.Close()

	database := goodDatabase()
	embedder := goodEmbedder()
	checker, err := newDependencyChecker(database, ollama.Client(), answer.Client(), ollama.URL, answer.URL, embedder, "embed", "openai-compatible", "deepseek-v4-flash", "test-key", 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if gotPath != "/v1/models" {
		t.Fatalf("answer models path = %q, want /v1/models", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want Bearer test-key", gotAuthorization)
	}
}

func TestDependencyCheckerRejectsUnavailableOpenAICompatibleModel(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(goodModels()))
	}))
	defer ollama.Close()

	answer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"other-model"}]}`))
	}))
	defer answer.Close()

	checker, err := newDependencyChecker(goodDatabase(), ollama.Client(), answer.Client(), ollama.URL, answer.URL, goodEmbedder(), "embed", "openai-compatible", "deepseek-v4-flash", "", 3)
	if err != nil {
		t.Fatal(err)
	}
	err = checker.Check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("Check() error = %v, want unavailable model", err)
	}
}

func TestDependencyCheckerFailures(t *testing.T) {
	tests := []struct {
		name     string
		database *fakeDatabase
		embedder fakeEmbedder
		models   string
		want     string
	}{
		{name: "postgres", database: &fakeDatabase{pingErr: errors.New("down")}, want: "PostgreSQL"},
		{name: "extension", database: &fakeDatabase{rows: []fakeRow{{values: []any{false}}}}, want: "extension"},
		{name: "database dimension", database: &fakeDatabase{rows: []fakeRow{{values: []any{true}}, {values: []any{"vector(2)"}}}}, want: "vector(3)"},
		{name: "missing model", database: goodDatabase(), embedder: goodEmbedder(), models: `{"models":[]}`, want: "unavailable"},
		{name: "probe dimension", database: goodDatabase(), embedder: fakeEmbedder{model: "embed", dimension: 3, vector: make([]float32, 2)}, models: goodModels(), want: "dimension"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				payload := test.models
				if payload == "" {
					payload = goodModels()
				}
				_, _ = w.Write([]byte(payload))
			}))
			defer server.Close()
			checker, err := newDependencyChecker(test.database, server.Client(), server.Client(), server.URL, server.URL, test.embedder, "embed", "ollama", "answer", "", 3)
			if err != nil {
				t.Fatal(err)
			}
			err = checker.Check(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Check() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestWaitForDependenciesRetriesAndHonorsCancellation(t *testing.T) {
	calls := 0
	checker := checkFunc(func(context.Context) error {
		calls++
		if calls < 2 {
			return errors.New("not yet")
		}
		return nil
	})
	if err := waitForDependencies(context.Background(), checker, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := waitForDependencies(ctx, checkFunc(func(context.Context) error { return errors.New("down") }), time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want canceled", err)
	}
}

type checkFunc func(context.Context) error

func (f checkFunc) Check(ctx context.Context) error { return f(ctx) }

func goodDatabase() *fakeDatabase {
	return &fakeDatabase{rows: []fakeRow{{values: []any{true}}, {values: []any{"vector(3)"}}}}
}
func goodEmbedder() fakeEmbedder {
	return fakeEmbedder{model: "embed", dimension: 3, vector: make([]float32, 3)}
}
func goodModels() string {
	return `{"models":[{"name":"embed"},{"name":"answer"}]}`
}
