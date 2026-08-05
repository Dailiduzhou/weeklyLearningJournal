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

func TestResponsesToolCallingConversion(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, ok := body["tools"].([]any)
		if body["parallel_tool_calls"] != false || !ok || len(tools) != 1 {
			t.Errorf("unexpected tool declaration: %#v", body)
		}
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" || tool["name"] != "calculator" || tool["strict"] != false {
			t.Errorf("unexpected function tool: %#v", tool)
		}
		input := body["input"].([]any)
		if len(input) != 1 || input[0].(map[string]any)["role"] != "user" {
			t.Errorf("unexpected input: %#v", input)
		}
		response := `{"id":"resp-test","object":"response","created_at":1,"model":"test-model","status":"completed","output":[{"id":"msg-1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"I'll calculate.","annotations":[]}]},{"id":"fc-1","type":"function_call","call_id":"call-1","name":"calculator","arguments":"{\"expression\":\"1+1\"}","status":"completed"}]}`
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
	if response.Message.Content != "I'll calculate." || len(response.Message.ToolCalls) != 1 || response.Message.ToolCalls[0].Name != "calculator" || string(response.Message.ToolCalls[0].Arguments) != `{"expression":"1+1"}` {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestResponsesHistoryConversion(t *testing.T) {
	items, err := convertMessages([]llm.Message{
		{Role: llm.RoleSystem, Content: "be concise"},
		{Role: llm.RoleUser, Content: "calculate"},
		{Role: llm.RoleAssistant, Content: "working", ToolCalls: []llm.ToolCall{{ID: "call-1", Name: "calculator", Arguments: json.RawMessage(`{"expression":"1+1"}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-1", Content: `{"result":2}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	wantParts := []string{
		`"role":"system"`, `"role":"user"`, `"role":"assistant"`,
		`"type":"function_call"`, `"call_id":"call-1"`, `"name":"calculator"`,
		`"type":"function_call_output"`, `"output":"{\"result\":2}"`,
	}
	for _, want := range wantParts {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("converted input %s does not contain %s", encoded, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
