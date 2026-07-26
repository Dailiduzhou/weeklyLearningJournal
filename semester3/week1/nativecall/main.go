package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL       = "https://api.deepseek.com"
	model                = "deepseek-v4-flash"
	maxRetries           = 3
	maxResponseBodyBytes = 4 << 20
)

var errInvalidResponse = errors.New("model output does not match Response")

type Response struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Priority string   `json:"priority"`
	Tags     []string `json:"tags"`
}

var responseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title":    map[string]any{"type": "string"},
		"summary":  map[string]any{"type": "string"},
		"priority": map[string]any{"type": "string"},
		"tags": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required":             []string{"title", "summary", "priority", "tags"},
	"additionalProperties": false,
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	// Stream is deliberately serialized even when false. This is the native,
	// non-streaming Chat Completions path.
	Stream bool `json:"stream"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content *string `json:"content"`
			Refusal *string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
}

type apiClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	maxRetries int
	retryDelay func(attempt int) time.Duration
}

type clientConfig struct {
	apiKey     string
	baseURL    string
	maxRetries int
}

type httpStatusError struct {
	statusCode int
	status     string
	body       string
}

func (e *httpStatusError) Error() string {
	if e.body == "" {
		return "chat completion returned " + e.status
	}
	return fmt.Sprintf("chat completion returned %s: %s", e.status, e.body)
}

func main() {
	config, err := clientConfigFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := newAPIClient(http.DefaultClient, config)
	err = interactiveLoop(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		func(ctx context.Context, prompt string, output io.Writer) error {
			return createResponse(ctx, client, prompt, output)
		},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newAPIClient(httpClient *http.Client, config clientConfig) *apiClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &apiClient{
		httpClient: httpClient,
		apiKey:     config.apiKey,
		baseURL:    strings.TrimRight(config.baseURL, "/"),
		maxRetries: config.maxRetries,
		retryDelay: func(attempt int) time.Duration {
			if attempt > 5 {
				attempt = 5
			}
			return 100 * time.Millisecond * time.Duration(1<<attempt)
		},
	}
}

type promptHandler func(context.Context, string, io.Writer) error

func interactiveLoop(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	diagnostics io.Writer,
	handle promptHandler,
) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for {
		fmt.Fprint(diagnostics, "> ")
		if !scanner.Scan() {
			return scanner.Err()
		}

		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			continue
		}
		if strings.EqualFold(prompt, "quit") || strings.EqualFold(prompt, "exit") {
			return nil
		}

		if err := handle(ctx, prompt, output); err != nil {
			if errors.Is(err, errInvalidResponse) {
				continue
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			fmt.Fprintln(diagnostics, err)
		}
	}
}

func createResponse(ctx context.Context, client *apiClient, userPrompt string, output io.Writer) error {
	schemaJSON, err := json.Marshal(responseSchema)
	if err != nil {
		return fmt.Errorf("marshal response schema: %w", err)
	}
	systemPrompt := "Return only one JSON object that exactly matches the following JSON Schema. Do not use Markdown or add explanatory text.\nJSON Schema:\n" + string(schemaJSON)

	completion, err := client.createChatCompletion(ctx, chatCompletionRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: false,
	})
	if err != nil {
		return fmt.Errorf("create chat completion: %w", err)
	}
	if len(completion.Choices) == 0 {
		return errInvalidResponse
	}

	responseMessage := completion.Choices[0].Message
	if responseMessage.Refusal != nil && *responseMessage.Refusal != "" {
		return errInvalidResponse
	}
	if responseMessage.Content == nil {
		return errInvalidResponse
	}

	result, ok := decodeResponse(*responseMessage.Content)
	if !ok {
		return errInvalidResponse
	}
	if err := json.NewEncoder(output).Encode(result); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

func (c *apiClient) createChatCompletion(ctx context.Context, request chatCompletionRequest) (chatCompletionResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		completion, err := c.sendChatCompletion(ctx, payload)
		if err == nil {
			return completion, nil
		}
		if attempt >= c.maxRetries || !isRetryable(err) {
			return chatCompletionResponse{}, err
		}
		if err := waitForRetry(ctx, c.retryDelay(attempt)); err != nil {
			return chatCompletionResponse{}, err
		}
	}
}

func (c *apiClient) sendChatCompletion(ctx context.Context, payload []byte) (chatCompletionResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("send request: %w", err)
	}

	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return chatCompletionResponse{}, fmt.Errorf("read response: %w", readErr)
	}
	if closeErr != nil {
		return chatCompletionResponse{}, fmt.Errorf("close response: %w", closeErr)
	}
	if len(body) > maxResponseBodyBytes {
		return chatCompletionResponse{}, errors.New("chat completion response is too large")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return chatCompletionResponse{}, &httpStatusError{
			statusCode: response.StatusCode,
			status:     response.Status,
			body:       strings.TrimSpace(string(body)),
		}
	}

	var completion chatCompletionResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&completion); err != nil {
		return chatCompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return chatCompletionResponse{}, errors.New("decode response: trailing JSON content")
	}
	return completion, nil
}

func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusRequestTimeout ||
			statusErr.statusCode == http.StatusConflict ||
			statusErr.statusCode == http.StatusTooManyRequests ||
			statusErr.statusCode >= http.StatusInternalServerError
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func clientConfigFromEnv() (clientConfig, error) {
	return clientConfigWith(os.Getenv)
}

func clientConfigWith(getEnv func(string) string) (clientConfig, error) {
	apiKey := firstNonEmpty(getEnv, "OPENAI_API_KEY", "OPENAI_APIKEY")
	if apiKey == "" {
		return clientConfig{}, errors.New("OPENAI_API_KEY is required")
	}

	baseURL := firstNonEmpty(getEnv, "OPENAI_BASEURL", "OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return clientConfig{
		apiKey:     apiKey,
		baseURL:    baseURL,
		maxRetries: maxRetries,
	}, nil
}

func firstNonEmpty(getEnv func(string) string, names ...string) string {
	for _, name := range names {
		if value := getEnv(name); value != "" {
			return value
		}
	}
	return ""
}

func decodeResponse(raw string) (Response, bool) {
	var decoded struct {
		Title    *string    `json:"title"`
		Summary  *string    `json:"summary"`
		Priority *string    `json:"priority"`
		Tags     *[]*string `json:"tags"`
	}

	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return Response{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Response{}, false
	}
	if decoded.Title == nil || decoded.Summary == nil || decoded.Priority == nil || decoded.Tags == nil {
		return Response{}, false
	}

	tags := make([]string, len(*decoded.Tags))
	for i, tag := range *decoded.Tags {
		if tag == nil {
			return Response{}, false
		}
		tags[i] = *tag
	}

	return Response{
		Title:    *decoded.Title,
		Summary:  *decoded.Summary,
		Priority: *decoded.Priority,
		Tags:     tags,
	}, true
}
