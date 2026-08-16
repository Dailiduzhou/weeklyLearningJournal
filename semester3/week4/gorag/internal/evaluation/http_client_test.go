package evaluation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientAsk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q, want application/json", request.Header.Get("Content-Type"))
		}
		var input struct {
			Question string `json:"question"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if input.Question != "Redis?" {
			t.Errorf("question = %q, want Redis?", input.Question)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"answerable":true,"answer":"内存存储 [S1]","sources":[{"id":"S1","source_path":"Redis.md","document_title":"Redis","heading_path":["Redis"],"start_line":10,"end_line":12,"similarity":0.91}]}`))
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL+"/api/v1/questions", server.Client())
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}
	response, err := client.Ask(context.Background(), "Redis?")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if !response.Answerable || response.Sources[0].SourcePath != "Redis.md" || response.Sources[0].HeadingPath[0] != "Redis" {
		t.Fatalf("Ask() response = %#v", response)
	}
}

func TestHTTPClientRejectsBadResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{name: "status", status: http.StatusServiceUnavailable, body: `{}`, want: "status 503"},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, want: "decode"},
		{name: "trailing JSON", status: http.StatusOK, body: `{}` + "\n" + `{}`, want: "multiple JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client, _ := NewHTTPClient(server.URL, server.Client())
			_, err := client.Ask(context.Background(), "q")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Ask() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestNewHTTPClientRejectsRelativeEndpoint(t *testing.T) {
	if _, err := NewHTTPClient("/api/v1/questions", nil); err == nil {
		t.Fatal("NewHTTPClient() error = nil, want invalid endpoint")
	}
}
