package evaluation

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
)

const maxHTTPResponseBytes = 1024 * 1024

// Doer is the subset of http.Client used by HTTPClient.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// HTTPClient invokes the configured POST question endpoint.
type HTTPClient struct {
	endpoint string
	client   Doer
}

// NewHTTPClient validates endpoint and builds an evaluation Client. A nil
// doer uses http.DefaultClient; callers should normally pass a client with an
// explicit Timeout for live evaluations.
func NewHTTPClient(endpoint string, doer Doer) (*HTTPClient, error) {
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("evaluation endpoint must be an absolute HTTP(S) URL, got %q", endpoint)
	}
	if doer == nil {
		doer = http.DefaultClient
	}
	return &HTTPClient{endpoint: endpoint, client: doer}, nil
}

// Ask implements Client using the project POST /api/v1/questions JSON schema.
func (client *HTTPClient) Ask(ctx context.Context, question string) (Response, error) {
	if client == nil || client.client == nil {
		return Response{}, errors.New("HTTP evaluation client is not initialized")
	}
	body, err := json.Marshal(struct {
		Question string `json:"question"`
	}{Question: question})
	if err != nil {
		return Response{}, fmt.Errorf("encode question request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("build question request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	httpResponse, err := client.client.Do(request)
	if err != nil {
		return Response{}, fmt.Errorf("call question endpoint: %w", err)
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return Response{}, fmt.Errorf("question endpoint returned HTTP status %d", httpResponse.StatusCode)
	}

	limited := io.LimitReader(httpResponse.Body, maxHTTPResponseBytes+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return Response{}, fmt.Errorf("read question response: %w", err)
	}
	if len(responseBody) > maxHTTPResponseBytes {
		return Response{}, fmt.Errorf("question response exceeds %d bytes", maxHTTPResponseBytes)
	}

	var response Response
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	if err := decoder.Decode(&response); err != nil {
		return Response{}, fmt.Errorf("decode question response: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Response{}, errors.New("decode question response: multiple JSON values")
	}
	return response, nil
}
