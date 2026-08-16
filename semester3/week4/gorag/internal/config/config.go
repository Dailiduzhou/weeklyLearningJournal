// Package config loads and validates the process configuration shared by the
// server and indexer binaries.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultPath is used when Load receives an empty path.
	DefaultPath = "config.yaml"
	// EnvironmentPrefix namespaces every environment override.
	EnvironmentPrefix = "GORAG"
	// EmbeddingDimension is a system-wide invariant shared by indexing and query
	// embeddings and by the database vector column.
	EmbeddingDimension = 1024
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Ollama    OllamaConfig    `mapstructure:"ollama"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Documents DocumentsConfig `mapstructure:"documents"`
	Retrieval RetrievalConfig `mapstructure:"retrieval"`
	Answer    AnswerConfig    `mapstructure:"answer"`
	Startup   StartupConfig   `mapstructure:"startup"`
}

type ServerConfig struct {
	Address           string        `mapstructure:"address"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type OllamaConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// EmbeddingConfig only exposes the vector-dimension invariant. The embedding
// model name is intentionally not configurable: document and query embeddings
// must stay on the fixed qwen3-embedding:0.6b model, and repository/migration
// constraints enforce that invariant.
type EmbeddingConfig struct {
	Dimension int `mapstructure:"dimension"`
}

type DocumentsConfig struct {
	Dir string `mapstructure:"dir"`
}

type RetrievalConfig struct {
	TopK                int     `mapstructure:"top_k"`
	MaxContext          int     `mapstructure:"max_context"`
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
}

type AnswerConfig struct {
	Provider string        `mapstructure:"provider"`
	BaseURL  string        `mapstructure:"base_url"`
	Model    string        `mapstructure:"model"`
	APIKey   string        `mapstructure:"api_key"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

type StartupConfig struct {
	CheckTimeout  time.Duration `mapstructure:"check_timeout"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
}

// LogValue makes slog.Any("config", cfg) safe by construction. Credentials
// are intentionally omitted or redacted while operational settings remain
// available for diagnostics.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("server", c.Server),
		slog.String("database_url", redactURL(c.Database.URL)),
		slog.Any("ollama", c.Ollama),
		slog.Any("embedding", c.Embedding),
		slog.Any("documents", c.Documents),
		slog.Any("retrieval", c.Retrieval),
		slog.Group("answer",
			slog.String("provider", c.Answer.Provider),
			slog.String("base_url", c.Answer.BaseURL),
			slog.String("model", c.Answer.Model),
			slog.Duration("timeout", c.Answer.Timeout),
			slog.Bool("api_key_configured", c.Answer.APIKey != ""),
		),
		slog.Any("startup", c.Startup),
	)
}

// Load reads YAML configuration, applies GORAG_-prefixed environment
// overrides, and validates the resulting configuration. Precedence is:
// environment variables, configuration file, then built-in defaults.
//
// Passing an empty path reads DefaultPath. Passing a non-empty path makes that
// exact file authoritative; a missing or unreadable file is an error.
func Load(path string) (Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix(EnvironmentPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	for _, key := range configurationKeys {
		if err := v.BindEnv(key); err != nil {
			return Config{}, fmt.Errorf("bind environment override for %q: %w", key, err)
		}
	}

	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read configuration %q: %w", path, err)
	}

	var cfg Config
	if err := v.UnmarshalExact(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode configuration %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}
	return cfg, nil
}

