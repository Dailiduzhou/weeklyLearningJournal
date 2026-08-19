package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// These tests verify Viper precedence explicitly and must not be affected
	// by GORAG_* variables exported in the developer's shell.
	saved := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.HasPrefix(key, "GORAG_") {
			saved[key] = value
			_ = os.Unsetenv(key)
		}
	}
	code := m.Run()
	for key, value := range saved {
		_ = os.Setenv(key, value)
	}
	os.Exit(code)
}

func TestLoadUsesFileAndEnvironmentPrecedence(t *testing.T) {
	path := writeConfig(t, `
server:
  address: ":9090"
database:
  url: "postgres://file-user:file-password@db.example:5432/gorag?sslmode=disable"
retrieval:
  top_k: 7
  similarity_threshold: 0.7
embedding:
  batch_size: 24
indexing:
  document_concurrency: 3
`)
	t.Setenv("GORAG_SERVER_ADDRESS", ":9191")
	t.Setenv("GORAG_DATABASE_URL", "postgres://env-user:env-password@db.example:5432/gorag?sslmode=disable")
	t.Setenv("GORAG_RETRIEVAL_TOP_K", "9")
	t.Setenv("GORAG_EMBEDDING_MAX_CONCURRENCY", "4")
	t.Setenv("GORAG_ANSWER_API_KEY", "secret-from-environment")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Address != ":9191" {
		t.Fatalf("Server.Address = %q, want environment override", cfg.Server.Address)
	}
	if cfg.Database.URL != "postgres://env-user:env-password@db.example:5432/gorag?sslmode=disable" {
		t.Fatalf("Database.URL = %q, want environment override", cfg.Database.URL)
	}
	if cfg.Retrieval.TopK != 9 {
		t.Fatalf("Retrieval.TopK = %d, want 9", cfg.Retrieval.TopK)
	}
	if cfg.Retrieval.SimilarityThreshold != 0.7 {
		t.Fatalf("Retrieval.SimilarityThreshold = %v, want file value", cfg.Retrieval.SimilarityThreshold)
	}
	if cfg.Answer.APIKey != "secret-from-environment" {
		t.Fatal("Answer.APIKey did not use the environment override")
	}
	if cfg.Embedding.Dimension != EmbeddingDimension {
		t.Fatalf("Embedding.Dimension = %d, want default %d", cfg.Embedding.Dimension, EmbeddingDimension)
	}
	if cfg.Embedding.BatchSize != 24 || cfg.Embedding.MaxConcurrency != 4 {
		t.Fatalf("Embedding batching = %#v, want file batch size and environment concurrency", cfg.Embedding)
	}
	if cfg.Indexing.DocumentConcurrency != 3 {
		t.Fatalf("Indexing.DocumentConcurrency = %d, want 3", cfg.Indexing.DocumentConcurrency)
	}
	if cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Fatalf("Server.ShutdownTimeout = %s, want default 15s", cfg.Server.ShutdownTimeout)
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil || !strings.Contains(err.Error(), "read configuration") {
		t.Fatalf("Load() error = %v, want missing configuration error", err)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := writeConfig(t, "database:\n  url: postgres://gorag:gorag@localhost:5432/gorag\nunexpected: true\n")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "decode configuration") {
		t.Fatalf("Load() error = %v, want strict decode error", err)
	}
}

func TestLoadRejectsEmbeddingModelField(t *testing.T) {
	path := writeConfig(t, "embedding:\n  model: some-other-model\n  dimension: 1024\n")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "decode configuration") {
		t.Fatalf("Load() error = %v, want embedding.model to be rejected as an unknown field", err)
	}
}

func TestLoadRejectsConflictingDimension(t *testing.T) {
	path := writeConfig(t, "embedding:\n  dimension: 768\n")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "embedding.dimension must be 1024") {
		t.Fatalf("Load() error = %v, want dimension invariant error", err)
	}
}

func TestLoadRejectsNonPositiveBatchingConfiguration(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		contents string
		want     string
	}{
		{name: "batch size", contents: "embedding:\n  batch_size: -1\n", want: "embedding.batch_size must be positive"},
		{name: "embedding concurrency", contents: "embedding:\n  max_concurrency: -1\n", want: "embedding.max_concurrency must be positive"},
		{name: "document concurrency", contents: "indexing:\n  document_concurrency: -1\n", want: "indexing.document_concurrency must be positive"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, testCase.contents))
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Load() error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func TestValidateReturnsAllRelevantProblems(t *testing.T) {
	cfg := Config{
		Server:    ServerConfig{ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second},
		Database:  DatabaseConfig{URL: "http://not-postgres.example"},
		Ollama:    OllamaConfig{BaseURL: "localhost:11434", Timeout: time.Second},
		Embedding: EmbeddingConfig{Dimension: EmbeddingDimension, BatchSize: 16, MaxConcurrency: 1},
		Indexing:  IndexingConfig{DocumentConcurrency: 1},
		Retrieval: RetrievalConfig{TopK: 101, SimilarityThreshold: 2},
		Answer:    AnswerConfig{Provider: "unknown", BaseURL: "http://localhost:11434", Timeout: time.Second},
		Startup:   StartupConfig{CheckTimeout: time.Second, RetryInterval: 2 * time.Second},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation problems")
	}
	for _, want := range []string{
		"server.address",
		"database.url",
		"ollama.base_url",
		"documents.dir",
		"retrieval.top_k",
		"retrieval.similarity_threshold",
		"answer.provider",
		"answer.model",
		"startup.retry_interval",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() error = %q, want it to contain %q", err, want)
		}
	}
}

func TestConfigSlogValueRedactsSecrets(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{URL: "postgres://gorag:database-secret@localhost:5432/gorag?password=query-secret&sslmode=disable"},
		Answer:   AnswerConfig{APIKey: "model-api-secret"},
	}
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	logger.Info("loaded", "config", cfg)

	got := output.String()
	for _, secret := range []string{"database-secret", "query-secret", "model-api-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("structured config log leaked %q: %s", secret, got)
		}
	}
	for _, want := range []string{"REDACTED", "api_key_configured=true"} {
		if !strings.Contains(got, want) {
			t.Errorf("structured config log = %q, want %q", got, want)
		}
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
