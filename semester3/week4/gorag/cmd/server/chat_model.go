package main

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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/httpurl"
)

type httpChatModel struct {
	provider   string
	baseURL    *url.URL
	modelName  string
	apiKey     string
	httpClient *http.Client
}

var _ model.BaseChatModel = (*httpChatModel)(nil)

func newHTTPChatModel(provider, baseURL, modelName, apiKey string, client *http.Client) (*httpChatModel, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("server: answer model base URL must be absolute")
	}
	if provider != "ollama" && provider != "openai-compatible" {
		return nil, fmt.Errorf("server: unsupported answer provider %q", provider)
	}
	if strings.TrimSpace(modelName) == "" {
		return nil, errors.New("server: answer model name is empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &httpChatModel{provider: provider, baseURL: parsed, modelName: modelName, apiKey: apiKey, httpClient: client}, nil
}

func (m *httpChatModel) Generate(ctx context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, errors.New("answer model: messages are empty")
	}
	payload, endpoint, err := m.request(messages)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("answer model: create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
	response, err := m.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("answer model: send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return nil, fmt.Errorf("answer model: HTTP %d", response.StatusCode)
	}

	content, err := m.decode(response.Body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("answer model: response content is empty")
	}
	return &schema.Message{Role: schema.Assistant, Content: content}, nil
}

func (m *httpChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("answer model: streaming is not supported")
}

func (m *httpChatModel) request(messages []*schema.Message) ([]byte, string, error) {
	type wireMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	wireMessages := make([]wireMessage, 0, len(messages))
	for i, message := range messages {
		if message == nil {
			return nil, "", fmt.Errorf("answer model: message %d is nil", i)
		}
		wireMessages = append(wireMessages, wireMessage{Role: string(message.Role), Content: message.Content})
	}
	payload := struct {
		Model    string        `json:"model"`
		Messages []wireMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}{Model: m.modelName, Messages: wireMessages, Stream: false}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("answer model: encode request: %w", err)
	}
	return encoded, m.endpoint(), nil
}

func (m *httpChatModel) endpoint() string {
	if m.provider == "ollama" {
		return httpurl.JoinPath(m.baseURL, "api/chat").String()
	}
	return httpurl.OpenAICompatible(m.baseURL, "chat/completions").String()
}

func (m *httpChatModel) decode(reader io.Reader) (string, error) {
	limited := io.LimitReader(reader, 8<<20)
	if m.provider == "ollama" {
		var response struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.NewDecoder(limited).Decode(&response); err != nil {
			return "", fmt.Errorf("answer model: decode Ollama response: %w", err)
		}
		return response.Message.Content, nil
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(limited).Decode(&response); err != nil {
		return "", fmt.Errorf("answer model: decode OpenAI-compatible response: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", errors.New("answer model: response has no choices")
	}
	return response.Choices[0].Message.Content, nil
}
