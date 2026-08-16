package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestHTTPChatModelGenerate(t *testing.T) {
	for _, test := range []struct {
		name       string
		provider   string
		baseSuffix string
		wantPath   string
		response   string
	}{
		{name: "ollama", provider: "ollama", wantPath: "/api/chat", response: `{"message":{"content":"answer [S1]"}}`},
		{name: "openai compatible", provider: "openai-compatible", baseSuffix: "/v1", wantPath: "/v1/chat/completions", response: `{"choices":[{"message":{"content":"answer [S1]"}}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.wantPath {
					t.Fatalf("path = %q, want %q", request.URL.Path, test.wantPath)
				}
				if request.Method != http.MethodPost {
					t.Fatalf("method = %q", request.Method)
				}
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			model, err := newHTTPChatModel(test.provider, server.URL+test.baseSuffix, "model", "", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			message, err := model.Generate(context.Background(), []*schema.Message{{Role: schema.User, Content: "question"}})
			if err != nil {
				t.Fatal(err)
			}
			if message.Role != schema.Assistant || message.Content != "answer [S1]" {
				t.Fatalf("message = %#v", message)
			}
		})
	}
}

func TestHTTPChatModelPreservesBasePathQueryAndSendsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/gateway/v1/chat/completions" {
			t.Fatalf("path = %q, want /gateway/v1/chat/completions", request.URL.Path)
		}
		if request.URL.Query().Get("api-version") != "2024-06-01" {
			t.Fatalf("query = %q, want api-version=2024-06-01", request.URL.RawQuery)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"answer [S1]"}}]}`))
	}))
	defer server.Close()

	model, err := newHTTPChatModel("openai-compatible", server.URL+"/gateway?api-version=2024-06-01", "model", "test-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	message, err := model.Generate(context.Background(), []*schema.Message{{Role: schema.User, Content: "question"}})
	if err != nil {
		t.Fatal(err)
	}
	if message.Content != "answer [S1]" {
		t.Fatalf("message = %#v", message)
	}
}
