package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorag/internal/httpurl"
)

const (
	DefaultModel     = "qwen3-embedding:0.6b"
	VectorDimension  = 1024
	defaultBatchSize = 16
	defaultTimeout   = 30 * time.Second
	defaultRetries   = 2
	defaultRetryWait = 100 * time.Millisecond
)

// DefaultQueryInstruction is applied only to queries. Document text is sent
// byte-for-byte so document and query task semantics cannot be mixed up.
const DefaultQueryInstruction = "从后端学习知识库中检索可直接回答下列问题的片段："

// Config controls bounded calls to Ollama's /api/embed endpoint.
type Config struct {
	BaseURL          string
	Model            string
	BatchSize        int
	MaxConcurrency   int
	RequestTimeout   time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	QueryInstruction string
}

// DocumentInput carries non-sensitive identity used to make failures
// actionable. Text is deliberately excluded from returned errors.
type DocumentInput struct {
	SourcePath string
	ChunkIndex int
	Text       string
}

// Embedder is the stable boundary used by indexing and retrieval code.
type Embedder interface {
	EmbedDocuments(context.Context, []DocumentInput) ([][]float32, error)
	EmbedQuery(context.Context, string) ([]float32, error)
	Model() string
	Dimension() int
}

// Option customizes the client, primarily for deterministic tests.
type Option func(*Client)

// WithHTTPClient supplies an HTTP client. RequestTimeout is still enforced on
// each attempt through a derived context.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithRetryWait replaces sleeping between attempts.
func WithRetryWait(wait func(context.Context, time.Duration) error) Option {
	return func(c *Client) {
		if wait != nil {
			c.wait = wait
		}
	}
}

type Client struct {
	baseURL          *url.URL
	model            string
	batchSize        int
	maxConcurrency   int
	requestTimeout   time.Duration
	maxRetries       int
	retryDelay       time.Duration
	queryInstruction string
	httpClient       *http.Client
	wait             func(context.Context, time.Duration) error
	requestSlots     chan struct{}
}

func NewClient(config Config, options ...Option) (*Client, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://127.0.0.1:11434"
	}
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("embedding: invalid Ollama base URL %q", config.BaseURL)
	}
	if config.Model == "" {
		config.Model = DefaultModel
	}
	if config.Model != DefaultModel {
		return nil, fmt.Errorf("embedding: model must be %q, got %q", DefaultModel, config.Model)
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaultBatchSize
	}
	if config.BatchSize < 0 {
		return nil, errors.New("embedding: batch size must be positive")
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 1
	}
	if config.MaxConcurrency < 0 {
		return nil, errors.New("embedding: max concurrency must be positive")
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = defaultTimeout
	}
	if config.RequestTimeout < 0 {
		return nil, errors.New("embedding: request timeout must be positive")
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultRetries
	}
	if config.MaxRetries < 0 {
		return nil, errors.New("embedding: max retries cannot be negative")
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = defaultRetryWait
	}
	if config.RetryDelay < 0 {
		return nil, errors.New("embedding: retry delay cannot be negative")
	}
	if config.QueryInstruction == "" {
		config.QueryInstruction = DefaultQueryInstruction
	}

	client := &Client{
		baseURL:          baseURL,
		model:            config.Model,
		batchSize:        config.BatchSize,
		maxConcurrency:   config.MaxConcurrency,
		requestTimeout:   config.RequestTimeout,
		maxRetries:       config.MaxRetries,
		retryDelay:       config.RetryDelay,
		queryInstruction: config.QueryInstruction,
		httpClient:       &http.Client{},
		wait:             waitContext,
		requestSlots:     make(chan struct{}, config.MaxConcurrency),
	}
	for _, option := range options {
		option(client)
	}
	return client, nil
}

func (c *Client) Model() string { return c.model }

func (c *Client) Dimension() int { return VectorDimension }

