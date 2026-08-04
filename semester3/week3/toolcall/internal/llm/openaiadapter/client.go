package openaiadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"toolcall/internal/llm"
)

type Client struct {
	client openai.Client
	model  string
}

func New(apiKey, baseURL, model string) *Client {
	return newWithOptions(apiKey, baseURL, model)
}

func newWithOptions(apiKey, baseURL, model string, extra ...option.RequestOption) *Client {
	options := []option.RequestOption{
		option.WithAPIKey(apiKey),
		// Runtime owns retry policy so the SDK must not add hidden retries.
		option.WithMaxRetries(0),
	}
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}
	options = append(options, extra...)
	return &Client{client: openai.NewClient(options...), model: model}
}

func (c *Client) Complete(ctx context.Context, request llm.Request) (llm.Response, error) {
	messages, err := convertMessages(request.Messages)
	if err != nil {
		return llm.Response{}, &llm.Error{Kind: llm.ErrorInvalidData, Err: err}
	}
	tools := make([]openai.ChatCompletionToolUnionParam, 0, len(request.Tools))
	for _, spec := range request.Tools {
		tools = append(tools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        spec.Name,
			Description: openai.String(spec.Description),
			Parameters:  openai.FunctionParameters(spec.Schema),
		}))
	}

	completion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:             c.model,
		Messages:          messages,
		Tools:             tools,
		ParallelToolCalls: openai.Bool(false),
	})
	if err != nil {
		return llm.Response{}, classify(err)
	}
	if len(completion.Choices) == 0 {
		return llm.Response{}, &llm.Error{Kind: llm.ErrorInvalidData, Err: errors.New("completion has no choices")}
	}
	message := completion.Choices[0].Message
	if message.Refusal != "" {
		return llm.Response{}, &llm.Error{Kind: llm.ErrorPermanent, Err: errors.New("model refused the request")}
	}
	result := llm.Message{Role: llm.RoleAssistant, Content: message.Content}
	for _, call := range message.ToolCalls {
		if call.Type != "function" {
			return llm.Response{}, &llm.Error{Kind: llm.ErrorInvalidData, Err: fmt.Errorf("unsupported tool call type %q", call.Type)}
		}
		result.ToolCalls = append(result.ToolCalls, llm.ToolCall{
			ID: call.ID, Name: call.Function.Name, Arguments: json.RawMessage(call.Function.Arguments),
		})
	}
	return llm.Response{Message: result}, nil
}

func convertMessages(messages []llm.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, message := range messages {
		switch message.Role {
		case llm.RoleSystem:
			result = append(result, openai.SystemMessage(message.Content))
		case llm.RoleUser:
			result = append(result, openai.UserMessage(message.Content))
		case llm.RoleTool:
			result = append(result, openai.ToolMessage(message.Content, message.ToolCallID))
		case llm.RoleAssistant:
			assistant := openai.ChatCompletionAssistantMessageParam{}
			if message.Content != "" {
				assistant.Content.OfString = openai.String(message.Content)
			}
			for _, call := range message.ToolCalls {
				assistant.ToolCalls = append(assistant.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: call.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name: call.Name, Arguments: string(call.Arguments),
						},
					},
				})
			}
			result = append(result, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
		default:
			return nil, fmt.Errorf("unsupported message role %q", message.Role)
		}
	}
	return result, nil
}

func classify(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &llm.Error{Kind: llm.ErrorTemporary, Retryable: true, Err: err}
	}
	if errors.Is(err, context.Canceled) {
		return &llm.Error{Kind: llm.ErrorPermanent, Err: err}
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		retryable := apiErr.StatusCode == http.StatusRequestTimeout || apiErr.StatusCode == http.StatusConflict ||
			apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= http.StatusInternalServerError
		kind := llm.ErrorPermanent
		if retryable {
			kind = llm.ErrorTemporary
		}
		return &llm.Error{Kind: kind, Retryable: retryable, Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &llm.Error{Kind: llm.ErrorTemporary, Retryable: true, Err: err}
	}
	return &llm.Error{Kind: llm.ErrorPermanent, Err: err}
}
