package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestRerankDefaultsAreDisabledAndIndependent(t *testing.T) {
	cfg, err := Load(writeConfig(t, "answer:\n  model: answer-only\n  api_key: answer-secret\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Rerank
	if r.Enabled || r.Provider != "openai-compatible" || r.BaseURL != "" || r.Model != "" || r.APIKey != "" || r.Timeout != 30*time.Second || r.MaxInputChars != 24000 || r.MaxConcurrency != 1 {
		t.Fatalf("unexpected rerank defaults: enabled=%t provider=%q model=%q timeout=%s budget=%d concurrency=%d", r.Enabled, r.Provider, r.Model, r.Timeout, r.MaxInputChars, r.MaxConcurrency)
	}
}

func TestRerankConfigurationPrecedenceAndRedaction(t *testing.T) {
	path := writeConfig(t, `
rerank:
  enabled: true
  provider: ollama
  base_url: http://file.example
  model: file-model
  api_key: file-secret
  timeout: 10s
  max_input_chars: 10000
  max_concurrency: 2
`)
	overrides := map[string]string{
		"GORAG_RERANK_ENABLED": "true", "GORAG_RERANK_PROVIDER": "openai-compatible",
		"GORAG_RERANK_BASE_URL": "https://user:url-secret@env.example/private-path?token=query-secret",
		"GORAG_RERANK_MODEL":    "env-model", "GORAG_RERANK_API_KEY": "env-secret",
		"GORAG_RERANK_TIMEOUT": "11s", "GORAG_RERANK_MAX_INPUT_CHARS": "21000", "GORAG_RERANK_MAX_CONCURRENCY": "3",
	}
	for key, value := range overrides {
		t.Setenv(key, value)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Rerank
	if !r.Enabled || r.Provider != "openai-compatible" || r.BaseURL != overrides["GORAG_RERANK_BASE_URL"] || r.Model != "env-model" || r.APIKey != "env-secret" || r.Timeout != 11*time.Second || r.MaxInputChars != 21000 || r.MaxConcurrency != 3 {
		t.Fatal("rerank environment overrides not loaded")
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("configuration", "config", cfg)
	for _, secret := range []string{"file-secret", "env-secret", "url-secret", "query-secret", "private-path"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("config log leaks %s", secret)
		}
	}
	if !strings.Contains(logs.String(), `"rerank":`) || !strings.Contains(logs.String(), `"api_key_configured":true`) {
		t.Fatal("missing safe rerank diagnostics")
	}
	t.Setenv("GORAG_RERANK_ENABLED", "false")
	cfg, err = Load(path)
	if err != nil || cfg.Rerank.Enabled {
		t.Fatalf("disable override failed: %v", err)
	}
}

func TestRerankValidatesOnlyWhenEnabled(t *testing.T) {
	for _, test := range []struct {
		field       string
		breakConfig func(*RerankConfig)
	}{
		{"provider", func(c *RerankConfig) { c.Provider = "bad" }},
		{"base_url", func(c *RerankConfig) { c.BaseURL = "not-absolute" }},
		{"base_url", func(c *RerankConfig) { c.BaseURL = "ftp://example.org" }},
		{"base_url", func(c *RerankConfig) { c.BaseURL = "" }},
		{"model", func(c *RerankConfig) { c.Model = " " }},
		{"timeout", func(c *RerankConfig) { c.Timeout = 0 }},
		{"max_input_chars", func(c *RerankConfig) { c.MaxInputChars = -1 }},
		{"max_concurrency", func(c *RerankConfig) { c.MaxConcurrency = 0 }},
	} {
		t.Run(test.field, func(t *testing.T) {
			cfg := validTestConfig()
			cfg.Rerank = RerankConfig{Enabled: true, Provider: "openai-compatible", BaseURL: "https://rerank.example/v1", Model: "reranker", Timeout: 30 * time.Second, MaxInputChars: 24000, MaxConcurrency: 1}
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
			test.breakConfig(&cfg.Rerank)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "rerank."+test.field) {
				t.Fatalf("missing validation: %v", err)
			}
			cfg.Rerank.Enabled = false
			if err := cfg.Validate(); err != nil {
				t.Fatalf("disabled settings validated: %v", err)
			}
		})
	}
}

func TestRerankUnknownFieldRejected(t *testing.T) {
	if _, err := Load(writeConfig(t, "rerank:\n  max_input_char: 100\n")); err == nil {
		t.Fatal("unknown rerank key accepted")
	}
}