func (c *Client) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("embedding: query is empty")
	}
	input := c.queryInstruction + "\n" + query
	vectors, err := c.embedBatch(ctx, []string{input})
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}
	return vectors[0], nil
}

func (c *Client) EmbedDocuments(ctx context.Context, inputs []DocumentInput) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}
	for i, input := range inputs {
		if strings.TrimSpace(input.Text) == "" {
			return nil, fmt.Errorf("embedding document %q chunk %d (input %d): text is empty", input.SourcePath, input.ChunkIndex, i)
		}
	}

	results := make([][]float32, len(inputs))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type batchFailure struct {
		start int
		end   int
		err   error
	}
	batchCount := (len(inputs) + c.batchSize - 1) / c.batchSize
	jobs := make(chan int)
	failures := make(chan batchFailure, batchCount)
	var workers sync.WaitGroup
	workerCount := min(c.maxConcurrency, batchCount)

	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for start := range jobs {
				end := min(start+c.batchSize, len(inputs))
				texts := make([]string, end-start)
				for i := start; i < end; i++ {
					texts[i-start] = inputs[i].Text
				}
				vectors, err := c.embedBatch(ctx, texts)
				if err != nil {
					failures <- batchFailure{start: start, end: end, err: err}
					cancel()
					continue
				}
				copy(results[start:end], vectors)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for start := 0; start < len(inputs); start += c.batchSize {
			select {
			case jobs <- start:
			case <-ctx.Done():
				return
			}
		}
	}()
	workers.Wait()
	close(failures)

	for failure := range failures {
		first := inputs[failure.start]
		last := inputs[failure.end-1]
		return nil, fmt.Errorf(
			"embedding documents batch [%d:%d], first=%q chunk=%d, last=%q chunk=%d: %w",
			failure.start, failure.end, first.SourcePath, first.ChunkIndex,
			last.SourcePath, last.ChunkIndex, failure.err,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *Client) embedBatch(ctx context.Context, inputs []string) ([][]float32, error) {
	payload, err := json.Marshal(embedRequest{Model: c.model, Input: inputs})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vectors, retry, err := c.doEmbedLimited(ctx, payload, len(inputs))
		if err == nil {
			return vectors, nil
		}
		lastErr = err
		if !retry || attempt == c.maxRetries {
			break
		}
		if err := c.wait(ctx, c.retryDelay); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("Ollama embed failed after at most %d retries: %w", c.maxRetries, lastErr)
}

// doEmbedLimited applies one process-local limit across every request made by
// this client, including concurrent document batches and query embeddings.
// Retry backoff deliberately happens outside the slot so healthy work can
// continue while a failed request waits.
func (c *Client) doEmbedLimited(ctx context.Context, payload []byte, expected int) ([][]float32, bool, error) {
	select {
	case c.requestSlots <- struct{}{}:
		defer func() { <-c.requestSlots }()
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
	return c.doEmbed(ctx, payload, expected)
}

func (c *Client) doEmbed(ctx context.Context, payload []byte, expected int) ([][]float32, bool, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	endpoint := httpurl.JoinPath(c.baseURL, "api/embed")
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		if attemptCtx.Err() != nil {
			return nil, !errors.Is(ctx.Err(), context.Canceled), attemptCtx.Err()
		}
		return nil, true, fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		retry := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		return nil, retry, fmt.Errorf("Ollama returned HTTP %d", response.StatusCode)
	}

	limited := io.LimitReader(response.Body, 64<<20)
	var decoded embedResponse
	if err := json.NewDecoder(limited).Decode(&decoded); err != nil {
		return nil, true, fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Embeddings) != expected {
		return nil, false, fmt.Errorf("expected %d embeddings, got %d", expected, len(decoded.Embeddings))
	}
	for i, vector := range decoded.Embeddings {
		if len(vector) != VectorDimension {
			return nil, false, fmt.Errorf("embedding %d has dimension %d, want %d", i, len(vector), VectorDimension)
		}
	}
	return decoded.Embeddings, false, nil
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
