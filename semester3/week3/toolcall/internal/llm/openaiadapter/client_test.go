package openaiadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3/option"
	"toolcall/internal/llm"
)

func TestChatCompletionToolCallingConversion(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["parallel_tool_calls"] != false || len(body["tools"].([]any)) != 1 {
			t.Errorf("unexpected tool declaration: %#v", body)
		}
		response := `{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"calculator","arguments":"{\"expression\":\"1+1\"}"}}]},"finish_reason":"tool_calls"}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
	})

	client := newWithOptions("test-key", "http://openai.test", "test-model", option.WithHTTPClient(&http.Client{Transport: transport}))
	response, err := client.Complete(context.Background(), llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "calculate"}},
		Tools: []llm.ToolSpec{{Name: "calculator", Description: "calculate", Schema: map[string]any{
			"type": "object", "properties": map[string]any{"expression": map[string]any{"type": "string"}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Message.ToolCalls) != 1 || response.Message.ToolCalls[0].Name != "calculator" || string(response.Message.ToolCalls[0].Arguments) != `{"expression":"1+1"}` {
		t.Fatalf("unexpected response: %+v", response)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
