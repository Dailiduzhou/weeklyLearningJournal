package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"gorag/internal/config"
	"gorag/internal/retriever"
)

type serverRerankChild struct{ docs []*schema.Document }

func (c *serverRerankChild) Retrieve(context.Context, string, ...einoretriever.Option) ([]*schema.Document, error) {
	return c.docs, nil
}

func TestRerankWiringDisabledReturnsOriginalEvenWithInvalidSettings(t *testing.T) {
	child := &serverRerankChild{}
	got, err := buildRerankRetriever(config.RerankConfig{Provider: "bad", BaseURL: "://invalid", Timeout: -1}, child, nil)
	if err != nil || got != child {
		t.Fatalf("disabled rerank changed retriever: %v", err)
	}
}

func TestRerankWiringIndependentHTTPConfigAndNoStartupProbe(t *testing.T) {
	for _, provider := range []string{"openai-compatible", "ollama"} {
		t.Run(provider, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				wantPath := "/gateway/v1/chat/completions"
				if provider == "ollama" {
					wantPath = "/gateway/api/chat"
				}
				if request.URL.Path != wantPath || request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer rerank-key" || request.URL.Query().Get("api-version") != "test" {
					t.Error("wrong rerank endpoint/auth")
				}
				var body struct {
					Model    string `json:"model"`
					Messages []struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"messages"`
					Stream bool `json:"stream"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Model != "independent-reranker" || body.Stream || len(body.Messages) != 2 {
					t.Error("wrong rerank body")
				}
				if provider == "ollama" {
					_, _ = w.Write([]byte(`{"message":{"content":"{\"ranking\":[\"C2\",\"C1\"]}"}}`))
				} else {
					_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ranking\":[\"C2\",\"C1\"]}"}}]}`))
				}
			}))
			defer server.Close()
			child := &serverRerankChild{docs: []*schema.Document{{ID: "first", Content: "first"}, {ID: "second", Content: "second"}}}
			cfg := config.RerankConfig{Enabled: true, Provider: provider, BaseURL: server.URL + "/gateway?api-version=test", Model: "independent-reranker", APIKey: "rerank-key", Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}
			online, err := buildRerankRetriever(cfg, child, nil)
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 0 {
				t.Fatal("optional dependency probed during construction")
			}
			got, err := online.Retrieve(context.Background(), "query")
			if err != nil || calls.Load() != 1 || len(got) != 2 || got[0].ID != "second" {
				t.Fatalf("ranked HTTP response not applied: %v %v", got, err)
			}
			if rank, ok := retriever.RerankPosition(got[0]); !ok || rank != 1 {
				t.Fatal("missing internal rank")
			}
		})
	}
}

func TestRerankHTTPFailuresFallBackWithoutRetry(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusOK} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not JSON"}}]}`))
		}))
		child := &serverRerankChild{docs: []*schema.Document{{ID: "first", Content: "first"}, {ID: "second", Content: "second"}}}
		online, err := buildRerankRetriever(config.RerankConfig{Enabled: true, Provider: "openai-compatible", BaseURL: server.URL, Model: "reranker", Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}, child, nil)
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		got, err := online.Retrieve(context.Background(), "query")
		server.Close()
		if err != nil || calls.Load() != 1 || len(got) != 2 || got[0] != child.docs[0] || got[1] != child.docs[1] {
			t.Fatalf("HTTP %d did not fall back unchanged: %v", status, err)
		}
	}
}

func TestRerankUnavailableDependencyDoesNotPreventConstruction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("unexpected probe") }))
	server.Close()
	child := &serverRerankChild{docs: []*schema.Document{{ID: "first", Content: "first"}, {ID: "second", Content: "second"}}}
	online, err := buildRerankRetriever(config.RerankConfig{Enabled: true, Provider: "openai-compatible", BaseURL: server.URL, Model: "reranker", Timeout: time.Second, MaxInputChars: 24000, MaxConcurrency: 1}, child, nil)
	if err != nil {
		t.Fatalf("unavailable optional dependency blocked construction: %v", err)
	}
	got, err := online.Retrieve(context.Background(), "query")
	if err != nil || len(got) != 2 || got[0] != child.docs[0] {
		t.Fatalf("unavailable dependency did not fall back: %v", err)
	}
}
