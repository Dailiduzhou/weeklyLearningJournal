package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func TestCreateResponseUsesChatCompletions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("request path = %q, want %q", r.URL.Path, "/chat/completions")
		}

		var request struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			ResponseFormat json.RawMessage `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" || request.Messages[1].Role != "user" || request.Messages[1].Content != "test prompt" {
			t.Errorf("messages = %#v", request.Messages)
		}
		if len(request.ResponseFormat) != 0 {
			t.Errorf("response_format should be omitted, got %s", request.ResponseFormat)
		}

		schemaStart := strings.Index(request.Messages[0].Content, "{")
		if schemaStart < 0 {
			t.Error("system message does not contain a JSON schema")
		} else {
			var schema map[string]any
			if err := json.Unmarshal([]byte(request.Messages[0].Content[schemaStart:]), &schema); err != nil {
				t.Errorf("decode schema from system message: %v", err)
			} else if schema["additionalProperties"] != false {
				t.Error("system message schema allows additional properties")
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   "deepseek-v4-flash",
			"choices": []any{map[string]any{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": `{"title":"Chat Completion","summary":"Uses the chat endpoint.","priority":"high","tags":["chat"]}`,
					"refusal": nil,
				},
				"finish_reason": "stop",
				"logprobs":      nil,
			}},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := openai.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(server.URL),
		option.WithMaxRetries(0),
	)
	var output bytes.Buffer
	if err := createResponse(context.Background(), &client, "test prompt", &output); err != nil {
		t.Fatalf("createResponse() error = %v", err)
	}

	want := `{"title":"Chat Completion","summary":"Uses the chat endpoint.","priority":"high","tags":["chat"]}` + "\n"
	if output.String() != want {
		t.Fatalf("createResponse() output = %q, want %q", output.String(), want)
	}
}

func TestInteractiveLoop(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("\nfirst prompt\ninvalid response\nrequest error\nlast prompt\nQUIT\nignored prompt\n")
	var output bytes.Buffer
	var diagnostics bytes.Buffer
	var prompts []string
	requestErr := errors.New("request failed")

	err := interactiveLoop(
		context.Background(),
		input,
		&output,
		&diagnostics,
		func(_ context.Context, prompt string, output io.Writer) error {
			prompts = append(prompts, prompt)
			switch prompt {
			case "invalid response":
				return errInvalidResponse
			case "request error":
				return requestErr
			default:
				_, err := fmt.Fprintf(output, "response for %s\n", prompt)
				return err
			}
		},
	)
	if err != nil {
		t.Fatalf("interactiveLoop() error = %v", err)
	}

	wantPrompts := []string{"first prompt", "invalid response", "request error", "last prompt"}
	if !reflect.DeepEqual(prompts, wantPrompts) {
		t.Fatalf("interactiveLoop() prompts = %v, want %v", prompts, wantPrompts)
	}
	if got, want := output.String(), "response for first prompt\nresponse for last prompt\n"; got != want {
		t.Fatalf("interactiveLoop() output = %q, want %q", got, want)
	}
	if strings.Contains(diagnostics.String(), errInvalidResponse.Error()) {
		t.Fatal("invalid model response was written to diagnostics")
	}
	if !strings.Contains(diagnostics.String(), requestErr.Error()) {
		t.Fatal("request error was not written to diagnostics")
	}
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()

	want := Response{
		Title:    "SDK migration",
		Summary:  "Use structured output.",
		Priority: "high",
		Tags:     []string{"go", "sdk"},
	}

	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{
			name: "exact response",
			raw:  `{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":["go","sdk"]}`,
			ok:   true,
		},
		{name: "missing field", raw: `{"title":"SDK migration","summary":"Use structured output.","priority":"high"}`},
		{name: "extra field", raw: `{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[],"owner":"team"}`},
		{name: "wrong type", raw: `{"title":"SDK migration","summary":"Use structured output.","priority":1,"tags":[]}`},
		{name: "null field", raw: `{"title":null,"summary":"Use structured output.","priority":"high","tags":[]}`},
		{name: "null tag", raw: `{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[null]}`},
		{name: "trailing content", raw: `{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[]} {}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := decodeResponse(tt.raw)
			if ok != tt.ok {
				t.Fatalf("decodeResponse() ok = %v, want %v", ok, tt.ok)
			}
			if ok && !reflect.DeepEqual(got, want) {
				t.Fatalf("decodeResponse() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestResponseSchemaIsStrict(t *testing.T) {
	t.Parallel()

	if responseSchema["additionalProperties"] != false {
		t.Fatal("response schema must reject additional properties")
	}

	required, ok := responseSchema["required"].([]string)
	if !ok {
		t.Fatal("response schema required field has the wrong type")
	}
	want := []string{"title", "summary", "priority", "tags"}
	if !reflect.DeepEqual(required, want) {
		t.Fatalf("response schema required = %v, want %v", required, want)
	}
}
