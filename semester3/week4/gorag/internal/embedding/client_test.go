package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmbedDocumentsBatchesPreservesOrderAndText(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	received := make([]string, 0, 5)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body embedRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, body.Input...)
		mu.Unlock()
		vectors := make([][]float32, len(body.Input))
		for i, text := range body.Input {
			vectors[i] = testVector(float32(text[len(text)-1] - '0'))
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: vectors})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{BatchSize: 2, MaxConcurrency: 1})
	inputs := make([]DocumentInput, 5)
	for i := range inputs {
		inputs[i] = DocumentInput{SourcePath: "doc.md", ChunkIndex: i, Text: fmt.Sprintf("body-%d", i)}
	}
	vectors, err := client.EmbedDocuments(context.Background(), inputs)
	if err != nil {
		t.Fatalf("EmbedDocuments() error = %v", err)
	}
	if len(vectors) != len(inputs) {
		t.Fatalf("got %d vectors, want %d", len(vectors), len(inputs))
	}
	for i, vector := range vectors {
		if vector[0] != float32(i) {
			t.Errorf("vector %d marker = %v, want %d", i, vector[0], i)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(received, ",") != "body-0,body-1,body-2,body-3,body-4" {
		t.Fatalf("document input changed or reordered: %q", received)
	}
}

func TestEmbedQueryAddsInstruction(t *testing.T) {
	t.Parallel()

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body embedRequest
		_ = json.NewDecoder(request.Body).Decode(&body)
		got = body.Input[0]
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{testVector(1)}})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{QueryInstruction: "retrieve"})
	_, err := client.EmbedQuery(context.Background(), "how?")
	if err != nil {
		t.Fatalf("EmbedQuery() error = %v", err)
	}
	if got != "retrieve\nhow?" {
		t.Fatalf("query input = %q, want instruction and query", got)
	}
}

func TestEmbedEndpointPreservesBasePathAndQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/ollama/api/embed" {
			t.Fatalf("path = %q, want /ollama/api/embed", request.URL.Path)
		}
		if request.URL.Query().Get("token") != "secret" {
			t.Fatalf("query = %q, want token=secret", request.URL.RawQuery)
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{testVector(1)}})
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL: server.URL + "/ollama?token=secret", Model: DefaultModel,
		BatchSize: 1, MaxConcurrency: 1, RequestTimeout: time.Second,
		MaxRetries: 1, RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.EmbedQuery(context.Background(), "question"); err != nil {
		t.Fatalf("EmbedQuery() error = %v", err)
	}
}
func TestEmbedRetriesRetryableFailure(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(writer, "try later", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{testVector(1)}})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{MaxRetries: 2}, WithRetryWait(func(context.Context, time.Duration) error { return nil }))
	if _, err := client.EmbedQuery(context.Background(), "question"); err != nil {
		t.Fatalf("EmbedQuery() error = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestEmbedRejectsWrongDimensionWithoutRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{{1, 2}}})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{MaxRetries: 3})
	_, err := client.EmbedQuery(context.Background(), "question")
	if err == nil || !strings.Contains(err.Error(), "dimension 2, want 1024") {
		t.Fatalf("EmbedQuery() error = %v, want dimension error", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want non-retryable single call", calls.Load())
	}
}

func TestEmbedDocumentsLimitsConcurrency(t *testing.T) {
	t.Parallel()

	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		var body embedRequest
		_ = json.NewDecoder(request.Body).Decode(&body)
		time.Sleep(15 * time.Millisecond)
		vectors := make([][]float32, len(body.Input))
		for i := range vectors {
			vectors[i] = testVector(1)
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: vectors})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{BatchSize: 1, MaxConcurrency: 2})
	inputs := make([]DocumentInput, 8)
	for i := range inputs {
		inputs[i] = DocumentInput{SourcePath: "doc.md", ChunkIndex: i, Text: "text"}
	}
	if _, err := client.EmbedDocuments(context.Background(), inputs); err != nil {
		t.Fatalf("EmbedDocuments() error = %v", err)
	}
	if maximum.Load() > 2 {
		t.Fatalf("maximum concurrency = %d, want <= 2", maximum.Load())
	}
	if maximum.Load() < 2 {
		t.Fatalf("maximum concurrency = %d, concurrency was not exercised", maximum.Load())
	}
}

func TestEmbedClientLimitsConcurrencyAcrossCalls(t *testing.T) {
	t.Parallel()

	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		var body embedRequest
		_ = json.NewDecoder(request.Body).Decode(&body)
		time.Sleep(20 * time.Millisecond)
		vectors := make([][]float32, len(body.Input))
		for index := range vectors {
			vectors[index] = testVector(1)
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: vectors})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{BatchSize: 1, MaxConcurrency: 2})
	start := make(chan struct{})
	errors := make(chan error, 6)
	var calls sync.WaitGroup
	for index := range 6 {
		calls.Add(1)
		go func() {
			defer calls.Done()
			<-start
			if index%2 == 0 {
				_, err := client.EmbedQuery(context.Background(), fmt.Sprintf("question-%d", index))
				errors <- err
				return
			}
			_, err := client.EmbedDocuments(context.Background(), []DocumentInput{{
				SourcePath: fmt.Sprintf("doc-%d.md", index), ChunkIndex: 0, Text: "body",
			}})
			errors <- err
		}()
	}
	close(start)
	calls.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent embedding call error = %v", err)
		}
	}
	if maximum.Load() != 2 {
		t.Fatalf("maximum concurrency = %d, want 2 across all client calls", maximum.Load())
	}
}

