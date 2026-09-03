package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsEnableVectorAndDisableBM25(t *testing.T) {
	path := writeConfig(t, "database:\n  url: \"postgres://u:p@db.example:5432/gorag\"\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Retrieval.Vector.Enabled {
		t.Fatal("Retrieval.Vector.Enabled should default to true")
	}
	if cfg.Retrieval.BM25.Enabled {
		t.Fatal("Retrieval.BM25.Enabled should default to false")
	}
	if cfg.Retrieval.BM25.IndexPath != "./data/bm25" {
		t.Fatalf("Retrieval.BM25.IndexPath = %q, want default", cfg.Retrieval.BM25.IndexPath)
	}
	if cfg.Retrieval.BM25.MinScore != 0 {
		t.Fatalf("Retrieval.BM25.MinScore = %v, want 0", cfg.Retrieval.BM25.MinScore)
	}
}

func TestLoadReadsRetrievalTogglesFromFileAndEnvironment(t *testing.T) {
	path := writeConfig(t, `
retrieval:
  vector:
    enabled: true
  bm25:
    enabled: true
    index_path: "/var/lib/gorag/bm25"
    min_score: 0.4
`)
	t.Setenv("GORAG_RETRIEVAL_BM25_INDEX_PATH", "/env/bm25")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Retrieval.BM25.Enabled || cfg.Retrieval.BM25.MinScore != 0.4 {
		t.Fatalf("BM25 config = %#v, want enabled with min score 0.4", cfg.Retrieval.BM25)
	}
	if cfg.Retrieval.BM25.IndexPath != "/env/bm25" {
		t.Fatalf("BM25 IndexPath = %q, want environment override", cfg.Retrieval.BM25.IndexPath)
	}
	if !cfg.Retrieval.Vector.Enabled {
		t.Fatal("Vector toggle from file was not read")
	}
}

func TestValidateRejectsDisablingBothRetrievers(t *testing.T) {
	cfg := validTestConfig()
	cfg.Retrieval.Vector.Enabled = false
	cfg.Retrieval.BM25.Enabled = false
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "at least one of retrieval.vector.enabled") {
		t.Fatalf("Validate() error = %v, want both-disabled rejection", err)
	}
}

func TestValidateRejectsBM25WithoutIndexPath(t *testing.T) {
	cfg := validTestConfig()
	cfg.Retrieval.BM25.Enabled = true
	cfg.Retrieval.BM25.IndexPath = " "
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "retrieval.bm25.index_path") {
		t.Fatalf("Validate() error = %v, want empty index_path rejection", err)
	}
}

func TestValidateRejectsNegativeBM25MinScore(t *testing.T) {
	cfg := validTestConfig()
	cfg.Retrieval.BM25.Enabled = true
	cfg.Retrieval.BM25.MinScore = -0.5
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "retrieval.bm25.min_score") {
		t.Fatalf("Validate() error = %v, want negative min_score rejection", err)
	}
}

func TestValidateAcceptsBM25Only(t *testing.T) {
	cfg := validTestConfig()
	cfg.Retrieval.Vector.Enabled = false
	cfg.Retrieval.BM25.Enabled = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

// validTestConfig returns a minimal configuration that passes Validate.
func validTestConfig() Config {
	return Config{
		Server:    ServerConfig{Address: ":8080", ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second},
		Database:  DatabaseConfig{URL: "postgres://u:p@db.example:5432/gorag"},
		Ollama:    OllamaConfig{BaseURL: "http://localhost:11434", Timeout: time.Second},
		Embedding: EmbeddingConfig{Dimension: EmbeddingDimension, BatchSize: 16, MaxConcurrency: 1},
		Indexing:  IndexingConfig{DocumentConcurrency: 1},
		Documents: DocumentsConfig{Dir: "./docs"},
		Retrieval: RetrievalConfig{TopK: 10, MaxContext: 5, SimilarityThreshold: 0.5, Vector: VectorConfig{Enabled: true}, BM25: BM25Config{IndexPath: "./data/bm25"}},
		Answer:    AnswerConfig{Provider: "ollama", BaseURL: "http://localhost:11434", Model: "m", Timeout: time.Second},
		Startup:   StartupConfig{CheckTimeout: time.Second, RetryInterval: time.Second},
	}
}