// Validate checks process-independent invariants. Individual commands may add
// narrower checks for the dependencies they actually construct.
func (c Config) Validate() error {
	var problems []error

	if strings.TrimSpace(c.Server.Address) == "" {
		problems = append(problems, errors.New("server.address must not be empty"))
	}
	positiveDuration(&problems, "server.read_header_timeout", c.Server.ReadHeaderTimeout)
	positiveDuration(&problems, "server.shutdown_timeout", c.Server.ShutdownTimeout)

	if err := validateURL("database.url", c.Database.URL, "postgres", "postgresql"); err != nil {
		problems = append(problems, err)
	}
	if err := validateURL("ollama.base_url", c.Ollama.BaseURL, "http", "https"); err != nil {
		problems = append(problems, err)
	}
	positiveDuration(&problems, "ollama.timeout", c.Ollama.Timeout)

	if c.Embedding.Dimension != EmbeddingDimension {
		problems = append(problems, fmt.Errorf("embedding.dimension must be %d, got %d", EmbeddingDimension, c.Embedding.Dimension))
	}
	if strings.TrimSpace(c.Documents.Dir) == "" {
		problems = append(problems, errors.New("documents.dir must not be empty"))
	}
	if c.Retrieval.TopK <= 0 || c.Retrieval.TopK > 100 {
		problems = append(problems, fmt.Errorf("retrieval.top_k must be between 1 and 100, got %d", c.Retrieval.TopK))
	}
	if c.Retrieval.MaxContext <= 0 || c.Retrieval.MaxContext > c.Retrieval.TopK {
		problems = append(problems, fmt.Errorf("retrieval.max_context must be between 1 and retrieval.top_k (%d), got %d", c.Retrieval.TopK, c.Retrieval.MaxContext))
	}
	if c.Retrieval.SimilarityThreshold < -1 || c.Retrieval.SimilarityThreshold > 1 {
		problems = append(problems, fmt.Errorf("retrieval.similarity_threshold must be between -1 and 1, got %v", c.Retrieval.SimilarityThreshold))
	}

	switch c.Answer.Provider {
	case "ollama", "openai-compatible":
	case "":
		problems = append(problems, errors.New("answer.provider must not be empty"))
	default:
		problems = append(problems, fmt.Errorf("answer.provider must be ollama or openai-compatible, got %q", c.Answer.Provider))
	}
	if err := validateURL("answer.base_url", c.Answer.BaseURL, "http", "https"); err != nil {
		problems = append(problems, err)
	}
	if strings.TrimSpace(c.Answer.Model) == "" {
		problems = append(problems, errors.New("answer.model must not be empty"))
	}
	positiveDuration(&problems, "answer.timeout", c.Answer.Timeout)
	positiveDuration(&problems, "startup.check_timeout", c.Startup.CheckTimeout)
	positiveDuration(&problems, "startup.retry_interval", c.Startup.RetryInterval)
	if c.Startup.RetryInterval > c.Startup.CheckTimeout {
		problems = append(problems, errors.New("startup.retry_interval must not exceed startup.check_timeout"))
	}

	return errors.Join(problems...)
}

func validateURL(name, raw string, schemes ...string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	for _, scheme := range schemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return fmt.Errorf("%s must use one of schemes %s", name, strings.Join(schemes, ", "))
}

func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<invalid>"
	}
	// Query parameters are omitted as database clients may accept credentials
	// there as well as in URL userinfo.
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	if parsed.User != nil {
		if _, set := parsed.User.Password(); set {
			parsed.User = url.UserPassword(parsed.User.Username(), "REDACTED")
		} else {
			parsed.User = url.User(parsed.User.Username())
		}
	}
	return parsed.String()
}

func positiveDuration(problems *[]error, name string, value time.Duration) {
	if value <= 0 {
		*problems = append(*problems, fmt.Errorf("%s must be positive, got %s", name, value))
	}
}

var configurationKeys = []string{
	"server.address",
	"server.read_header_timeout",
	"server.shutdown_timeout",
	"database.url",
	"ollama.base_url",
	"ollama.timeout",
	"embedding.dimension",
	"documents.dir",
	"retrieval.top_k",
	"retrieval.max_context",
	"retrieval.similarity_threshold",
	"answer.provider",
	"answer.base_url",
	"answer.model",
	"answer.api_key",
	"answer.timeout",
	"startup.check_timeout",
	"startup.retry_interval",
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.read_header_timeout", "5s")
	v.SetDefault("server.shutdown_timeout", "15s")
	v.SetDefault("database.url", "postgres://gorag:gorag@localhost:5432/gorag?sslmode=disable")
	v.SetDefault("ollama.base_url", "http://localhost:11434")
	v.SetDefault("ollama.timeout", "30s")
	v.SetDefault("embedding.dimension", EmbeddingDimension)
	v.SetDefault("documents.dir", "./docs")
	v.SetDefault("retrieval.top_k", 10)
	v.SetDefault("retrieval.max_context", 5)
	v.SetDefault("retrieval.similarity_threshold", 0.5)
	v.SetDefault("answer.provider", "ollama")
	v.SetDefault("answer.base_url", "http://localhost:11434")
	v.SetDefault("answer.model", "qwen3:4b")
	v.SetDefault("answer.api_key", "")
	v.SetDefault("answer.timeout", "60s")
	v.SetDefault("startup.check_timeout", "30s")
	v.SetDefault("startup.retry_interval", "1s")
}