func TestEmbedRetryWaitDoesNotHoldConcurrencySlot(t *testing.T) {
	t.Parallel()

	var retryCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body embedRequest
		_ = json.NewDecoder(request.Body).Decode(&body)
		if strings.HasSuffix(body.Input[0], "retry") && retryCalls.Add(1) == 1 {
			http.Error(writer, "try later", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{testVector(1)}})
	}))
	defer server.Close()

	waiting := make(chan struct{}, 1)
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeRetry := func() { resumeOnce.Do(func() { close(resume) }) }
	t.Cleanup(resumeRetry)
	client := newTestClient(t, server.URL, Config{MaxConcurrency: 1, MaxRetries: 1}, WithRetryWait(func(ctx context.Context, _ time.Duration) error {
		waiting <- struct{}{}
		select {
		case <-resume:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))
	retryDone := make(chan error, 1)
	go func() {
		_, err := client.EmbedQuery(context.Background(), "retry")
		retryDone <- err
	}()
	select {
	case <-waiting:
	case <-time.After(time.Second):
		t.Fatal("retry did not enter backoff")
	}

	healthyCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := client.EmbedQuery(healthyCtx, "healthy"); err != nil {
		t.Fatalf("healthy query was blocked by retry backoff: %v", err)
	}
	resumeRetry()
	if err := <-retryDone; err != nil {
		t.Fatalf("retried query error = %v", err)
	}
}

func TestEmbedCancellationWhileWaitingForConcurrencySlot(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseFirst)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		started <- struct{}{}
		<-release
		_ = json.NewEncoder(writer).Encode(embedResponse{Embeddings: [][]float32{testVector(1)}})
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, Config{MaxConcurrency: 1})

	firstDone := make(chan error, 1)
	go func() {
		_, err := client.EmbedQuery(context.Background(), "first")
		firstDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first query did not acquire concurrency slot")
	}

	waitingCtx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() {
		_, err := client.EmbedQuery(waitingCtx, "second")
		secondDone <- err
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	if err := <-secondDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting query error = %v, want context.Canceled", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, canceled waiter should not send a request", requests.Load())
	}
	releaseFirst()
	if err := <-firstDone; err != nil {
		t.Fatalf("first query error = %v", err)
	}
}

func TestEmbedPropagatesRequestTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{RequestTimeout: 5 * time.Millisecond, MaxRetries: 1}, WithRetryWait(func(context.Context, time.Duration) error { return nil }))
	_, err := client.EmbedQuery(context.Background(), "question")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("EmbedQuery() error = %v, want deadline exceeded", err)
	}
}

func TestDocumentFailureIncludesIdentityButNotText(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, Config{})
	_, err := client.EmbedDocuments(context.Background(), []DocumentInput{{
		SourcePath: "notes/doc.md", ChunkIndex: 7, Text: "TOP SECRET TEXT",
	}})
	if err == nil {
		t.Fatal("EmbedDocuments() error = nil")
	}
	if !strings.Contains(err.Error(), "notes/doc.md") || !strings.Contains(err.Error(), "chunk=7") {
		t.Fatalf("error lacks input identity: %v", err)
	}
	if strings.Contains(err.Error(), "TOP SECRET TEXT") {
		t.Fatalf("error leaked document text: %v", err)
	}
}

func newTestClient(t *testing.T, baseURL string, override Config, options ...Option) *Client {
	t.Helper()
	config := Config{
		BaseURL:          baseURL,
		Model:            DefaultModel,
		BatchSize:        4,
		MaxConcurrency:   1,
		RequestTimeout:   time.Second,
		MaxRetries:       1,
		RetryDelay:       time.Millisecond,
		QueryInstruction: "retrieve",
	}
	if override.BatchSize != 0 {
		config.BatchSize = override.BatchSize
	}
	if override.MaxConcurrency != 0 {
		config.MaxConcurrency = override.MaxConcurrency
	}
	if override.RequestTimeout != 0 {
		config.RequestTimeout = override.RequestTimeout
	}
	if override.MaxRetries != 0 {
		config.MaxRetries = override.MaxRetries
	}
	if override.RetryDelay != 0 {
		config.RetryDelay = override.RetryDelay
	}
	if override.QueryInstruction != "" {
		config.QueryInstruction = override.QueryInstruction
	}
	client, err := NewClient(config, options...)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func testVector(marker float32) []float32 {
	vector := make([]float32, VectorDimension)
	vector[0] = marker
	return vector
}
