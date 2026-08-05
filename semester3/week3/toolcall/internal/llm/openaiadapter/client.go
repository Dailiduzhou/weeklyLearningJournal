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
	"github.com/openai/openai-go/v3/responses"
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
	tools := make([]responses.ToolUnionParam, 0, len(request.Tools))
	for _, spec := range request.Tools {
		tool := responses.ToolParamOfFunction(spec.Name, spec.Schema, false)
		tool.OfFunction.Description = openai.String(spec.Description)
		tools = append(tools, tool)
	}

	response, err := c.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:             c.model,
		Input:             responses.ResponseNewParamsInputUnion{OfInputItemList: messages},
		Tools:             tools,
		ParallelToolCalls: openai.Bool(false),
	})
	if err != nil {
		return llm.Response{}, classify(err)
	}
	if response.Status != "" && response.Status != responses.ResponseStatusCompleted {
		detail := string(response.Status)
		if response.Error.Message != "" {
			detail += ": " + response.Error.Message
		} else if response.IncompleteDetails.Reason != "" {
			detail += ": " + response.IncompleteDetails.Reason
		}
		return llm.Response{}, &llm.Error{Kind: llm.ErrorInvalidData, Err: fmt.Errorf("response was not completed: %s", detail)}
	}

	result := llm.Message{Role: llm.RoleAssistant, Content: response.OutputText()}
	for _, item := range response.Output {
		switch item.Type {
		case "message":
			for _, content := range item.Content {
				if content.Type == "refusal" {
					return llm.Response{}, &llm.Error{Kind: llm.ErrorPermanent, Err: errors.New("model refused the request")}
				}
			}
		case "function_call":
			call := item.AsFunctionCall()
			result.ToolCalls = append(result.ToolCalls, llm.ToolCall{
				ID: call.CallID, Name: call.Name, Arguments: json.RawMessage(call.Arguments),
			})
		case "reasoning":
			// Reasoning items contain no user-visible content or locally executable call.
		default:
			return llm.Response{}, &llm.Error{Kind: llm.ErrorInvalidData, Err: fmt.Errorf("unsupported response output type %q", item.Type)}
		}
	}
	return llm.Response{Message: result}, nil
}

func convertMessages(messages []llm.Message) (responses.ResponseInputParam, error) {
	result := make(responses.ResponseInputParam, 0, len(messages))
	for _, message := range messages {
		switch message.Role {
		case llm.RoleSystem:
			result = append(result, responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRoleSystem))
		case llm.RoleUser:
			result = append(result, responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRoleUser))
		case llm.RoleTool:
			result = append(result, responses.ResponseInputItemParamOfFunctionCallOutput(message.ToolCallID, message.Content))
		case llm.RoleAssistant:
			if message.Content != "" || len(message.ToolCalls) == 0 {
				result = append(result, responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRoleAssistant))
			}
			for _, call := range message.ToolCalls {
				result = append(result, responses.ResponseInputItemParamOfFunctionCall(string(call.Arguments), call.ID, call.Name))
			}
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
