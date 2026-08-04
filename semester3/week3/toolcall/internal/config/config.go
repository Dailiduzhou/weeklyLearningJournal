package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Model     ModelConfig     `mapstructure:"model"`
	Agent     AgentConfig     `mapstructure:"agent"`
	Documents DocumentsConfig `mapstructure:"documents"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Audit     AuditConfig     `mapstructure:"audit"`
}

type ModelConfig struct {
	APIKey     string        `mapstructure:"api_key"`
	BaseURL    string        `mapstructure:"base_url"`
	Name       string        `mapstructure:"name"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

type AgentConfig struct {
	MaxRounds           int           `mapstructure:"max_rounds"`
	TaskTimeout         time.Duration `mapstructure:"task_timeout"`
	ToolTimeout         time.Duration `mapstructure:"tool_timeout"`
	MaxToolResultBytes  int           `mapstructure:"max_tool_result_bytes"`
	MaxHistoryBytes     int           `mapstructure:"max_history_bytes"`
	MaxRepeatedFailures int           `mapstructure:"max_repeated_failures"`
	MaxUnknownTools     int           `mapstructure:"max_unknown_tools"`
}

type DocumentsConfig struct {
	Directory  string `mapstructure:"directory"`
	ChunkRunes int    `mapstructure:"chunk_runes"`
	MaxResults int    `mapstructure:"max_results"`
}

type DatabaseConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	DSN          string        `mapstructure:"dsn"`
	QueryTimeout time.Duration `mapstructure:"query_timeout"`
	MaxRows      int           `mapstructure:"max_rows"`
	MaxBytes     int           `mapstructure:"max_bytes"`
	Queries      []QueryConfig `mapstructure:"queries"`
}

type QueryConfig struct {
	Name        string             `mapstructure:"name"`
	Description string             `mapstructure:"description"`
	SQL         string             `mapstructure:"sql"`
	Params      []QueryParamConfig `mapstructure:"params"`
}

type QueryParamConfig struct {
	Name      string   `mapstructure:"name"`
	Type      string   `mapstructure:"type"`
	Required  bool     `mapstructure:"required"`
	MaxLength int      `mapstructure:"max_length"`
	Minimum   *float64 `mapstructure:"minimum"`
	Maximum   *float64 `mapstructure:"maximum"`
}

type AuditConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	defaults(v)
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) {
				return Config{}, fmt.Errorf("read config: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	// Compatibility with the conventional OpenAI variable names.
	if cfg.Model.APIKey == "" {
		cfg.Model.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if cfg.Model.BaseURL == "" {
		cfg.Model.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func defaults(v *viper.Viper) {
	v.SetDefault("model.name", "gpt-5.6-terra")
	v.SetDefault("model.api_key", "")
	v.SetDefault("model.base_url", "")
	v.SetDefault("model.timeout", "30s")
	v.SetDefault("model.max_retries", 2)
	v.SetDefault("agent.max_rounds", 8)
	v.SetDefault("agent.task_timeout", "2m")
	v.SetDefault("agent.tool_timeout", "10s")
	v.SetDefault("agent.max_tool_result_bytes", 32768)
	v.SetDefault("agent.max_history_bytes", 262144)
	v.SetDefault("agent.max_repeated_failures", 3)
	v.SetDefault("agent.max_unknown_tools", 3)
	v.SetDefault("documents.directory", "./docs")
	v.SetDefault("documents.chunk_runes", 1200)
	v.SetDefault("documents.max_results", 5)
	v.SetDefault("database.enabled", false)
	v.SetDefault("database.dsn", "")
	v.SetDefault("database.query_timeout", "3s")
	v.SetDefault("database.max_rows", 100)
	v.SetDefault("database.max_bytes", 65536)
	v.SetDefault("audit.level", "info")
}

func (c Config) Validate() error {
	if c.Model.APIKey == "" {
		return errors.New("model API key is required (AGENT_MODEL__API_KEY or OPENAI_API_KEY)")
	}
	if c.Model.Name == "" || c.Model.Timeout <= 0 || c.Model.MaxRetries < 0 {
		return errors.New("invalid model configuration")
	}
	if c.Agent.MaxRounds <= 0 || c.Agent.TaskTimeout <= 0 || c.Agent.ToolTimeout <= 0 ||
		c.Agent.MaxToolResultBytes < 256 || c.Agent.MaxHistoryBytes < 1024 ||
		c.Agent.MaxRepeatedFailures <= 0 || c.Agent.MaxUnknownTools <= 0 {
		return errors.New("invalid agent limits")
	}
	if c.Documents.ChunkRunes <= 0 || c.Documents.MaxResults <= 0 {
		return errors.New("invalid document search limits")
	}
	if c.Database.Enabled {
		if c.Database.DSN == "" {
			return errors.New("database DSN is required when database is enabled")
		}
		if c.Database.QueryTimeout <= 0 || c.Database.MaxRows <= 0 || c.Database.MaxBytes < 256 {
			return errors.New("invalid database limits")
		}
		seen := make(map[string]struct{}, len(c.Database.Queries))
		for _, q := range c.Database.Queries {
			if q.Name == "" || q.SQL == "" {
				return errors.New("database query name and SQL are required")
			}
			if _, ok := seen[q.Name]; ok {
				return fmt.Errorf("duplicate database query %q", q.Name)
			}
			seen[q.Name] = struct{}{}
			paramSeen := make(map[string]struct{}, len(q.Params))
			for _, p := range q.Params {
				if p.Name == "" || (p.Type != "string" && p.Type != "integer" && p.Type != "number" && p.Type != "boolean") {
					return fmt.Errorf("query %q has invalid parameter %q", q.Name, p.Name)
				}
				if _, ok := paramSeen[p.Name]; ok {
					return fmt.Errorf("query %q has duplicate parameter %q", q.Name, p.Name)
				}
				paramSeen[p.Name] = struct{}{}
				if p.MaxLength < 0 || (p.Minimum != nil && p.Maximum != nil && *p.Minimum > *p.Maximum) {
					return fmt.Errorf("query %q has invalid limits for parameter %q", q.Name, p.Name)
				}
			}
		}
	}
	return nil
}
