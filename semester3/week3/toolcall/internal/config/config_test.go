package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentOverridesFileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("model:\n  api_key: file-key\n  name: file-model\nagent:\n  max_rounds: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_MODEL__API_KEY", "env-key")
	t.Setenv("AGENT_MODEL__NAME", "env-model")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model.APIKey != "env-key" || cfg.Model.Name != "env-model" || cfg.Agent.MaxRounds != 3 || cfg.Agent.ToolTimeout <= 0 {
		t.Fatalf("unexpected precedence result: %+v", cfg)
	}
}

func TestDotenvProvidesOpenAIConfiguration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("model:\n  name: test-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("OPENAI_API_KEY=dotenv-key\nOPENAI_BASE_URL=https://openai.example/v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model.APIKey != "dotenv-key" || cfg.Model.BaseURL != "https://openai.example/v1" {
		t.Fatalf("dotenv configuration was not loaded: %+v", cfg.Model)
	}
}

func TestProcessEnvironmentOverridesDotenv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("model:\n  name: test-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("OPENAI_API_KEY=dotenv-key\nOPENAI_BASE_URL=https://dotenv.example/v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("OPENAI_API_KEY", "process-key")
	t.Setenv("OPENAI_BASE_URL", "https://process.example/v1")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model.APIKey != "process-key" || cfg.Model.BaseURL != "https://process.example/v1" {
		t.Fatalf("process environment did not take precedence: %+v", cfg.Model)
	}
}

func TestDatabaseRequiresDSNWhenEnabled(t *testing.T) {
	t.Setenv("AGENT_MODEL__API_KEY", "test-key")
	t.Setenv("AGENT_DATABASE__ENABLED", "true")
	if _, err := Load(""); err == nil {
		t.Fatal("expected missing database DSN to fail validation")
	}
}

func TestDotenvProvidesDatabaseDSN(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("model:\n  name: test-model\n  api_key: test-key\ndatabase:\n  enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("AGENT_DATABASE__DSN=postgres://agent_readonly:secret@localhost:5432/agent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("AGENT_DATABASE__DSN", "")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://agent_readonly:secret@localhost:5432/agent" {
		t.Fatalf("dotenv DSN was not loaded: %+v", cfg.Database)
	}
}

func TestProcessEnvironmentOverridesDotenvDSN(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("model:\n  name: test-model\n  api_key: test-key\ndatabase:\n  enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("AGENT_DATABASE__DSN=postgres://dotenv.example/agent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("AGENT_DATABASE__DSN", "postgres://process.example/agent")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://process.example/agent" {
		t.Fatalf("process environment did not take precedence: %+v", cfg.Database)
	}
}
