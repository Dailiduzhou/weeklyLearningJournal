package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorag/internal/rag"
)

type answererFunc func(context.Context, string, rag.Filter) (rag.Answer, error)

func (f answererFunc) AnswerQuestion(ctx context.Context, question string, filter rag.Filter) (rag.Answer, error) {
	return f(ctx, question, filter)
}

type checkerFunc func(context.Context) error

func (f checkerFunc) Check(ctx context.Context) error { return f(ctx) }

func TestQuestionHandlerSuccess(t *testing.T) {
	answerer := answererFunc(func(_ context.Context, question string, _ rag.Filter) (rag.Answer, error) {
		if question != "问题" {
			t.Fatalf("question = %q", question)
		}
		return rag.Answer{Answerable: true, Text: "答案 [S1]", Sources: []rag.AnswerSource{{ID: "S1"}}}, nil
	})
	handler := NewHandler(answerer, checkerFunc(func(context.Context) error { return nil }), nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"question":"  问题  "}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var got rag.Answer
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Answerable || got.Text != "答案 [S1]" || len(got.Sources) != 1 {
		t.Fatalf("response = %#v", got)
	}
}

func TestQuestionHandlerValidation(t *testing.T) {
	called := false
	answerer := answererFunc(func(context.Context, string, rag.Filter) (rag.Answer, error) {
		called = true
		return rag.Answer{}, nil
	})
	tests := []struct {
		name   string
		body   string
		limit  int64
		status int
	}{
		{name: "malformed", body: `{"question":`, status: http.StatusBadRequest},
		{name: "unknown field", body: `{"question":"q","secret":true}`, status: http.StatusBadRequest},
		{name: "multiple values", body: `{"question":"q"}{}`, status: http.StatusBadRequest},
		{name: "empty", body: `{"question":"  "}`, status: http.StatusBadRequest},
		{name: "oversized", body: `{"question":"far too long"}`, limit: 8, status: http.StatusRequestEntityTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandlerWithConfig(answerer, nil, nil, HandlerConfig{MaxRequestBytes: test.limit})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, test.status, response.Body.String())
			}
		})
	}
	if called {
		t.Fatal("answerer called for invalid request")
	}
}

func TestQuestionHandlerDependencyFailureReturnsSafeRefusal(t *testing.T) {
	answerer := answererFunc(func(context.Context, string, rag.Filter) (rag.Answer, error) {
		return rag.Answer{}, errors.New("postgres password secret")
	})
	handler := NewHandler(answerer, nil, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"question":"q"}`)))
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "secret") || !strings.Contains(response.Body.String(), rag.RefusalAnswer) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestQuestionHandlerCancelledContext(t *testing.T) {
	answerer := answererFunc(func(ctx context.Context, _ string, _ rag.Filter) (rag.Answer, error) {
		return rag.Answer{}, ctx.Err()
	})
	handler := NewHandler(answerer, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(`{"question":"q"}`)).WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestTimeout {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestHealthAndReadinessHandlers(t *testing.T) {
	readyErr := errors.New("dependency down")
	handler := NewHandler(nil, checkerFunc(func(context.Context) error { return readyErr }), nil)

	for _, test := range []struct {
		path   string
		status int
	}{
		{path: "/healthz", status: http.StatusOK},
		{path: "/readyz", status: http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s status = %d, body = %s", test.path, response.Code, response.Body.String())
		}
	}

	ready := NewHandler(nil, checkerFunc(func(context.Context) error { return nil }), nil)
	response := httptest.NewRecorder()
	ready.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d", response.Code)
	}
}

func TestQuestionMethodNotAllowed(t *testing.T) {
	handler := NewHandler(nil, nil, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/questions", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", response.Code)
	}
}

func TestQuestionHandlerAcceptsAndForwardsMetadataFilter(t *testing.T) {
	var received rag.Filter
	answerer := answererFunc(func(_ context.Context, question string, filter rag.Filter) (rag.Answer, error) {
		received = filter
		return rag.Answer{Answerable: true, Text: "答案 [S1]", Sources: []rag.AnswerSource{{ID: "S1"}}}, nil
	})
	handler := NewHandler(answerer, checkerFunc(func(context.Context) error { return nil }), nil)
	body := `{"question":"bff 的加密优化","filter":{"metadata":{"category":"ccnubox","module":"bff"},"tags":["ccnubox/bff"]}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if received.Metadata["category"] != "ccnubox" || received.Metadata["module"] != "bff" ||
		len(received.Tags) != 1 || received.Tags[0] != "ccnubox/bff" {
		t.Fatalf("forwarded filter = %#v, want the decoded request filter", received)
	}

	// Absent and null filters decode to the empty filter and pass validation.
	for _, body := range []string{`{"question":"问题"}`, `{"question":"问题","filter":null}`} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/questions", strings.NewReader(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("body %s: status = %d", body, response.Code)
		}
		if !received.Empty() {
			t.Fatalf("body %s: filter = %#v, want empty", body, received)
		}
	}
}

func TestQuestionHandlerRejectsInvalidMetadataFilter(t *testing.T) {
	called := false
	answerer := answererFunc(func(context.Context, string, rag.Filter) (rag.Answer, error) {
		called = true
		return rag.Answer{}, nil
	})
	handler := NewHandler(answerer, checkerFunc(func(context.Context) error { return nil }), nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/questions",
		strings.NewReader(`{"question":"问题","filter":{"metadata":{"category":" "}}}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "invalid_filter") || called {
		t.Fatalf("response = %s, answerer called = %v, want invalid_filter without calling the answerer", response.Body.String(), called)
	}

	// Unknown filter fields are still rejected by DisallowUnknownFields.
	request = httptest.NewRequest(http.MethodPost, "/api/v1/questions",
		strings.NewReader(`{"question":"问题","filter":{"metadata":{"category":"ccnubox"},"source_path":"api/auth.md"}}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_json") {
		t.Fatalf("status = %d, body = %s, want invalid_json for unknown filter fields", response.Code, response.Body.String())
	}
}
