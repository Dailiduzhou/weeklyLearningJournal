package httpurl

import (
	"net/url"
	"testing"
)

func TestJoinPathPreservesBasePathAndQuery(t *testing.T) {
	base, err := url.Parse("https://example.com/ollama?token=secret#fragment")
	if err != nil {
		t.Fatal(err)
	}
	got := JoinPath(base, "api/tags")
	want := "https://example.com/ollama/api/tags?token=secret"
	if got.String() != want {
		t.Fatalf("JoinPath() = %q, want %q", got.String(), want)
	}
}

func TestOpenAICompatibleHonorsV1BasePath(t *testing.T) {
	for _, test := range []struct {
		base string
		want string
	}{
		{base: "https://api.example.com", want: "https://api.example.com/v1/chat/completions"},
		{base: "https://api.example.com/", want: "https://api.example.com/v1/chat/completions"},
		{base: "https://api.example.com/v1", want: "https://api.example.com/v1/chat/completions"},
		{base: "https://api.example.com/gateway/v1/", want: "https://api.example.com/gateway/v1/chat/completions"},
	} {
		baseURL, err := url.Parse(test.base)
		if err != nil {
			t.Fatal(err)
		}
		got := OpenAICompatible(baseURL, "chat/completions")
		if got.String() != test.want {
			t.Fatalf("OpenAICompatible(%q) = %q, want %q", test.base, got.String(), test.want)
		}
	}
}
